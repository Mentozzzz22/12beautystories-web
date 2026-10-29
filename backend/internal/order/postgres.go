package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/pricing"
)

// Postgres implements Store.
type Postgres struct {
	db *pgxpool.Pool
}

// NewPostgres creates an order store.
func NewPostgres(db *pgxpool.Pool) *Postgres {
	return &Postgres{db: db}
}

// Variants returns purchasable variants (active variant, product and brand) by ID.
func (p *Postgres) Variants(ctx context.Context, ids []int64) (map[int64]Variant, error) {
	rows, err := p.db.Query(ctx, `
		SELECT v.id, p.brand_id, p.name, v.label, v.price, v.stock
		FROM product_variants v
		JOIN products p ON p.id = v.product_id AND p.is_active
		JOIN brands b ON b.id = p.brand_id AND b.is_active
		WHERE v.id = ANY($1) AND v.is_active`, ids)
	if err != nil {
		return nil, fmt.Errorf("query variants: %w", err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Variant])
	if err != nil {
		return nil, fmt.Errorf("collect variants: %w", err)
	}
	out := make(map[int64]Variant, len(list))
	for _, v := range list {
		out[v.ID] = v
	}
	return out, nil
}

// ActiveRules returns discount rules that are enabled and within their dates.
func (p *Postgres) ActiveRules(ctx context.Context) ([]pricing.Rule, error) {
	rows, err := p.db.Query(ctx, `
		SELECT id, name, brand_id, min_subtotal, kind, value
		FROM discount_rules
		WHERE is_active
		  AND (starts_at IS NULL OR starts_at <= now())
		  AND (ends_at IS NULL OR ends_at > now())
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query rules: %w", err)
	}
	rules, err := pgx.CollectRows(rows, pgx.RowToStructByPos[pricing.Rule])
	if err != nil {
		return nil, fmt.Errorf("collect rules: %w", err)
	}
	return rules, nil
}

// Promo returns a usable promo code or nil.
func (p *Postgres) Promo(ctx context.Context, code string) (*pricing.Promo, error) {
	var pr pricing.Promo
	err := p.db.QueryRow(ctx, `
		SELECT code, kind, value, min_subtotal, combinable_with_rules
		FROM promo_codes
		WHERE upper(code) = $1 AND is_active
		  AND (max_uses IS NULL OR used_count < max_uses)
		  AND (starts_at IS NULL OR starts_at <= now())
		  AND (ends_at IS NULL OR ends_at > now())`, code).
		Scan(&pr.Code, &pr.Kind, &pr.Value, &pr.MinSubtotal, &pr.CombinableWithRules)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query promo: %w", err)
	}
	return &pr, nil
}

// Create stores the order in one transaction, reserving stock and promo usage.
func (p *Postgres) Create(ctx context.Context, o Order) (Order, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, l := range o.Lines {
		tag, err := tx.Exec(ctx,
			`UPDATE product_variants SET stock = stock - $2 WHERE id = $1 AND stock >= $2`, l.VariantID, l.Quantity)
		if err != nil {
			return Order{}, fmt.Errorf("reserve stock: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return Order{}, fmt.Errorf("%w: %s %s", ErrInsufficientStock, l.ProductName, l.VariantLabel)
		}
	}

	if o.PromoCode != "" {
		tag, err := tx.Exec(ctx, `
			UPDATE promo_codes SET used_count = used_count + 1
			WHERE upper(code) = $1 AND (max_uses IS NULL OR used_count < max_uses)`, o.PromoCode)
		if err != nil {
			return Order{}, fmt.Errorf("use promo: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return Order{}, ErrInvalidPromo
		}
	}

	c := o.Customer
	customerID, err := upsertCustomer(ctx, tx, c)
	if err != nil {
		return Order{}, err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (customer_id, customer_name, phone, delivery_method, address, comment, promo_code,
		                    subtotal, discount_total, total, privacy_accepted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		RETURNING id, created_at`,
		customerID, c.Name, c.Phone, c.Delivery, c.Address, c.Comment, o.PromoCode,
		o.Quote.Subtotal, o.Quote.DiscountTotal, o.Quote.Total).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	batch := &pgx.Batch{}
	for _, l := range o.Lines {
		batch.Queue(`
			INSERT INTO order_items (order_id, variant_id, product_name, variant_label, unit_price, quantity, line_total)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			o.ID, l.VariantID, l.ProductName, l.VariantLabel, l.UnitPrice, l.Quantity, l.LineTotal)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return Order{}, fmt.Errorf("insert order items: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit: %w", err)
	}
	return o, nil
}

// upsertCustomer finds the customer by phone or creates one. The latest name wins; marketing consent
// can only be granted here, withdrawing it is done by the owner in the admin panel.
func upsertCustomer(ctx context.Context, tx pgx.Tx, c Customer) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO customers (phone, name, marketing_consent, marketing_consent_at)
		VALUES ($1, $2, $3::boolean, CASE WHEN $3::boolean THEN now() END)
		ON CONFLICT (phone) DO UPDATE SET
			name = EXCLUDED.name,
			marketing_consent = customers.marketing_consent OR EXCLUDED.marketing_consent,
			marketing_consent_at = CASE
				WHEN EXCLUDED.marketing_consent AND NOT customers.marketing_consent THEN now()
				ELSE customers.marketing_consent_at
			END
		RETURNING id`,
		c.Phone, c.Name, c.MarketingConsent).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert customer: %w", err)
	}
	return id, nil
}
