package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/order"
)

// Postgres is the admin repository.
type Postgres struct {
	db *pgxpool.Pool
}

// NewPostgres creates the admin repository.
func NewPostgres(db *pgxpool.Pool) *Postgres {
	return &Postgres{db: db}
}

// querier is satisfied by both the pool and a transaction.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// mapErr turns driver errors into application errors.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			if strings.Contains(pgErr.ConstraintName, "code") {
				return apperr.Field("code", apperr.Taken)
			}
			return apperr.Field("slug", apperr.Taken)
		case "23503": // foreign_key_violation
			if strings.Contains(pgErr.Detail, "is not present") {
				return apperr.Field(fkField(pgErr.ConstraintName), apperr.Invalid)
			}
			return fmt.Errorf("%w: still referenced", apperr.ErrConflict)
		}
	}
	return err
}

func fkField(constraint string) string {
	switch {
	case strings.Contains(constraint, "brand_id"):
		return "brandId"
	case strings.Contains(constraint, "line_id"):
		return "lineId"
	case strings.Contains(constraint, "category_id"):
		return "categoryId"
	default:
		return "id"
	}
}

func affected(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// uniqueSlug appends -2, -3, ... to a generated slug until it is free.
// table and scope are constants from this package, never user input.
func uniqueSlug(ctx context.Context, q querier, table, scope, base string, excludeID int64, scopeArg any) (string, error) {
	for i := 1; i <= 100; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		sql := `SELECT EXISTS (SELECT 1 FROM ` + table + ` WHERE slug = $1 AND id <> $2` + scope + `)`
		args := []any{candidate, excludeID}
		if scope != "" {
			args = append(args, scopeArg)
		}
		var taken bool
		if err := q.QueryRow(ctx, sql, args...).Scan(&taken); err != nil {
			return "", fmt.Errorf("check slug: %w", err)
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", apperr.Field("slug", apperr.Taken)
}

// ---- Brands and lines ----

// Brands returns all brands with their lines, active or not.
func (p *Postgres) Brands(ctx context.Context) ([]Brand, error) {
	rows, err := p.db.Query(ctx,
		`SELECT id, slug, name, description, logo_url, sort, is_active FROM brands ORDER BY sort, name`)
	if err != nil {
		return nil, fmt.Errorf("query brands: %w", err)
	}
	brands, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Brand, error) {
		var b Brand
		err := r.Scan(&b.ID, &b.Slug, &b.Name, &b.Description, &b.LogoURL, &b.Sort, &b.IsActive)
		b.Lines = []Line{}
		return b, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect brands: %w", err)
	}

	lineRows, err := p.db.Query(ctx, `
		SELECT id, brand_id, slug, name, description, image_url, sort, is_active
		FROM product_lines ORDER BY sort, name`)
	if err != nil {
		return nil, fmt.Errorf("query lines: %w", err)
	}
	lines, err := pgx.CollectRows(lineRows, pgx.RowToStructByPos[Line])
	if err != nil {
		return nil, fmt.Errorf("collect lines: %w", err)
	}

	index := make(map[int64]int, len(brands))
	for i, b := range brands {
		index[b.ID] = i
	}
	for _, l := range lines {
		if i, ok := index[l.BrandID]; ok {
			brands[i].Lines = append(brands[i].Lines, l)
		}
	}
	if brands == nil {
		brands = []Brand{}
	}
	return brands, nil
}

// SaveBrand creates (ID 0) or updates a brand.
func (p *Postgres) SaveBrand(ctx context.Context, b Brand) (Brand, error) {
	auto, err := b.validate()
	if err != nil {
		return Brand{}, err
	}
	if auto {
		if b.Slug, err = uniqueSlug(ctx, p.db, "brands", "", b.Slug, b.ID, nil); err != nil {
			return Brand{}, err
		}
	}
	if b.ID == 0 {
		err = p.db.QueryRow(ctx, `
			INSERT INTO brands (slug, name, description, logo_url, sort, is_active)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			b.Slug, b.Name, b.Description, b.LogoURL, b.Sort, b.IsActive).Scan(&b.ID)
		b.Lines = []Line{}
		return b, mapErr(err)
	}
	err = affected(p.db.Exec(ctx, `
		UPDATE brands SET slug = $2, name = $3, description = $4, logo_url = $5, sort = $6, is_active = $7
		WHERE id = $1`,
		b.ID, b.Slug, b.Name, b.Description, b.LogoURL, b.Sort, b.IsActive))
	b.Lines = []Line{}
	return b, err
}

// DeleteBrand removes a brand; it fails with ErrConflict while products reference it.
func (p *Postgres) DeleteBrand(ctx context.Context, id int64) error {
	return affected(p.db.Exec(ctx, `DELETE FROM brands WHERE id = $1`, id))
}

// SaveLine creates (ID 0) or updates a product line.
func (p *Postgres) SaveLine(ctx context.Context, l Line) (Line, error) {
	auto, err := l.validate()
	if err != nil {
		return Line{}, err
	}
	if auto {
		if l.Slug, err = uniqueSlug(ctx, p.db, "product_lines", " AND brand_id = $3", l.Slug, l.ID, l.BrandID); err != nil {
			return Line{}, err
		}
	}
	if l.ID == 0 {
		err = p.db.QueryRow(ctx, `
			INSERT INTO product_lines (brand_id, slug, name, description, image_url, sort, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			l.BrandID, l.Slug, l.Name, l.Description, l.ImageURL, l.Sort, l.IsActive).Scan(&l.ID)
		return l, mapErr(err)
	}
	return l, affected(p.db.Exec(ctx, `
		UPDATE product_lines SET brand_id = $2, slug = $3, name = $4, description = $5, image_url = $6,
		       sort = $7, is_active = $8
		WHERE id = $1`,
		l.ID, l.BrandID, l.Slug, l.Name, l.Description, l.ImageURL, l.Sort, l.IsActive))
}

// DeleteLine removes a line; its products stay without a line.
func (p *Postgres) DeleteLine(ctx context.Context, id int64) error {
	return affected(p.db.Exec(ctx, `DELETE FROM product_lines WHERE id = $1`, id))
}

// ---- Categories ----

// Categories returns all categories.
func (p *Postgres) Categories(ctx context.Context) ([]Category, error) {
	rows, err := p.db.Query(ctx, `SELECT id, slug, name, sort, is_active FROM categories ORDER BY sort, name`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	cats, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Category])
	if err != nil {
		return nil, fmt.Errorf("collect categories: %w", err)
	}
	if cats == nil {
		cats = []Category{}
	}
	return cats, nil
}

// SaveCategory creates (ID 0) or updates a category.
func (p *Postgres) SaveCategory(ctx context.Context, c Category) (Category, error) {
	auto, err := c.validate()
	if err != nil {
		return Category{}, err
	}
	if auto {
		if c.Slug, err = uniqueSlug(ctx, p.db, "categories", "", c.Slug, c.ID, nil); err != nil {
			return Category{}, err
		}
	}
	if c.ID == 0 {
		err = p.db.QueryRow(ctx,
			`INSERT INTO categories (slug, name, sort, is_active) VALUES ($1, $2, $3, $4) RETURNING id`,
			c.Slug, c.Name, c.Sort, c.IsActive).Scan(&c.ID)
		return c, mapErr(err)
	}
	return c, affected(p.db.Exec(ctx,
		`UPDATE categories SET slug = $2, name = $3, sort = $4, is_active = $5 WHERE id = $1`,
		c.ID, c.Slug, c.Name, c.Sort, c.IsActive))
}

// DeleteCategory removes a category; its products stay uncategorized.
func (p *Postgres) DeleteCategory(ctx context.Context, id int64) error {
	return affected(p.db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id))
}

// ---- Products ----

// Products lists products for the admin table, including hidden ones.
func (p *Postgres) Products(ctx context.Context, f ProductFilter) (ProductPage, error) {
	page := max(f.Page, 1)
	q := strings.TrimSpace(f.Query)
	pattern := ""
	if q != "" {
		pattern = "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
	}
	rows, err := p.db.Query(ctx, `
		SELECT p.id, p.slug, p.name, b.name, COALESCE(l.name, ''), COALESCE(c.name, ''),
		       COALESCE((SELECT m.url FROM product_media m WHERE m.product_id = p.id AND m.kind = 'image'
		                 ORDER BY m.sort, m.id LIMIT 1), ''),
		       v.price_from, COALESCE(v.total_stock, 0)::int, v.variant_count::int, p.is_active,
		       count(*) OVER ()
		FROM products p
		JOIN brands b ON b.id = p.brand_id
		LEFT JOIN product_lines l ON l.id = p.line_id
		LEFT JOIN categories c ON c.id = p.category_id
		CROSS JOIN LATERAL (
		    SELECT min(price) AS price_from, sum(stock) AS total_stock, count(*) AS variant_count
		    FROM product_variants WHERE product_id = p.id
		) v
		WHERE ($1 = '' OR p.name ILIKE $1 OR b.name ILIKE $1 OR l.name ILIKE $1 OR p.slug ILIKE $1)
		  AND ($2 = 0 OR p.brand_id = $2)
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $3 OFFSET $4`, pattern, f.BrandID, PageSize, (page-1)*PageSize)
	if err != nil {
		return ProductPage{}, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	res := ProductPage{Items: []ProductListItem{}, Page: page, PageSize: PageSize}
	for rows.Next() {
		var it ProductListItem
		if err := rows.Scan(&it.ID, &it.Slug, &it.Name, &it.BrandName, &it.LineName, &it.CategoryName,
			&it.ImageURL, &it.PriceFrom, &it.TotalStock, &it.VariantCount, &it.IsActive, &res.Total); err != nil {
			return ProductPage{}, fmt.Errorf("scan product: %w", err)
		}
		res.Items = append(res.Items, it)
	}
	if err := rows.Err(); err != nil {
		return ProductPage{}, fmt.Errorf("iterate products: %w", err)
	}
	return res, nil
}

// Product returns a product with media and all variants.
func (p *Postgres) Product(ctx context.Context, id int64) (Product, error) {
	return getProduct(ctx, p.db, id)
}

func getProduct(ctx context.Context, q querier, id int64) (Product, error) {
	var pr Product
	err := q.QueryRow(ctx, `
		SELECT id, slug, name, brand_id, line_id, category_id, description, ingredients, option_kind, sort, is_active
		FROM products WHERE id = $1`, id).
		Scan(&pr.ID, &pr.Slug, &pr.Name, &pr.BrandID, &pr.LineID, &pr.CategoryID,
			&pr.Description, &pr.Ingredients, &pr.OptionKind, &pr.Sort, &pr.IsActive)
	if err != nil {
		return Product{}, mapErr(err)
	}

	mediaRows, err := q.Query(ctx, `SELECT url, kind FROM product_media WHERE product_id = $1 ORDER BY sort, id`, id)
	if err != nil {
		return Product{}, fmt.Errorf("query media: %w", err)
	}
	if pr.Media, err = pgx.CollectRows(mediaRows, pgx.RowToStructByPos[Media]); err != nil {
		return Product{}, fmt.Errorf("collect media: %w", err)
	}

	variantRows, err := q.Query(ctx, `
		SELECT id, label, sku, price, compare_at_price, stock, is_active, image_url
		FROM product_variants WHERE product_id = $1 ORDER BY sort, id`, id)
	if err != nil {
		return Product{}, fmt.Errorf("query variants: %w", err)
	}
	if pr.Variants, err = pgx.CollectRows(variantRows, pgx.RowToStructByPos[Variant]); err != nil {
		return Product{}, fmt.Errorf("collect variants: %w", err)
	}
	if pr.Media == nil {
		pr.Media = []Media{}
	}
	if pr.Variants == nil {
		pr.Variants = []Variant{}
	}
	return pr, nil
}

// SaveProduct creates (ID 0) or updates a product in one transaction.
// Media is replaced; variants are matched by ID so customers' carts keep working.
func (p *Postgres) SaveProduct(ctx context.Context, pr Product) (Product, error) {
	auto, err := pr.validate()
	if err != nil {
		return Product{}, err
	}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return Product{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if pr.LineID != nil {
		var lineBrand int64
		err := tx.QueryRow(ctx, `SELECT brand_id FROM product_lines WHERE id = $1`, *pr.LineID).Scan(&lineBrand)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && lineBrand != pr.BrandID) {
			return Product{}, apperr.Field("lineId", apperr.Invalid)
		}
		if err != nil {
			return Product{}, fmt.Errorf("check line: %w", err)
		}
	}
	if auto {
		if pr.Slug, err = uniqueSlug(ctx, tx, "products", "", pr.Slug, pr.ID, nil); err != nil {
			return Product{}, err
		}
	}

	if pr.ID == 0 {
		err = tx.QueryRow(ctx, `
			INSERT INTO products (slug, name, brand_id, line_id, category_id, description, ingredients,
			                      option_kind, sort, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
			pr.Slug, pr.Name, pr.BrandID, pr.LineID, pr.CategoryID, pr.Description, pr.Ingredients,
			pr.OptionKind, pr.Sort, pr.IsActive).Scan(&pr.ID)
		if err != nil {
			return Product{}, mapErr(err)
		}
	} else {
		err = affected(tx.Exec(ctx, `
			UPDATE products SET slug = $2, name = $3, brand_id = $4, line_id = $5, category_id = $6,
			       description = $7, ingredients = $8, option_kind = $9, sort = $10, is_active = $11,
			       updated_at = now()
			WHERE id = $1`,
			pr.ID, pr.Slug, pr.Name, pr.BrandID, pr.LineID, pr.CategoryID, pr.Description, pr.Ingredients,
			pr.OptionKind, pr.Sort, pr.IsActive))
		if err != nil {
			return Product{}, err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM product_media WHERE product_id = $1`, pr.ID); err != nil {
		return Product{}, fmt.Errorf("clear media: %w", err)
	}
	for i, m := range pr.Media {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_media (product_id, url, kind, sort) VALUES ($1, $2, $3, $4)`,
			pr.ID, m.URL, m.Kind, i); err != nil {
			return Product{}, fmt.Errorf("insert media: %w", err)
		}
	}

	if err := saveVariants(ctx, tx, pr.ID, pr.Variants); err != nil {
		return Product{}, err
	}

	saved, err := getProduct(ctx, tx, pr.ID)
	if err != nil {
		return Product{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Product{}, fmt.Errorf("commit: %w", err)
	}
	return saved, nil
}

func saveVariants(ctx context.Context, tx pgx.Tx, productID int64, variants []Variant) error {
	keep := make([]int64, 0, len(variants))
	for _, v := range variants {
		if v.ID != 0 {
			keep = append(keep, v.ID)
		}
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM product_variants WHERE product_id = $1 AND NOT (id = ANY($2))`, productID, keep); err != nil {
		return fmt.Errorf("delete variants: %w", err)
	}

	for i, v := range variants {
		if v.ID == 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO product_variants (product_id, label, sku, price, compare_at_price, stock, sort, is_active, image_url)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				productID, v.Label, v.SKU, v.Price, v.CompareAtPrice, v.Stock, i, v.IsActive, v.ImageURL); err != nil {
				return fmt.Errorf("insert variant: %w", mapErr(err))
			}
			continue
		}
		err := affected(tx.Exec(ctx, `
			UPDATE product_variants SET label = $3, sku = $4, price = $5, compare_at_price = $6, stock = $7,
			       sort = $8, is_active = $9, image_url = $10
			WHERE id = $1 AND product_id = $2`,
			v.ID, productID, v.Label, v.SKU, v.Price, v.CompareAtPrice, v.Stock, i, v.IsActive, v.ImageURL))
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.Field(fmt.Sprintf("variants.%d.id", i), apperr.Invalid)
		}
		if err != nil {
			return fmt.Errorf("update variant: %w", err)
		}
	}
	return nil
}

// DeleteProduct removes a product with its media and variants; past orders keep their snapshots.
func (p *Postgres) DeleteProduct(ctx context.Context, id int64) error {
	return affected(p.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id))
}

// ---- Banners ----

// Banners returns all banners.
func (p *Postgres) Banners(ctx context.Context) ([]Banner, error) {
	rows, err := p.db.Query(ctx, `
		SELECT id, title, subtitle, image_url, mobile_image_url, link_url, sort, is_active, starts_at, ends_at
		FROM banners ORDER BY sort, id`)
	if err != nil {
		return nil, fmt.Errorf("query banners: %w", err)
	}
	banners, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Banner])
	if err != nil {
		return nil, fmt.Errorf("collect banners: %w", err)
	}
	if banners == nil {
		banners = []Banner{}
	}
	return banners, nil
}

// SaveBanner creates (ID 0) or updates a banner.
func (p *Postgres) SaveBanner(ctx context.Context, b Banner) (Banner, error) {
	if err := b.validate(); err != nil {
		return Banner{}, err
	}
	if b.ID == 0 {
		err := p.db.QueryRow(ctx, `
			INSERT INTO banners (title, subtitle, image_url, mobile_image_url, link_url, sort, is_active, starts_at, ends_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
			b.Title, b.Subtitle, b.ImageURL, b.MobileImageURL, b.LinkURL, b.Sort, b.IsActive, b.StartsAt, b.EndsAt).
			Scan(&b.ID)
		return b, mapErr(err)
	}
	return b, affected(p.db.Exec(ctx, `
		UPDATE banners SET title = $2, subtitle = $3, image_url = $4, mobile_image_url = $5, link_url = $6,
		       sort = $7, is_active = $8, starts_at = $9, ends_at = $10
		WHERE id = $1`,
		b.ID, b.Title, b.Subtitle, b.ImageURL, b.MobileImageURL, b.LinkURL, b.Sort, b.IsActive, b.StartsAt, b.EndsAt))
}

// DeleteBanner removes a banner.
func (p *Postgres) DeleteBanner(ctx context.Context, id int64) error {
	return affected(p.db.Exec(ctx, `DELETE FROM banners WHERE id = $1`, id))
}

// ---- Promo codes ----

// Promos returns all promo codes.
func (p *Postgres) Promos(ctx context.Context) ([]Promo, error) {
	rows, err := p.db.Query(ctx, `
		SELECT id, code, kind, value, min_subtotal, combinable_with_rules, max_uses, used_count, is_active,
		       starts_at, ends_at
		FROM promo_codes ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("query promos: %w", err)
	}
	promos, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Promo])
	if err != nil {
		return nil, fmt.Errorf("collect promos: %w", err)
	}
	if promos == nil {
		promos = []Promo{}
	}
	return promos, nil
}

// SavePromo creates (ID 0) or updates a promo code. Usage count is never overwritten.
func (p *Postgres) SavePromo(ctx context.Context, pr Promo) (Promo, error) {
	if err := pr.validate(); err != nil {
		return Promo{}, err
	}
	if pr.ID == 0 {
		err := p.db.QueryRow(ctx, `
			INSERT INTO promo_codes (code, kind, value, min_subtotal, combinable_with_rules, max_uses, is_active,
			                         starts_at, ends_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, used_count`,
			pr.Code, pr.Kind, pr.Value, pr.MinSubtotal, pr.CombinableWithRules, pr.MaxUses, pr.IsActive,
			pr.StartsAt, pr.EndsAt).Scan(&pr.ID, &pr.UsedCount)
		return pr, mapErr(err)
	}
	err := p.db.QueryRow(ctx, `
		UPDATE promo_codes SET code = $2, kind = $3, value = $4, min_subtotal = $5, combinable_with_rules = $6,
		       max_uses = $7, is_active = $8, starts_at = $9, ends_at = $10
		WHERE id = $1 RETURNING used_count`,
		pr.ID, pr.Code, pr.Kind, pr.Value, pr.MinSubtotal, pr.CombinableWithRules, pr.MaxUses, pr.IsActive,
		pr.StartsAt, pr.EndsAt).Scan(&pr.UsedCount)
	return pr, mapErr(err)
}

// DeletePromo removes a promo code; orders keep the code as text.
func (p *Postgres) DeletePromo(ctx context.Context, id int64) error {
	return affected(p.db.Exec(ctx, `DELETE FROM promo_codes WHERE id = $1`, id))
}

// ---- Discount rules ----

// Rules returns all discount rules with brand names.
func (p *Postgres) Rules(ctx context.Context) ([]Rule, error) {
	rows, err := p.db.Query(ctx, `
		SELECT r.id, r.name, r.brand_id, COALESCE(b.name, ''), r.min_subtotal, r.kind, r.value, r.is_active,
		       r.starts_at, r.ends_at
		FROM discount_rules r LEFT JOIN brands b ON b.id = r.brand_id
		ORDER BY r.id`)
	if err != nil {
		return nil, fmt.Errorf("query rules: %w", err)
	}
	rules, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Rule])
	if err != nil {
		return nil, fmt.Errorf("collect rules: %w", err)
	}
	if rules == nil {
		rules = []Rule{}
	}
	return rules, nil
}

// SaveRule creates (ID 0) or updates a discount rule.
func (p *Postgres) SaveRule(ctx context.Context, r Rule) (Rule, error) {
	if err := r.validate(); err != nil {
		return Rule{}, err
	}
	var err error
	if r.ID == 0 {
		err = p.db.QueryRow(ctx, `
			INSERT INTO discount_rules (name, brand_id, min_subtotal, kind, value, is_active, starts_at, ends_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
			r.Name, r.BrandID, r.MinSubtotal, r.Kind, r.Value, r.IsActive, r.StartsAt, r.EndsAt).Scan(&r.ID)
	} else {
		err = affected(p.db.Exec(ctx, `
			UPDATE discount_rules SET name = $2, brand_id = $3, min_subtotal = $4, kind = $5, value = $6,
			       is_active = $7, starts_at = $8, ends_at = $9
			WHERE id = $1`,
			r.ID, r.Name, r.BrandID, r.MinSubtotal, r.Kind, r.Value, r.IsActive, r.StartsAt, r.EndsAt))
	}
	if err != nil {
		return Rule{}, mapErr(err)
	}
	r.BrandName = ""
	if r.BrandID != nil {
		_ = p.db.QueryRow(ctx, `SELECT name FROM brands WHERE id = $1`, *r.BrandID).Scan(&r.BrandName)
	}
	return r, nil
}

// DeleteRule removes a discount rule.
func (p *Postgres) DeleteRule(ctx context.Context, id int64) error {
	return affected(p.db.Exec(ctx, `DELETE FROM discount_rules WHERE id = $1`, id))
}

// ---- Orders ----

const orderListColumns = `
	o.id, o.status, o.customer_name, o.phone, o.delivery_method, o.total,
	(SELECT COALESCE(sum(quantity), 0)::int FROM order_items WHERE order_id = o.id), o.created_at`

func scanOrderListItem(row pgx.Row, it *OrderListItem, extra ...any) error {
	dest := append([]any{&it.ID, &it.Status, &it.CustomerName, &it.Phone, &it.Delivery, &it.Total,
		&it.ItemCount, &it.CreatedAt}, extra...)
	if err := row.Scan(dest...); err != nil {
		return err
	}
	it.Number = order.Number(it.ID)
	return nil
}

var orderNumberPattern = regexp.MustCompile(`^(?:BS-?)?0*(\d{1,9})$`)

// searchTerms turns a search box value into an ILIKE pattern for names and the digits to look for
// in phones. Callers match phones only when there are at least 4 digits.
func searchTerms(q string) (namePattern, digits string) {
	digits = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, q)
	// Local format "8 777 ..." is stored as "+7777...".
	if strings.HasPrefix(digits, "8") {
		digits = "7" + digits[1:]
	}
	if q != "" {
		namePattern = "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
	}
	return namePattern, digits
}

// Orders lists orders, newest first. Query matches the phone, name or order number.
func (p *Postgres) Orders(ctx context.Context, f OrderFilter) (OrderPage, error) {
	page := max(f.Page, 1)
	q := strings.TrimSpace(f.Query)
	var orderID int64
	if m := orderNumberPattern.FindStringSubmatch(strings.ToUpper(q)); m != nil {
		orderID, _ = strconv.ParseInt(m[1], 10, 64)
	}
	namePattern, digits := searchTerms(q)

	rows, err := p.db.Query(ctx, `
		SELECT `+orderListColumns+`, count(*) OVER ()
		FROM orders o
		WHERE ($1 = '' OR o.status = $1)
		  AND ($2 = '' OR o.customer_name ILIKE $2 OR o.id = $3 OR (length($4) >= 4 AND o.phone LIKE '%' || $4 || '%'))
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT $5 OFFSET $6`,
		f.Status, namePattern, orderID, digits, PageSize, (page-1)*PageSize)
	if err != nil {
		return OrderPage{}, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	res := OrderPage{Items: []OrderListItem{}, Page: page, PageSize: PageSize}
	for rows.Next() {
		var it OrderListItem
		if err := scanOrderListItem(rows, &it, &res.Total); err != nil {
			return OrderPage{}, fmt.Errorf("scan order: %w", err)
		}
		res.Items = append(res.Items, it)
	}
	if err := rows.Err(); err != nil {
		return OrderPage{}, fmt.Errorf("iterate orders: %w", err)
	}
	return res, nil
}

// Order returns an order with its items.
func (p *Postgres) Order(ctx context.Context, id int64) (Order, error) {
	return getOrder(ctx, p.db, id)
}

func getOrder(ctx context.Context, q querier, id int64) (Order, error) {
	var o Order
	err := scanOrderListItem(q.QueryRow(ctx, `
		SELECT `+orderListColumns+`, o.customer_id, o.address, o.comment, o.promo_code, o.subtotal, o.discount_total
		FROM orders o WHERE o.id = $1`, id),
		&o.OrderListItem, &o.CustomerID, &o.Address, &o.Comment, &o.PromoCode, &o.Subtotal, &o.DiscountTotal)
	if err != nil {
		return Order{}, mapErr(err)
	}

	rows, err := q.Query(ctx, `
		SELECT variant_id, product_name, variant_label, unit_price, quantity, line_total
		FROM order_items WHERE order_id = $1 ORDER BY id`, id)
	if err != nil {
		return Order{}, fmt.Errorf("query items: %w", err)
	}
	if o.Items, err = pgx.CollectRows(rows, pgx.RowToStructByPos[OrderItem]); err != nil {
		return Order{}, fmt.Errorf("collect items: %w", err)
	}
	return o, nil
}

// ErrNotEnoughStock is returned when reopening a cancelled order whose items sold out meanwhile.
var ErrNotEnoughStock = fmt.Errorf("%w: not enough stock to reopen the order", apperr.ErrConflict)

// SetOrderStatus changes the status. Cancelling returns items to stock; reopening reserves them again.
func (p *Postgres) SetOrderStatus(ctx context.Context, id int64, status string) (Order, error) {
	if !orderStatuses[status] {
		return Order{}, apperr.Field("status", apperr.Invalid)
	}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1 FOR UPDATE`, id).Scan(&current); err != nil {
		return Order{}, mapErr(err)
	}

	switch {
	case current != StatusCancelled && status == StatusCancelled:
		if _, err := tx.Exec(ctx, `
			UPDATE product_variants v SET stock = v.stock + i.quantity
			FROM order_items i WHERE i.order_id = $1 AND i.variant_id = v.id`, id); err != nil {
			return Order{}, fmt.Errorf("restore stock: %w", err)
		}
	case current == StatusCancelled && status != StatusCancelled:
		tag, err := tx.Exec(ctx, `
			UPDATE product_variants v SET stock = v.stock - i.quantity
			FROM order_items i WHERE i.order_id = $1 AND i.variant_id = v.id AND v.stock >= i.quantity`, id)
		if err != nil {
			return Order{}, fmt.Errorf("reserve stock: %w", err)
		}
		var lines int64
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM order_items WHERE order_id = $1 AND variant_id IS NOT NULL`, id).Scan(&lines); err != nil {
			return Order{}, fmt.Errorf("count items: %w", err)
		}
		if tag.RowsAffected() != lines {
			return Order{}, ErrNotEnoughStock
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, id, status); err != nil {
		return Order{}, fmt.Errorf("update status: %w", err)
	}
	o, err := getOrder(ctx, tx, id)
	if err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit: %w", err)
	}
	return o, nil
}

// NewOrdersCount returns how many orders wait for the owner's reaction.
func (p *Postgres) NewOrdersCount(ctx context.Context) (int, error) {
	var n int
	err := p.db.QueryRow(ctx, `SELECT count(*) FROM orders WHERE status = 'new'`).Scan(&n)
	return n, err
}

// ---- Customers ----

// customerColumns needs the customerStats join aliased as s.
const customerColumns = `
	c.id, c.name, c.phone, c.marketing_consent,
	COALESCE(s.order_count, 0), COALESCE(s.total_spent, 0), s.last_order_at, c.created_at`

// customerStats aggregates non-cancelled orders, so cancelling an order updates the numbers at once.
const customerStats = `
	LEFT JOIN LATERAL (
		SELECT count(*)::int AS order_count, sum(total)::bigint AS total_spent, max(created_at) AS last_order_at
		FROM orders WHERE customer_id = c.id AND status <> 'cancelled'
	) s ON true`

var customerOrderings = map[string]string{
	SortRecent:   `s.last_order_at DESC NULLS LAST`,
	SortSpent:    `COALESCE(s.total_spent, 0) DESC`,
	SortOrders:   `COALESCE(s.order_count, 0) DESC`,
	SortInactive: `s.last_order_at ASC NULLS LAST`,
}

func scanCustomerListItem(row pgx.Row, it *CustomerListItem, extra ...any) error {
	dest := append([]any{&it.ID, &it.Name, &it.Phone, &it.MarketingConsent,
		&it.OrderCount, &it.TotalSpent, &it.LastOrderAt, &it.CreatedAt}, extra...)
	return row.Scan(dest...)
}

// Customers lists customers. Query matches the name or phone.
func (p *Postgres) Customers(ctx context.Context, f CustomerFilter) (CustomerPage, error) {
	page := max(f.Page, 1)
	limit := PageSize
	items, total, err := p.queryCustomers(ctx, f, &limit, (page-1)*PageSize)
	if err != nil {
		return CustomerPage{}, err
	}
	return CustomerPage{Items: items, Total: total, Page: page, PageSize: PageSize}, nil
}

// ExportCustomers returns every customer matching the filter, for a CSV download.
func (p *Postgres) ExportCustomers(ctx context.Context, f CustomerFilter) ([]CustomerListItem, error) {
	items, _, err := p.queryCustomers(ctx, f, nil, 0)
	return items, err
}

// queryCustomers runs the list query; a nil limit returns all rows.
func (p *Postgres) queryCustomers(ctx context.Context, f CustomerFilter, limit *int, offset int) ([]CustomerListItem, int, error) {
	order, ok := customerOrderings[f.Sort]
	if !ok {
		order = customerOrderings[SortRecent]
	}
	namePattern, digits := searchTerms(strings.TrimSpace(f.Query))

	// order comes from customerOrderings, never from user input.
	rows, err := p.db.Query(ctx, `
		SELECT `+customerColumns+`, count(*) OVER ()
		FROM customers c`+customerStats+`
		WHERE ($1 = '' OR c.name ILIKE $1 OR (length($2) >= 4 AND c.phone LIKE '%' || $2 || '%'))
		  AND (NOT $3 OR c.marketing_consent)
		ORDER BY `+order+`, c.id DESC
		LIMIT $4 OFFSET $5`,
		namePattern, digits, f.Marketing, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query customers: %w", err)
	}
	defer rows.Close()

	items := []CustomerListItem{}
	var total int
	for rows.Next() {
		var it CustomerListItem
		if err := scanCustomerListItem(rows, &it, &total); err != nil {
			return nil, 0, fmt.Errorf("scan customer: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate customers: %w", err)
	}
	return items, total, nil
}

// Customer returns the customer card with all orders, including cancelled ones.
func (p *Postgres) Customer(ctx context.Context, id int64) (Customer, error) {
	var c Customer
	err := scanCustomerListItem(p.db.QueryRow(ctx, `
		SELECT `+customerColumns+`, c.note, c.marketing_consent_at
		FROM customers c`+customerStats+`
		WHERE c.id = $1`, id),
		&c.CustomerListItem, &c.Note, &c.MarketingConsentAt)
	if err != nil {
		return Customer{}, mapErr(err)
	}

	rows, err := p.db.Query(ctx, `
		SELECT `+orderListColumns+`
		FROM orders o WHERE o.customer_id = $1
		ORDER BY o.created_at DESC, o.id DESC`, id)
	if err != nil {
		return Customer{}, fmt.Errorf("query customer orders: %w", err)
	}
	defer rows.Close()
	c.Orders = []OrderListItem{}
	for rows.Next() {
		var it OrderListItem
		if err := scanOrderListItem(rows, &it); err != nil {
			return Customer{}, fmt.Errorf("scan order: %w", err)
		}
		c.Orders = append(c.Orders, it)
	}
	if err := rows.Err(); err != nil {
		return Customer{}, fmt.Errorf("iterate orders: %w", err)
	}
	return c, nil
}

// UpdateCustomer saves the owner's edits. Changing the marketing consent records when it happened.
func (p *Postgres) UpdateCustomer(ctx context.Context, id int64, u CustomerUpdate) (Customer, error) {
	if err := u.validate(); err != nil {
		return Customer{}, err
	}
	// SET expressions see the old row, so the CASE compares against the previous consent.
	if err := affected(p.db.Exec(ctx, `
		UPDATE customers SET
			name = $2,
			note = $3,
			marketing_consent = $4,
			marketing_consent_at = CASE WHEN marketing_consent <> $4 THEN now() ELSE marketing_consent_at END
		WHERE id = $1`, id, u.Name, u.Note, u.MarketingConsent)); err != nil {
		return Customer{}, err
	}
	return p.Customer(ctx, id)
}
