package order

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/pricing"
)

type fakeStore struct {
	variants map[int64]Variant
	rules    []pricing.Rule
	promos   map[string]*pricing.Promo
	created  []Order
}

func (f *fakeStore) Variants(_ context.Context, ids []int64) (map[int64]Variant, error) {
	out := make(map[int64]Variant)
	for _, id := range ids {
		if v, ok := f.variants[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

func (f *fakeStore) ActiveRules(context.Context) ([]pricing.Rule, error) { return f.rules, nil }

func (f *fakeStore) Promo(_ context.Context, code string) (*pricing.Promo, error) {
	return f.promos[code], nil
}

func (f *fakeStore) Create(_ context.Context, o Order) (Order, error) {
	o.ID = int64(len(f.created) + 1)
	o.CreatedAt = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	f.created = append(f.created, o)
	return o, nil
}

type fakeNotifier struct {
	mu   sync.Mutex
	got  []Order
	done chan struct{}
}

func (n *fakeNotifier) NewOrder(_ context.Context, o Order) error {
	n.mu.Lock()
	n.got = append(n.got, o)
	n.mu.Unlock()
	close(n.done)
	return nil
}

func newStore() *fakeStore {
	davines := int64(1)
	return &fakeStore{
		variants: map[int64]Variant{
			10: {ID: 10, BrandID: 1, ProductName: "Energizing Shampoo", Label: "250 ml", Price: 25_000, Stock: 5},
			11: {ID: 11, BrandID: 1, ProductName: "Energizing Shampoo", Label: "1000 ml", Price: 60_000, Stock: 1},
			20: {ID: 20, BrandID: 2, ProductName: "Hydra Cream", Label: "", Price: 15_000, Stock: 3},
		},
		rules: []pricing.Rule{{ID: 1, Name: "Davines 50k -10%", BrandID: &davines, MinSubtotal: 50_000, Kind: pricing.Percent, Value: 10}},
		promos: map[string]*pricing.Promo{
			"WELCOME": {Code: "WELCOME", Kind: pricing.Percent, Value: 5, CombinableWithRules: true},
		},
	}
}

func validCustomer() Customer {
	return Customer{Name: " Aigerim ", Phone: "8 (701) 123-45-67", Delivery: DeliveryCourier, Address: "Astana, Mangilik El 10"}
}

func TestQuoteMergesDuplicatesAndAppliesRules(t *testing.T) {
	svc := NewService(newStore(), nil, nil)

	res, err := svc.Quote(context.Background(), []Item{{VariantID: 10, Quantity: 1}, {VariantID: 10, Quantity: 1}, {VariantID: 20, Quantity: 1}}, "")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if len(res.Lines) != 2 || res.Lines[0].Quantity != 2 {
		t.Fatalf("lines = %+v, want merged variant 10 x2 and variant 20", res.Lines)
	}
	if res.Quote.Subtotal != 65_000 || res.Quote.Total != 60_000 {
		t.Errorf("quote = %+v, want subtotal 65000 and total 60000", res.Quote)
	}
}

func TestQuoteErrors(t *testing.T) {
	svc := NewService(newStore(), nil, nil)
	ctx := context.Background()

	tests := []struct {
		name  string
		items []Item
		promo string
		want  error
	}{
		{name: "empty cart", items: nil, want: ErrEmptyCart},
		{name: "only invalid quantities", items: []Item{{VariantID: 10, Quantity: 0}}, want: ErrEmptyCart},
		{name: "unknown variant", items: []Item{{VariantID: 99, Quantity: 1}}, want: ErrVariantUnavailable},
		{name: "unknown promo", items: []Item{{VariantID: 10, Quantity: 1}}, promo: "NOPE", want: ErrInvalidPromo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Quote(ctx, tt.items, tt.promo); !errors.Is(err, tt.want) {
				t.Errorf("Quote() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestQuoteDoesNotCheckStock(t *testing.T) {
	svc := NewService(newStore(), nil, nil)
	if _, err := svc.Quote(context.Background(), []Item{{VariantID: 11, Quantity: 3}}, ""); err != nil {
		t.Errorf("Quote() error = %v, want nil", err)
	}
}

func TestPlace(t *testing.T) {
	store := newStore()
	notifier := &fakeNotifier{done: make(chan struct{})}
	svc := NewService(store, notifier, func(err error) { t.Errorf("notify error: %v", err) })

	o, err := svc.Place(context.Background(), []Item{{VariantID: 10, Quantity: 2}}, " welcome ", validCustomer())
	if err != nil {
		t.Fatalf("Place() error = %v", err)
	}

	if o.Number != "BS-00001" {
		t.Errorf("Number = %q, want BS-00001", o.Number)
	}
	if o.Customer.Phone != "+77011234567" || o.Customer.Name != "Aigerim" {
		t.Errorf("customer not normalized: %+v", o.Customer)
	}
	if o.PromoCode != "WELCOME" {
		t.Errorf("PromoCode = %q, want WELCOME", o.PromoCode)
	}
	// 50 000 subtotal: rule -5 000, then promo 5% of 45 000 = -2 250.
	if o.Quote.Total != 42_750 {
		t.Errorf("Total = %d, want 42750", o.Quote.Total)
	}
	if len(store.created) != 1 {
		t.Fatalf("created %d orders, want 1", len(store.created))
	}

	select {
	case <-notifier.done:
	case <-time.After(time.Second):
		t.Fatal("notifier was not called")
	}
}

func TestPlaceInsufficientStock(t *testing.T) {
	store := newStore()
	svc := NewService(store, nil, nil)

	_, err := svc.Place(context.Background(), []Item{{VariantID: 11, Quantity: 2}}, "", validCustomer())
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Place() error = %v, want ErrInsufficientStock", err)
	}
	if len(store.created) != 0 {
		t.Error("order must not be created")
	}
}

func TestPlaceValidation(t *testing.T) {
	svc := NewService(newStore(), nil, nil)

	_, err := svc.Place(context.Background(), []Item{{VariantID: 10, Quantity: 1}}, "",
		Customer{Name: "", Phone: "123", Delivery: DeliveryCourier})

	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("Place() error = %v, want ValidationError", err)
	}
	for _, field := range []string{"name", "phone", "address"} {
		if _, ok := verr.Fields[field]; !ok {
			t.Errorf("missing validation error for %q in %v", field, verr.Fields)
		}
	}
}

func TestPlacePickupClearsAddress(t *testing.T) {
	svc := NewService(newStore(), nil, nil)
	c := validCustomer()
	c.Delivery = DeliveryPickup

	o, err := svc.Place(context.Background(), []Item{{VariantID: 20, Quantity: 1}}, "", c)
	if err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	if o.Customer.Address != "" {
		t.Errorf("Address = %q, want empty for pickup", o.Customer.Address)
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"+7 701 123 45 67", "+77011234567", true},
		{"8(701)123-45-67", "+77011234567", true},
		{"7011234567", "+77011234567", true},
		{"87171234567", "+77171234567", true},
		{"+1 202 555 0143", "", false},
		{"12345", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizePhone(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
