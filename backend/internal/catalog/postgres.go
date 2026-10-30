package catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres reads the catalog from PostgreSQL.
type Postgres struct {
	db *pgxpool.Pool
}

// NewPostgres creates a catalog repository.
func NewPostgres(db *pgxpool.Pool) *Postgres {
	return &Postgres{db: db}
}

// Menu returns active brands with their lines and active categories.
func (p *Postgres) Menu(ctx context.Context) (Menu, error) {
	rows, err := p.db.Query(ctx, `
		SELECT b.id, b.slug, b.name, b.description, b.logo_url,
		       l.id, l.slug, l.name, l.description, l.image_url
		FROM brands b
		LEFT JOIN product_lines l ON l.brand_id = b.id AND l.is_active
		WHERE b.is_active
		ORDER BY b.sort, b.name, l.sort, l.name`)
	if err != nil {
		return Menu{}, fmt.Errorf("query brands: %w", err)
	}
	defer rows.Close()

	menu := Menu{Brands: []Brand{}, Categories: []Category{}}
	for rows.Next() {
		var (
			b                                     Brand
			lineID                                *int64
			lineSlug, lineName, lineDesc, lineImg *string
		)
		if err := rows.Scan(&b.ID, &b.Slug, &b.Name, &b.Description, &b.LogoURL,
			&lineID, &lineSlug, &lineName, &lineDesc, &lineImg); err != nil {
			return Menu{}, fmt.Errorf("scan brand: %w", err)
		}
		if n := len(menu.Brands); n == 0 || menu.Brands[n-1].ID != b.ID {
			b.Lines = []Line{}
			menu.Brands = append(menu.Brands, b)
		}
		if lineID != nil {
			last := &menu.Brands[len(menu.Brands)-1]
			last.Lines = append(last.Lines, Line{ID: *lineID, Slug: *lineSlug, Name: *lineName, Description: *lineDesc, ImageURL: *lineImg})
		}
	}
	if err := rows.Err(); err != nil {
		return Menu{}, fmt.Errorf("iterate brands: %w", err)
	}

	catRows, err := p.db.Query(ctx, `SELECT id, slug, name FROM categories WHERE is_active ORDER BY sort, name`)
	if err != nil {
		return Menu{}, fmt.Errorf("query categories: %w", err)
	}
	cats, err := pgx.CollectRows(catRows, func(r pgx.CollectableRow) (Category, error) {
		var c Category
		err := r.Scan(&c.ID, &c.Slug, &c.Name)
		return c, err
	})
	if err != nil {
		return Menu{}, fmt.Errorf("collect categories: %w", err)
	}
	if len(cats) > 0 {
		menu.Categories = cats
	}
	return menu, nil
}

// Banners returns banners that are active right now.
func (p *Postgres) Banners(ctx context.Context) ([]Banner, error) {
	rows, err := p.db.Query(ctx, `
		SELECT id, title, subtitle, image_url, mobile_image_url, link_url
		FROM banners
		WHERE is_active
		  AND (starts_at IS NULL OR starts_at <= now())
		  AND (ends_at IS NULL OR ends_at > now())
		ORDER BY sort, id`)
	if err != nil {
		return nil, fmt.Errorf("query banners: %w", err)
	}
	banners, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Banner, error) {
		var b Banner
		err := r.Scan(&b.ID, &b.Title, &b.Subtitle, &b.ImageURL, &b.MobileImageURL, &b.LinkURL)
		return b, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect banners: %w", err)
	}
	if banners == nil {
		banners = []Banner{}
	}
	return banners, nil
}

// Products returns a filtered page of active products that have at least one active variant.
func (p *Postgres) Products(ctx context.Context, f Filter) (Page, error) {
	f = f.Normalize()
	rows, err := p.db.Query(ctx, `
		SELECT p.id, p.slug, p.name, b.slug, b.name, l.slug, l.name,
		       COALESCE((SELECT m.url FROM product_media m
		                 WHERE m.product_id = p.id AND m.kind = 'image'
		                 ORDER BY m.sort, m.id LIMIT 1), ''),
		       v.price, v.compare_at_price, v.variant_count > 1, v.total_stock > 0,
		       count(*) OVER ()
		FROM products p
		JOIN brands b ON b.id = p.brand_id AND b.is_active
		LEFT JOIN product_lines l ON l.id = p.line_id
		LEFT JOIN categories c ON c.id = p.category_id
		JOIN LATERAL (
		    SELECT (array_agg(pv.price ORDER BY pv.price, pv.id))[1] AS price,
		           (array_agg(pv.compare_at_price ORDER BY pv.price, pv.id))[1] AS compare_at_price,
		           count(*) AS variant_count,
		           sum(pv.stock) AS total_stock
		    FROM product_variants pv
		    WHERE pv.product_id = p.id AND pv.is_active
		    HAVING count(*) > 0
		) v ON true
		WHERE p.is_active
		  AND ($1 = '' OR b.slug = $1)
		  AND ($2 = '' OR l.slug = $2)
		  AND ($3 = '' OR c.slug = $3)
		  AND ($4 = '' OR p.name ILIKE $4 OR b.name ILIKE $4 OR l.name ILIKE $4)
		ORDER BY p.sort, p.created_at DESC, p.id DESC
		LIMIT $5 OFFSET $6`,
		f.Brand, f.Line, f.Category, likePattern(f.Query), f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return Page{}, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	page := Page{Items: []ProductSummary{}, Page: f.Page, PageSize: f.PageSize}
	for rows.Next() {
		var (
			s                  ProductSummary
			lineSlug, lineName *string
		)
		if err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.Brand.Slug, &s.Brand.Name, &lineSlug, &lineName,
			&s.ImageURL, &s.PriceFrom, &s.CompareAtPrice, &s.HasOptions, &s.InStock, &page.Total); err != nil {
			return Page{}, fmt.Errorf("scan product: %w", err)
		}
		s.Line = optionalRef(lineSlug, lineName)
		page.Items = append(page.Items, s)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate products: %w", err)
	}
	return page, nil
}

// Product returns the full product page by slug.
func (p *Postgres) Product(ctx context.Context, slug string) (Product, error) {
	var (
		pr                         Product
		lineSlug, lineName         *string
		categorySlug, categoryName *string
	)
	err := p.db.QueryRow(ctx, `
		SELECT p.id, p.slug, p.name, b.slug, b.name, l.slug, l.name, c.slug, c.name,
		       p.description, p.ingredients, p.option_kind
		FROM products p
		JOIN brands b ON b.id = p.brand_id AND b.is_active
		LEFT JOIN product_lines l ON l.id = p.line_id
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE p.slug = $1 AND p.is_active`, slug).
		Scan(&pr.ID, &pr.Slug, &pr.Name, &pr.Brand.Slug, &pr.Brand.Name, &lineSlug, &lineName,
			&categorySlug, &categoryName, &pr.Description, &pr.Ingredients, &pr.OptionKind)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("query product: %w", err)
	}
	pr.Line = optionalRef(lineSlug, lineName)
	pr.Category = optionalRef(categorySlug, categoryName)

	mediaRows, err := p.db.Query(ctx,
		`SELECT url, kind FROM product_media WHERE product_id = $1 ORDER BY sort, id`, pr.ID)
	if err != nil {
		return Product{}, fmt.Errorf("query media: %w", err)
	}
	pr.Media, err = pgx.CollectRows(mediaRows, pgx.RowToStructByPos[Media])
	if err != nil {
		return Product{}, fmt.Errorf("collect media: %w", err)
	}

	variantRows, err := p.db.Query(ctx, `
		SELECT id, label, sku, price, compare_at_price, stock > 0, image_url
		FROM product_variants
		WHERE product_id = $1 AND is_active
		ORDER BY sort, price, id`, pr.ID)
	if err != nil {
		return Product{}, fmt.Errorf("query variants: %w", err)
	}
	pr.Variants, err = pgx.CollectRows(variantRows, pgx.RowToStructByPos[Variant])
	if err != nil {
		return Product{}, fmt.Errorf("collect variants: %w", err)
	}
	if len(pr.Variants) == 0 {
		return Product{}, ErrNotFound
	}
	if pr.Media == nil {
		pr.Media = []Media{}
	}
	return pr, nil
}

func optionalRef(slug, name *string) *Ref {
	if slug == nil || name == nil {
		return nil
	}
	return &Ref{Slug: *slug, Name: *name}
}

// likePattern turns a search query into an ILIKE pattern with wildcards escaped.
func likePattern(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	q = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	return "%" + q + "%"
}

// SitemapProducts lists visible products for sitemap.xml.
func (p *Postgres) SitemapProducts(ctx context.Context) ([]SitemapProduct, error) {
	rows, err := p.db.Query(ctx, `
		SELECT p.slug, p.updated_at
		FROM products p
		JOIN brands b ON b.id = p.brand_id AND b.is_active
		WHERE p.is_active AND EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id AND v.is_active)
		ORDER BY p.id`)
	if err != nil {
		return nil, fmt.Errorf("query sitemap products: %w", err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SitemapProduct])
	if err != nil {
		return nil, fmt.Errorf("collect sitemap products: %w", err)
	}
	return items, nil
}
