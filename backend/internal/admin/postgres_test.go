package admin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/admin"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/database/dbtest"
)

func setup(t *testing.T) (*pgxpool.Pool, *admin.Postgres) {
	t.Helper()
	pool := dbtest.New(t)
	dbtest.Seed(t, pool)
	return pool, admin.NewPostgres(pool)
}

func ptr[T any](v T) *T { return &v }

func brandID(t *testing.T, pool *pgxpool.Pool, slug string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM brands WHERE slug = $1`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func fieldErr(t *testing.T, err error, field string) string {
	t.Helper()
	var verr *apperr.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	return verr.Fields[field]
}

func TestSaveBrandGeneratesUniqueSlug(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	b, err := repo.SaveBrand(ctx, admin.Brand{Name: "Davines", IsActive: true})
	if err != nil {
		t.Fatalf("SaveBrand() error = %v", err)
	}
	if b.Slug != "davines-2" {
		t.Errorf("slug = %q, want davines-2", b.Slug)
	}

	_, err = repo.SaveBrand(ctx, admin.Brand{Name: "Other", Slug: "payot"})
	if got := fieldErr(t, err, "slug"); got != apperr.Taken {
		t.Errorf("explicit duplicate slug error = %q, want taken", got)
	}
}

func TestDeleteBrandInUse(t *testing.T) {
	pool, repo := setup(t)
	err := repo.DeleteBrand(context.Background(), brandID(t, pool, "davines"))
	if !errors.Is(err, apperr.ErrConflict) {
		t.Errorf("DeleteBrand() error = %v, want ErrConflict", err)
	}
	if err := repo.DeleteBrand(context.Background(), 999_999); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("DeleteBrand(missing) error = %v, want ErrNotFound", err)
	}
}

func TestBrandsIncludeInactiveWithLines(t *testing.T) {
	pool, repo := setup(t)
	dbtest.Exec(t, pool, `UPDATE brands SET is_active = false WHERE slug = 'payot'`)

	brands, err := repo.Brands(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(brands) != 2 || len(brands[0].Lines) != 2 || brands[1].IsActive {
		t.Errorf("brands = %+v", brands)
	}
}

func TestSaveProductCreateAndReconcileVariants(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	davines := brandID(t, pool, "davines")

	created, err := repo.SaveProduct(ctx, admin.Product{
		Name:       "OI Shampoo",
		BrandID:    davines,
		OptionKind: "volume",
		IsActive:   true,
		Media:      []admin.Media{{URL: "/uploads/a.jpg"}, {URL: "/uploads/b.jpg"}},
		Variants: []admin.Variant{
			{Label: "250 ml", Price: 17000, Stock: 3, IsActive: true},
			{Label: "1000 ml", Price: 45000, CompareAtPrice: ptr[int64](50000), Stock: 1, IsActive: true},
		},
	})
	if err != nil {
		t.Fatalf("SaveProduct(create) error = %v", err)
	}
	if created.Slug != "oi-shampoo" || len(created.Variants) != 2 || created.Media[0].Kind != "image" {
		t.Fatalf("created = %+v", created)
	}
	keptID := created.Variants[0].ID

	// Update: change the first variant, drop the second, add a new one, reorder photos.
	created.Name = "OI Shampoo New"
	created.Variants = []admin.Variant{
		{ID: keptID, Label: "250 ml", Price: 18000, Stock: 5, IsActive: true},
		{Label: "100 ml", Price: 9000, Stock: 2, IsActive: true},
	}
	created.Media = []admin.Media{{URL: "/uploads/b.jpg"}}
	updated, err := repo.SaveProduct(ctx, created)
	if err != nil {
		t.Fatalf("SaveProduct(update) error = %v", err)
	}
	if updated.Slug != "oi-shampoo" {
		t.Errorf("slug changed to %q; existing slugs must stay stable", updated.Slug)
	}
	if len(updated.Variants) != 2 || updated.Variants[0].ID != keptID || updated.Variants[0].Price != 18000 {
		t.Errorf("variants = %+v, want kept ID %d first", updated.Variants, keptID)
	}
	if len(updated.Media) != 1 || updated.Media[0].URL != "/uploads/b.jpg" {
		t.Errorf("media = %+v", updated.Media)
	}
}

func TestSaveProductValidation(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	payotLine := int64(0)
	if err := pool.QueryRow(ctx, `SELECT id FROM product_lines WHERE slug = 'hydra-24'`).Scan(&payotLine); err != nil {
		t.Fatal(err)
	}

	_, err := repo.SaveProduct(ctx, admin.Product{
		Variants: []admin.Variant{{Price: 100, CompareAtPrice: ptr[int64](50)}, {Price: -1}},
	})
	for _, f := range []string{"name", "brandId", "variants.0.label", "variants.0.compareAtPrice", "variants.1.price"} {
		if fieldErr(t, err, f) == "" {
			t.Errorf("missing error for %s", f)
		}
	}

	_, err = repo.SaveProduct(ctx, admin.Product{
		Name: "Wrong line", BrandID: brandID(t, pool, "davines"), LineID: &payotLine,
		Variants: []admin.Variant{{Price: 100}},
	})
	if got := fieldErr(t, err, "lineId"); got != apperr.Invalid {
		t.Errorf("line of another brand error = %q, want invalid", got)
	}

	_, err = repo.SaveProduct(ctx, admin.Product{
		Name: "Ghost brand", BrandID: 999_999, Variants: []admin.Variant{{Price: 100}},
	})
	if got := fieldErr(t, err, "brandId"); got != apperr.Invalid {
		t.Errorf("missing brand error = %q, want invalid", got)
	}
}

func TestProductsListIncludesHidden(t *testing.T) {
	pool, repo := setup(t)
	dbtest.Exec(t, pool, `UPDATE products SET is_active = false WHERE slug = 'davines-minu-shampoo'`)

	page, err := repo.Products(context.Background(), admin.ProductFilter{Query: "minu"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].IsActive || page.Items[0].PriceFrom == nil {
		t.Errorf("page = %+v", page)
	}
}

func TestPromoCodeNormalizedAndUnique(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	p, err := repo.SavePromo(ctx, admin.Promo{Code: " spring10 ", Kind: "percent", Value: 10, IsActive: true})
	if err != nil || p.Code != "SPRING10" {
		t.Fatalf("SavePromo() = %+v, %v", p, err)
	}
	_, err = repo.SavePromo(ctx, admin.Promo{Code: "welcome5", Kind: "fixed", Value: 1000})
	if got := fieldErr(t, err, "code"); got != apperr.Taken {
		t.Errorf("duplicate code error = %q, want taken", got)
	}
	_, err = repo.SavePromo(ctx, admin.Promo{Code: "BAD CODE", Kind: "percent", Value: 150})
	if fieldErr(t, err, "code") == "" || fieldErr(t, err, "value") == "" {
		t.Errorf("want code and value errors, got %v", err)
	}
}

func TestToggleRule(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	rules, err := repo.Rules(ctx)
	if err != nil || len(rules) != 1 || rules[0].BrandName != "Davines" {
		t.Fatalf("Rules() = %+v, %v", rules, err)
	}
	r := rules[0]
	r.IsActive = false
	saved, err := repo.SaveRule(ctx, r)
	if err != nil || saved.IsActive || saved.BrandName != "Davines" {
		t.Errorf("SaveRule() = %+v, %v", saved, err)
	}
}

func TestOrderStatusMovesStock(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()

	var variantID int64
	if err := pool.QueryRow(ctx, `SELECT v.id FROM product_variants v JOIN products p ON p.id = v.product_id
		WHERE p.slug = 'davines-minu-shampoo'`).Scan(&variantID); err != nil {
		t.Fatal(err)
	}
	var orderID int64
	if err := pool.QueryRow(ctx, `INSERT INTO orders (customer_name, phone, delivery_method, subtotal, total)
		VALUES ('Test', '+77011234567', 'pickup', 39000, 39000) RETURNING id`).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	dbtest.Exec(t, pool, `INSERT INTO order_items (order_id, variant_id, product_name, unit_price, quantity, line_total)
		VALUES ($1, $2, 'MINU Shampoo', 19500, 2, 39000)`, orderID, variantID)
	stock := func() (s int) {
		_ = pool.QueryRow(ctx, `SELECT stock FROM product_variants WHERE id = $1`, variantID).Scan(&s)
		return s
	}
	before := stock()

	o, err := repo.SetOrderStatus(ctx, orderID, admin.StatusCancelled)
	if err != nil || o.Status != admin.StatusCancelled || len(o.Items) != 1 {
		t.Fatalf("cancel = %+v, %v", o, err)
	}
	if stock() != before+2 {
		t.Errorf("stock after cancel = %d, want %d", stock(), before+2)
	}

	if _, err := repo.SetOrderStatus(ctx, orderID, admin.StatusPaid); err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	if stock() != before {
		t.Errorf("stock after reopen = %d, want %d", stock(), before)
	}

	// Cancel again, sell everything, then reopening must fail without changing anything.
	if _, err := repo.SetOrderStatus(ctx, orderID, admin.StatusCancelled); err != nil {
		t.Fatal(err)
	}
	dbtest.Exec(t, pool, `UPDATE product_variants SET stock = 1 WHERE id = $1`, variantID)
	if _, err := repo.SetOrderStatus(ctx, orderID, admin.StatusNew); !errors.Is(err, admin.ErrNotEnoughStock) {
		t.Errorf("reopen without stock error = %v, want ErrNotEnoughStock", err)
	}
	if stock() != 1 {
		t.Errorf("stock changed on failed reopen: %d", stock())
	}

	if _, err := repo.SetOrderStatus(ctx, orderID, "lost"); fieldErr(t, err, "status") == "" {
		t.Errorf("invalid status accepted")
	}
}

func TestOrdersSearch(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	dbtest.Exec(t, pool, `INSERT INTO orders (customer_name, phone, delivery_method, subtotal, total) VALUES
		('Айгерим', '+77011234567', 'pickup', 1000, 1000),
		('Динара', '+77779998877', 'courier', 2000, 2000)`)
	dbtest.Exec(t, pool, `UPDATE orders SET status = 'paid' WHERE customer_name = 'Динара'`)

	tests := []struct {
		filter admin.OrderFilter
		want   int
	}{
		{admin.OrderFilter{}, 2},
		{admin.OrderFilter{Status: "paid"}, 1},
		{admin.OrderFilter{Query: "айгер"}, 1},
		{admin.OrderFilter{Query: "8 777 999"}, 1},
		{admin.OrderFilter{Query: "BS-00001"}, 1},
	}
	for _, tt := range tests {
		page, err := repo.Orders(ctx, tt.filter)
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != tt.want {
			t.Errorf("Orders(%+v) total = %d, want %d", tt.filter, page.Total, tt.want)
		}
	}
	if n, _ := repo.NewOrdersCount(ctx); n != 1 {
		t.Errorf("NewOrdersCount = %d, want 1", n)
	}
}
