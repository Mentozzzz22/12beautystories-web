// Package order quotes carts and places orders. Prices are always recalculated on the server.
package order

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/pricing"
)

// Errors returned to the client as validation failures.
var (
	ErrEmptyCart          = errors.New("cart is empty")
	ErrVariantUnavailable = errors.New("variant is unavailable")
	ErrInsufficientStock  = errors.New("not enough stock")
	ErrInvalidPromo       = errors.New("promo code is invalid or expired")
)

// ValidationError describes invalid checkout fields.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid fields: %d", len(e.Fields))
}

// MaxQuantity limits a single cart line.
const MaxQuantity = 99

// Item is a cart line sent by the client.
type Item struct {
	VariantID int64 `json:"variantId"`
	Quantity  int   `json:"quantity"`
}

// Variant is what the service needs to price and reserve a cart line.
type Variant struct {
	ID          int64
	BrandID     int64
	ProductName string
	Label       string
	Price       int64
	Stock       int
}

// DeliveryMethod is how the customer receives the order.
type DeliveryMethod string

const (
	DeliveryPickup  DeliveryMethod = "pickup"
	DeliveryCourier DeliveryMethod = "courier"
	DeliveryPost    DeliveryMethod = "post"
)

// Customer is the checkout form.
type Customer struct {
	Name     string         `json:"name"`
	Phone    string         `json:"phone"`
	Delivery DeliveryMethod `json:"delivery"`
	Address  string         `json:"address"`
	Comment  string         `json:"comment"`
}

// Line is a priced order line.
type Line struct {
	VariantID    int64  `json:"variantId"`
	ProductName  string `json:"productName"`
	VariantLabel string `json:"variantLabel"`
	UnitPrice    int64  `json:"unitPrice"`
	Quantity     int    `json:"quantity"`
	LineTotal    int64  `json:"lineTotal"`
}

// Order is a placed order.
type Order struct {
	ID        int64         `json:"id"`
	Number    string        `json:"number"`
	Customer  Customer      `json:"customer"`
	Lines     []Line        `json:"lines"`
	PromoCode string        `json:"promoCode,omitempty"`
	Quote     pricing.Quote `json:"quote"`
	CreatedAt time.Time     `json:"createdAt"`
}

// Number formats a human-friendly order number.
func Number(id int64) string {
	return fmt.Sprintf("BS-%05d", id)
}

// Store is the persistence the service depends on.
type Store interface {
	Variants(ctx context.Context, ids []int64) (map[int64]Variant, error)
	ActiveRules(ctx context.Context) ([]pricing.Rule, error)
	// Promo returns nil, nil when the code does not exist, is inactive, expired or used up.
	Promo(ctx context.Context, code string) (*pricing.Promo, error)
	// Create persists the order, decrements stock and counts promo usage atomically.
	// It returns ErrInsufficientStock or ErrInvalidPromo when a race lost.
	Create(ctx context.Context, o Order) (Order, error)
}

// Notifier tells the owner about a new order.
type Notifier interface {
	NewOrder(ctx context.Context, o Order) error
}

// Service implements quoting and checkout.
type Service struct {
	store    Store
	notifier Notifier
	onError  func(error)
}

// NewService creates a service. onError receives notification failures, which never fail checkout.
func NewService(store Store, notifier Notifier, onError func(error)) *Service {
	return &Service{store: store, notifier: notifier, onError: onError}
}

// QuoteResult is a priced cart.
type QuoteResult struct {
	Lines []Line        `json:"lines"`
	Quote pricing.Quote `json:"quote"`
}

// Quote prices the cart. An unknown promo is reported via ErrInvalidPromo.
func (s *Service) Quote(ctx context.Context, items []Item, promoCode string) (QuoteResult, error) {
	return s.price(ctx, items, promoCode, false)
}

// Place validates the checkout, prices it and stores the order.
func (s *Service) Place(ctx context.Context, items []Item, promoCode string, c Customer) (Order, error) {
	c, err := validateCustomer(c)
	if err != nil {
		return Order{}, err
	}
	res, err := s.price(ctx, items, promoCode, true)
	if err != nil {
		return Order{}, err
	}

	o := Order{Customer: c, Lines: res.Lines, Quote: res.Quote}
	if res.Quote.PromoStatus == pricing.PromoApplied {
		o.PromoCode = normalizeCode(promoCode)
	}
	o, err = s.store.Create(ctx, o)
	if err != nil {
		return Order{}, err
	}
	o.Number = Number(o.ID)

	if s.notifier != nil {
		go func() {
			nctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if err := s.notifier.NewOrder(nctx, o); err != nil && s.onError != nil {
				s.onError(fmt.Errorf("notify order %s: %w", o.Number, err))
			}
		}()
	}
	return o, nil
}

func (s *Service) price(ctx context.Context, items []Item, promoCode string, checkStock bool) (QuoteResult, error) {
	lines, priced, err := s.priceLines(ctx, items, checkStock)
	if err != nil {
		return QuoteResult{}, err
	}
	promo, err := s.promo(ctx, promoCode)
	if err != nil {
		return QuoteResult{}, err
	}
	rules, err := s.store.ActiveRules(ctx)
	if err != nil {
		return QuoteResult{}, fmt.Errorf("load rules: %w", err)
	}
	return QuoteResult{Lines: lines, Quote: pricing.Calculate(priced, rules, promo)}, nil
}

func (s *Service) promo(ctx context.Context, code string) (*pricing.Promo, error) {
	code = normalizeCode(code)
	if code == "" {
		return nil, nil
	}
	promo, err := s.store.Promo(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("load promo: %w", err)
	}
	if promo == nil {
		return nil, ErrInvalidPromo
	}
	return promo, nil
}

// priceLines merges duplicate variants and resolves current prices.
// With checkStock it fails when stock cannot cover the requested quantity.
func (s *Service) priceLines(ctx context.Context, items []Item, checkStock bool) ([]Line, []pricing.Line, error) {
	qty := make(map[int64]int)
	var ids []int64
	for _, it := range items {
		if it.Quantity <= 0 || it.VariantID <= 0 {
			continue
		}
		if _, ok := qty[it.VariantID]; !ok {
			ids = append(ids, it.VariantID)
		}
		qty[it.VariantID] = min(qty[it.VariantID]+it.Quantity, MaxQuantity)
	}
	if len(ids) == 0 {
		return nil, nil, ErrEmptyCart
	}

	variants, err := s.store.Variants(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("load variants: %w", err)
	}

	lines := make([]Line, 0, len(ids))
	priced := make([]pricing.Line, 0, len(ids))
	for _, id := range ids {
		v, ok := variants[id]
		if !ok {
			return nil, nil, fmt.Errorf("%w: %d", ErrVariantUnavailable, id)
		}
		q := qty[id]
		if checkStock && v.Stock < q {
			return nil, nil, fmt.Errorf("%w: %s %s", ErrInsufficientStock, v.ProductName, v.Label)
		}
		lines = append(lines, Line{
			VariantID: id, ProductName: v.ProductName, VariantLabel: v.Label,
			UnitPrice: v.Price, Quantity: q, LineTotal: v.Price * int64(q),
		})
		priced = append(priced, pricing.Line{VariantID: id, BrandID: v.BrandID, UnitPrice: v.Price, Quantity: q})
	}
	return lines, priced, nil
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

var nonDigits = regexp.MustCompile(`\D`)

// NormalizePhone converts Kazakhstan numbers ("8 701 123 45 67", "+7 (701) 123-45-67")
// to E.164 ("+77011234567"). It returns false for anything else.
func NormalizePhone(raw string) (string, bool) {
	d := nonDigits.ReplaceAllString(raw, "")
	switch {
	case len(d) == 11 && (d[0] == '8' || d[0] == '7'):
		d = d[1:]
	case len(d) == 10:
	default:
		return "", false
	}
	if d[0] != '7' && d[0] != '6' {
		return "", false
	}
	return "+7" + d, true
}

const (
	maxNameLen    = 100
	maxAddressLen = 300
	maxCommentLen = 1000
)

func validateCustomer(c Customer) (Customer, error) {
	fields := make(map[string]string)

	c.Name = strings.TrimSpace(c.Name)
	c.Address = strings.TrimSpace(c.Address)
	c.Comment = strings.TrimSpace(c.Comment)

	if c.Name == "" {
		fields["name"] = "required"
	} else if len([]rune(c.Name)) > maxNameLen {
		fields["name"] = "too_long"
	}
	if phone, ok := NormalizePhone(c.Phone); ok {
		c.Phone = phone
	} else {
		fields["phone"] = "invalid"
	}
	switch c.Delivery {
	case DeliveryPickup:
		c.Address = ""
	case DeliveryCourier, DeliveryPost:
		if c.Address == "" {
			fields["address"] = "required"
		}
	default:
		fields["delivery"] = "invalid"
	}
	if len([]rune(c.Address)) > maxAddressLen {
		fields["address"] = "too_long"
	}
	if len([]rune(c.Comment)) > maxCommentLen {
		fields["comment"] = "too_long"
	}

	if len(fields) > 0 {
		return c, &ValidationError{Fields: fields}
	}
	return c, nil
}
