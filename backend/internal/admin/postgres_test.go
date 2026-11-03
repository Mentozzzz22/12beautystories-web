package admin_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
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

// insertOrder stores a bare order and its customer the way checkout does.
func insertOrder(t *testing.T, pool *pgxpool.Pool, name, phone string, total int64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		WITH c AS (
			INSERT INTO customers (phone, name) VALUES ($2, $1)
			ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		)
		INSERT INTO orders (customer_id, customer_name, phone, delivery_method, subtotal, total)
		SELECT c.id, $1, $2, 'pickup', $3, $3 FROM c
		RETURNING id`, name, phone, total).Scan(&id)
	if err != nil {
		t.Fatalf("insert order: %v", err)
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
			{Label: "250 ml", Price: 17000, Stock: 3, IsActive: true, ImageURL: "/uploads/a.jpg"},
			{Label: "1000 ml", Price: 50000, SalePrice: ptr[int64](45000), Stock: 1, IsActive: true},
		},
	})
	if err != nil {
		t.Fatalf("SaveProduct(create) error = %v", err)
	}
	if created.Slug != "oi-shampoo" || len(created.Variants) != 2 || created.Media[0].Kind != "image" {
		t.Fatalf("created = %+v", created)
	}
	if created.Variants[0].ImageURL != "/uploads/a.jpg" || created.Variants[1].ImageURL != "" {
		t.Errorf("variant images = %+v", created.Variants)
	}
	keptID := created.Variants[0].ID

	// Update: change the first variant, drop the second, add a new one, reorder photos.
	created.Name = "OI Shampoo New"
	created.Variants = []admin.Variant{
		{ID: keptID, Label: "250 ml", Price: 18000, Stock: 5, IsActive: true, ImageURL: "/uploads/b.jpg"},
		{Label: "100 ml", Price: 9000, Stock: 2, IsActive: true, ImageURL: "/uploads/b.jpg"},
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
	for _, v := range updated.Variants {
		if v.ImageURL != "/uploads/b.jpg" {
			t.Errorf("variant %q image = %q, want the chosen photo", v.Label, v.ImageURL)
		}
	}
}

func TestSaveProductCategories(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	category := func(slug string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `SELECT id FROM categories WHERE slug = $1`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	shampoo, leaveIn, cream := category("shampoo"), category("leave-in"), category("cream")

	saved, err := repo.SaveProduct(ctx, admin.Product{
		Name:        "OI Oil",
		BrandID:     brandID(t, pool, "davines"),
		CategoryIDs: []int64{leaveIn, shampoo, leaveIn},
		Variants:    []admin.Variant{{Price: 100, IsActive: true}},
	})
	if err != nil {
		t.Fatalf("SaveProduct(create) error = %v", err)
	}
	if !slices.Equal(saved.CategoryIDs, []int64{shampoo, leaveIn}) {
		t.Errorf("categories = %v, want %v without duplicates", saved.CategoryIDs, []int64{shampoo, leaveIn})
	}

	saved.CategoryIDs = []int64{cream}
	if saved, err = repo.SaveProduct(ctx, saved); err != nil {
		t.Fatalf("SaveProduct(update) error = %v", err)
	}
	if !slices.Equal(saved.CategoryIDs, []int64{cream}) {
		t.Errorf("categories = %v, want only %d", saved.CategoryIDs, cream)
	}

	saved.CategoryIDs = nil
	if saved, err = repo.SaveProduct(ctx, saved); err != nil {
		t.Fatalf("SaveProduct(clear) error = %v", err)
	}
	if saved.CategoryIDs == nil || len(saved.CategoryIDs) != 0 {
		t.Errorf("categories = %#v, want an empty list", saved.CategoryIDs)
	}

	saved.CategoryIDs = []int64{999_999}
	_, err = repo.SaveProduct(ctx, saved)
	if got := fieldErr(t, err, "categoryIds"); got != apperr.Invalid {
		t.Errorf("unknown category error = %q, want invalid", got)
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
		Media: []admin.Media{{URL: "/uploads/clip.mp4", Kind: "video"}},
		Variants: []admin.Variant{
			{Price: 100, SalePrice: ptr[int64](150), ImageURL: "/uploads/other.jpg"},
			{Price: -1, ImageURL: "/uploads/clip.mp4"},
		},
	})
	for _, f := range []string{
		"name", "brandId", "variants.0.label", "variants.0.salePrice", "variants.0.imageUrl",
		"variants.1.price", "variants.1.imageUrl",
	} {
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
	orderID := insertOrder(t, pool, "Test", "+77011234567", 39000)
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
	insertOrder(t, pool, "Айгерим", "+77011234567", 1000)
	insertOrder(t, pool, "Динара", "+77779998877", 2000)
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

func TestCustomers(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()

	first := insertOrder(t, pool, "Айгерим", "+77011234567", 10_000)
	insertOrder(t, pool, "Айгерим", "+77011234567", 5_000)
	cancelled := insertOrder(t, pool, "Айгерим", "+77011234567", 99_000)
	insertOrder(t, pool, "Динара", "+77779998877", 30_000)
	dbtest.Exec(t, pool, `UPDATE orders SET status = 'cancelled' WHERE id = $1`, cancelled)
	dbtest.Exec(t, pool, `UPDATE orders SET created_at = now() - interval '30 days' WHERE id = $1`, first)
	dbtest.Exec(t, pool, `UPDATE customers SET marketing_consent = true WHERE name = 'Динара'`)

	page, err := repo.Customers(ctx, admin.CustomerFilter{Sort: admin.SortSpent})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Items[0].Name != "Динара" {
		t.Fatalf("Customers(spent) = %+v, want Динара first of 2", page.Items)
	}
	aigerim := page.Items[1]
	if aigerim.OrderCount != 2 || aigerim.TotalSpent != 15_000 || aigerim.LastOrderAt == nil {
		t.Errorf("stats = %+v, want 2 orders for 15000 without the cancelled one", aigerim)
	}

	tests := []struct {
		filter admin.CustomerFilter
		want   int
	}{
		{admin.CustomerFilter{Query: "дина"}, 1},
		{admin.CustomerFilter{Query: "8 701 123"}, 1},
		{admin.CustomerFilter{Marketing: true}, 1},
		{admin.CustomerFilter{Sort: "drop table"}, 2},
	}
	for _, tt := range tests {
		got, err := repo.Customers(ctx, tt.filter)
		if err != nil {
			t.Fatal(err)
		}
		if got.Total != tt.want {
			t.Errorf("Customers(%+v) total = %d, want %d", tt.filter, got.Total, tt.want)
		}
	}

	all, err := repo.ExportCustomers(ctx, admin.CustomerFilter{})
	if err != nil || len(all) != 2 {
		t.Errorf("ExportCustomers() = %d rows, %v; want 2", len(all), err)
	}

	card, err := repo.Customer(ctx, aigerim.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Orders) != 3 || card.MarketingConsentAt != nil {
		t.Errorf("card = %d orders, consent at %v; want all 3 orders and no consent", len(card.Orders), card.MarketingConsentAt)
	}
	if o, err := repo.Order(ctx, first); err != nil || o.CustomerID != aigerim.ID {
		t.Errorf("Order().CustomerID = %d, %v; want %d", o.CustomerID, err, aigerim.ID)
	}
}

func TestUpdateCustomer(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	insertOrder(t, pool, "Айгерим", "+77011234567", 1000)
	var id int64
	if err := pool.QueryRow(ctx, `SELECT id FROM customers`).Scan(&id); err != nil {
		t.Fatal(err)
	}

	c, err := repo.UpdateCustomer(ctx, id, admin.CustomerUpdate{Name: " Айгерим С. ", Note: "сухие волосы", MarketingConsent: true})
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "Айгерим С." || c.Note != "сухие волосы" || !c.MarketingConsent || c.MarketingConsentAt == nil {
		t.Errorf("UpdateCustomer() = %+v", c)
	}
	grantedAt := *c.MarketingConsentAt

	// Saving without changing the consent keeps its timestamp.
	c, err = repo.UpdateCustomer(ctx, id, admin.CustomerUpdate{Name: "Айгерим", MarketingConsent: true})
	if err != nil || !c.MarketingConsentAt.Equal(grantedAt) {
		t.Errorf("consent time changed: %v -> %v, %v", grantedAt, c.MarketingConsentAt, err)
	}

	if _, err := repo.UpdateCustomer(ctx, id, admin.CustomerUpdate{Name: " "}); fieldErr(t, err, "name") == "" {
		t.Error("empty name accepted")
	}
	if _, err := repo.UpdateCustomer(ctx, 999_999, admin.CustomerUpdate{Name: "X"}); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("unknown customer error = %v, want ErrNotFound", err)
	}
}

func productID(t *testing.T, pool *pgxpool.Pool, slug string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM products WHERE slug = $1`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestBundleOfferLifecycle(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	energizing := productID(t, pool, "davines-energizing-shampoo")
	minu := productID(t, pool, "davines-minu-shampoo")

	offer := admin.BundleOffer{
		Name:     "Наборы",
		IsActive: true,
		Groups: []admin.BundleGroup{
			{Name: "шампунь", ProductIDs: []int64{energizing}},
			{Name: "уход", ProductIDs: []int64{minu}},
		},
		Tiers: []admin.BundleTier{
			{Percent: 15, Slots: []admin.BundleSlot{{Group: 0, Label: "250 ml"}}},
			{Percent: 20, Slots: []admin.BundleSlot{{Group: 0, Label: "250 ml"}, {Group: 1}}},
		},
	}
	created, err := repo.SaveBundleOffer(ctx, offer)
	if err != nil {
		t.Fatalf("SaveBundleOffer(create) error = %v", err)
	}

	offers, err := repo.BundleOffers(ctx)
	if err != nil {
		t.Fatalf("BundleOffers() error = %v", err)
	}
	offer.ID = created.ID
	if !reflect.DeepEqual(offers, []admin.BundleOffer{offer}) {
		t.Fatalf("offers = %+v, want %+v", offers, offer)
	}

	// Saving replaces groups and tiers instead of adding to them.
	offer.Groups = offer.Groups[:1]
	offer.Tiers = offer.Tiers[:1]
	if _, err := repo.SaveBundleOffer(ctx, offer); err != nil {
		t.Fatalf("SaveBundleOffer(update) error = %v", err)
	}
	offers, _ = repo.BundleOffers(ctx)
	if !reflect.DeepEqual(offers, []admin.BundleOffer{offer}) {
		t.Fatalf("after update offers = %+v, want %+v", offers, offer)
	}

	// Deleting a product takes it out of the group but keeps the offer.
	if _, err := pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, energizing); err != nil {
		t.Fatal(err)
	}
	offers, _ = repo.BundleOffers(ctx)
	if len(offers) != 1 || len(offers[0].Groups[0].ProductIDs) != 0 {
		t.Errorf("after product delete offers = %+v", offers)
	}

	if err := repo.DeleteBundleOffer(ctx, created.ID); err != nil {
		t.Fatalf("DeleteBundleOffer() error = %v", err)
	}
	if offers, _ = repo.BundleOffers(ctx); len(offers) != 0 {
		t.Errorf("offers after delete = %+v", offers)
	}
}

func TestBundleOfferValidation(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()
	valid := func() admin.BundleOffer {
		return admin.BundleOffer{
			Name:   "Наборы",
			Groups: []admin.BundleGroup{{Name: "шампунь", ProductIDs: []int64{1}}},
			Tiers:  []admin.BundleTier{{Percent: 15, Slots: []admin.BundleSlot{{Group: 0}}}},
		}
	}
	tests := []struct {
		name   string
		change func(*admin.BundleOffer)
	}{
		{"no name", func(o *admin.BundleOffer) { o.Name = " " }},
		{"no groups", func(o *admin.BundleOffer) { o.Groups = nil; o.Tiers[0].Slots = nil }},
		{"group without products", func(o *admin.BundleOffer) { o.Groups[0].ProductIDs = nil }},
		{"no tiers", func(o *admin.BundleOffer) { o.Tiers = nil }},
		{"percent too big", func(o *admin.BundleOffer) { o.Tiers[0].Percent = 101 }},
		{"tier without slots", func(o *admin.BundleOffer) { o.Tiers[0].Slots = nil }},
		{"unknown group", func(o *admin.BundleOffer) { o.Tiers[0].Slots[0].Group = 5 }},
		{"same group twice", func(o *admin.BundleOffer) {
			o.Tiers[0].Slots = []admin.BundleSlot{{Group: 0}, {Group: 0, Label: "200"}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := valid()
			tt.change(&o)
			var verr *apperr.ValidationError
			if _, err := repo.SaveBundleOffer(ctx, o); !errors.As(err, &verr) {
				t.Errorf("error = %v, want validation fields", err)
			}
		})
	}
}

func TestBundleOptions(t *testing.T) {
	_, repo := setup(t)
	options, err := repo.BundleOptions(context.Background())
	if err != nil {
		t.Fatalf("BundleOptions() error = %v", err)
	}
	if len(options) != 4 {
		t.Fatalf("options = %+v, want 4 products", options)
	}
	var energizing admin.BundleOption
	for _, o := range options {
		if o.Name == "Energizing Shampoo" {
			energizing = o
		}
	}
	if energizing.LineName != "Energizing" || len(energizing.Labels) != 3 {
		t.Errorf("energizing = %+v", energizing)
	}
}

func productIDs(t *testing.T, pool *pgxpool.Pool, slugs ...string) []int64 {
	t.Helper()
	var ids []int64
	if err := pool.QueryRow(context.Background(),
		`SELECT array_agg(id ORDER BY id) FROM products WHERE slug = ANY($1)`, slugs).Scan(&ids); err != nil {
		t.Fatal(err)
	}
	return ids
}

func salePrices(t *testing.T, pool *pgxpool.Pool, name string) []any {
	t.Helper()
	var prices []*int64
	if err := pool.QueryRow(context.Background(), `
		SELECT array_agg(v.sale_price ORDER BY v.sort, v.id)
		FROM product_variants v JOIN products p ON p.id = v.product_id WHERE p.name = $1`, name).Scan(&prices); err != nil {
		t.Fatal(err)
	}
	out := make([]any, len(prices))
	for i, p := range prices {
		if p != nil {
			out[i] = *p
		}
	}
	return out
}

func TestApplyAndClearSale(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	ids := productIDs(t, pool, "davines-energizing-shampoo", "davines-minu-shampoo")

	if got := salePrices(t, pool, "Energizing Shampoo"); !reflect.DeepEqual(got, []any{nil, nil, int64(48000)}) {
		t.Fatalf("seeded sale prices = %v", got)
	}

	res, err := repo.ApplySale(ctx, admin.SaleRequest{ProductIDs: ids, Percent: 30})
	if err != nil {
		t.Fatalf("ApplySale() error = %v", err)
	}
	if res.Updated != 4 {
		t.Errorf("updated = %d, want 4", res.Updated)
	}
	// 9 000, 18 000 and 54 000 minus 30%; the old 48 000 sale price is replaced.
	if got := salePrices(t, pool, "Energizing Shampoo"); !reflect.DeepEqual(got, []any{int64(6300), int64(12600), int64(37800)}) {
		t.Errorf("energizing sale prices = %v", got)
	}
	if got := salePrices(t, pool, "MINU Shampoo"); !reflect.DeepEqual(got, []any{int64(13650)}) {
		t.Errorf("minu sale prices = %v", got)
	}

	res, err = repo.ClearSale(ctx, admin.SaleRequest{ProductIDs: ids[:1]})
	if err != nil {
		t.Fatalf("ClearSale() error = %v", err)
	}
	if res.Updated != 3 {
		t.Errorf("cleared = %d, want 3", res.Updated)
	}
	if got := salePrices(t, pool, "Energizing Shampoo"); !reflect.DeepEqual(got, []any{nil, nil, nil}) {
		t.Errorf("energizing after clear = %v", got)
	}
	if got := salePrices(t, pool, "MINU Shampoo"); !reflect.DeepEqual(got, []any{int64(13650)}) {
		t.Errorf("minu after clearing another product = %v", got)
	}
}

func TestSaleValidation(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	_, err := repo.ApplySale(ctx, admin.SaleRequest{Percent: 100})
	if fieldErr(t, err, "productIds") == "" || fieldErr(t, err, "percent") == "" {
		t.Errorf("ApplySale error = %v, want productIds and percent", err)
	}
	_, err = repo.ClearSale(ctx, admin.SaleRequest{})
	if fieldErr(t, err, "productIds") == "" {
		t.Errorf("ClearSale error = %v, want productIds", err)
	}
}

func TestProductsFilterByLineAndSale(t *testing.T) {
	pool, repo := setup(t)
	ctx := context.Background()
	var energizing int64
	if err := pool.QueryRow(ctx, `SELECT id FROM product_lines WHERE slug = 'energizing'`).Scan(&energizing); err != nil {
		t.Fatal(err)
	}

	page, err := repo.Products(ctx, admin.ProductFilter{LineID: energizing})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].Name != "Energizing Shampoo" {
		t.Fatalf("line page = %+v", page)
	}
	// Seeded 1000 ml is 54 000 on sale for 48 000; the cheapest variant is 9 000 without a sale.
	if it := page.Items[0]; *it.PriceFrom != 9000 || it.SalePriceFrom == nil || *it.SalePriceFrom != 48000 {
		t.Errorf("prices = %v, %v", *it.PriceFrom, it.SalePriceFrom)
	}

	if _, err := repo.ApplySale(ctx, admin.SaleRequest{ProductIDs: productIDs(t, pool, "davines-minu-shampoo"), Percent: 10}); err != nil {
		t.Fatal(err)
	}
	page, err = repo.Products(ctx, admin.ProductFilter{OnSale: true})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range page.Items {
		names = append(names, it.Name)
	}
	slices.Sort(names)
	if !reflect.DeepEqual(names, []string{"Energizing Shampoo", "MINU Shampoo"}) {
		t.Errorf("on sale = %v", names)
	}
}
