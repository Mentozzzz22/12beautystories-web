package order_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/database/dbtest"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/order"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/pricing"
)

func setup(t *testing.T) (*pgxpool.Pool, *order.Postgres) {
	t.Helper()
	pool := dbtest.New(t)
	dbtest.Seed(t, pool)
	return pool, order.NewPostgres(pool)
}

func variantID(t *testing.T, pool *pgxpool.Pool, productSlug, label string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		SELECT v.id FROM product_variants v JOIN products p ON p.id = v.product_id
		WHERE p.slug = $1 AND v.label = $2`, productSlug, label).Scan(&id)
	if err != nil {
		t.Fatalf("find variant %s %s: %v", productSlug, label, err)
	}
	return id
}

func stock(t *testing.T, pool *pgxpool.Pool, id int64) int {
	t.Helper()
	var s int
	if err := pool.QueryRow(context.Background(), `SELECT stock FROM product_variants WHERE id = $1`, id).Scan(&s); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	return s
}

func TestVariants(t *testing.T) {
	pool, store := setup(t)
	id := variantID(t, pool, "davines-energizing-shampoo", "250 ml")

	got, err := store.Variants(context.Background(), []int64{id, 999_999})
	if err != nil {
		t.Fatalf("Variants() error = %v", err)
	}
	v, ok := got[id]
	if len(got) != 1 || !ok {
		t.Fatalf("variants = %+v, want only %d", got, id)
	}
	if v.Price != 18000 || v.Stock != 8 || v.ProductName != "Energizing Shampoo" {
		t.Errorf("variant = %+v", v)
	}
}

func TestActiveRulesAndPromo(t *testing.T) {
	pool, store := setup(t)
	ctx := context.Background()

	rules, err := store.ActiveRules(ctx)
	if err != nil || len(rules) != 1 || rules[0].BrandID == nil {
		t.Fatalf("ActiveRules() = %+v, %v; want one brand rule", rules, err)
	}

	promo, err := store.Promo(ctx, "WELCOME5")
	if err != nil || promo == nil || promo.Kind != pricing.Percent {
		t.Fatalf("Promo() = %+v, %v", promo, err)
	}

	dbtest.Exec(t, pool, `UPDATE promo_codes SET max_uses = 1, used_count = 1`)
	dbtest.Exec(t, pool, `UPDATE discount_rules SET is_active = false`)

	if promo, _ := store.Promo(ctx, "WELCOME5"); promo != nil {
		t.Errorf("used-up promo returned: %+v", promo)
	}
	if rules, _ := store.ActiveRules(ctx); len(rules) != 0 {
		t.Errorf("disabled rule returned: %+v", rules)
	}
}

func testOrder(variantID int64, qty int, promo string) order.Order {
	return order.Order{
		Customer:  order.Customer{Name: "Test", Phone: "+77011234567", Delivery: order.DeliveryPickup},
		Lines:     []order.Line{{VariantID: variantID, ProductName: "Energizing Shampoo", VariantLabel: "1000 ml", UnitPrice: 48000, Quantity: qty, LineTotal: 48000 * int64(qty)}},
		PromoCode: promo,
		Quote:     pricing.Quote{Subtotal: 48000 * int64(qty), Total: 48000 * int64(qty)},
	}
}

func TestCreateReservesStockAndPromo(t *testing.T) {
	pool, store := setup(t)
	ctx := context.Background()
	id := variantID(t, pool, "davines-energizing-shampoo", "1000 ml")

	o, err := store.Create(ctx, testOrder(id, 2, "WELCOME5"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if o.ID == 0 || o.CreatedAt.IsZero() {
		t.Errorf("order not populated: %+v", o)
	}
	if s := stock(t, pool, id); s != 0 {
		t.Errorf("stock = %d, want 0", s)
	}

	var items, used int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM order_items WHERE order_id = $1`, o.ID).Scan(&items)
	_ = pool.QueryRow(ctx, `SELECT used_count FROM promo_codes WHERE code = 'WELCOME5'`).Scan(&used)
	if items != 1 || used != 1 {
		t.Errorf("items = %d, promo used = %d; want 1 and 1", items, used)
	}
}

func TestCreateRollsBackOnInsufficientStock(t *testing.T) {
	pool, store := setup(t)
	ctx := context.Background()
	id := variantID(t, pool, "davines-energizing-shampoo", "1000 ml")

	_, err := store.Create(ctx, testOrder(id, 3, "WELCOME5"))
	if !errors.Is(err, order.ErrInsufficientStock) {
		t.Fatalf("Create() error = %v, want ErrInsufficientStock", err)
	}

	var orders, used int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orders)
	_ = pool.QueryRow(ctx, `SELECT used_count FROM promo_codes WHERE code = 'WELCOME5'`).Scan(&used)
	if orders != 0 || used != 0 || stock(t, pool, id) != 2 {
		t.Errorf("transaction not rolled back: orders=%d promo used=%d", orders, used)
	}
}

func TestCreateConcurrentLastItem(t *testing.T) {
	pool, store := setup(t)
	id := variantID(t, pool, "davines-energizing-shampoo", "1000 ml")

	const buyers = 5
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
	)
	for range buyers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Create(context.Background(), testOrder(id, 1, ""))
			if err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			} else if !errors.Is(err, order.ErrInsufficientStock) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if succeeded != 2 {
		t.Errorf("succeeded = %d, want 2 (stock was 2)", succeeded)
	}
	if s := stock(t, pool, id); s != 0 {
		t.Errorf("stock = %d, want 0", s)
	}
}
