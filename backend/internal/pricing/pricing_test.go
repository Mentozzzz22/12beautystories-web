package pricing

import (
	"reflect"
	"testing"
)

func ptr[T any](v T) *T { return &v }

const (
	davines int64 = 1
	payot   int64 = 2
)

func TestCalculate(t *testing.T) {
	davinesRule := Rule{ID: 1, Name: "Davines 50k -10%", BrandID: ptr(davines), MinSubtotal: 50_000, Kind: Percent, Value: 10}

	tests := []struct {
		name  string
		lines []Line
		rules []Rule
		promo *Promo
		want  Quote
	}{
		{
			name: "empty cart",
			want: Quote{Discounts: []Discount{}},
		},
		{
			name:  "no discounts",
			lines: []Line{{BrandID: davines, UnitPrice: 12_000, Quantity: 2}, {BrandID: payot, UnitPrice: 9_000, Quantity: 1}},
			want:  Quote{Subtotal: 33_000, Discounts: []Discount{}, Total: 33_000},
		},
		{
			name:  "brand rule below threshold",
			lines: []Line{{BrandID: davines, UnitPrice: 20_000, Quantity: 2}, {BrandID: payot, UnitPrice: 20_000, Quantity: 1}},
			rules: []Rule{davinesRule},
			want:  Quote{Subtotal: 60_000, Discounts: []Discount{}, Total: 60_000},
		},
		{
			name:  "brand rule applies only to its brand",
			lines: []Line{{BrandID: davines, UnitPrice: 25_000, Quantity: 2}, {BrandID: payot, UnitPrice: 10_000, Quantity: 1}},
			rules: []Rule{davinesRule},
			want: Quote{
				Subtotal:      60_000,
				Discounts:     []Discount{{Source: SourceRule, Label: davinesRule.Name, Amount: 5_000}},
				DiscountTotal: 5_000,
				Total:         55_000,
			},
		},
		{
			name:  "best rule per scope wins",
			lines: []Line{{BrandID: davines, UnitPrice: 100_000, Quantity: 1}},
			rules: []Rule{
				davinesRule,
				{Name: "Davines 100k -15%", BrandID: ptr(davines), MinSubtotal: 100_000, Kind: Percent, Value: 15},
				{Name: "Cart -3000", MinSubtotal: 30_000, Kind: Fixed, Value: 3_000},
			},
			want: Quote{
				Subtotal: 100_000,
				Discounts: []Discount{
					{Source: SourceRule, Label: "Davines 100k -15%", Amount: 15_000},
					{Source: SourceRule, Label: "Cart -3000", Amount: 3_000},
				},
				DiscountTotal: 18_000,
				Total:         82_000,
			},
		},
		{
			name:  "fixed discount capped by subtotal",
			lines: []Line{{BrandID: payot, UnitPrice: 2_000, Quantity: 1}},
			promo: &Promo{Code: "GIFT5000", Kind: Fixed, Value: 5_000},
			want: Quote{
				Subtotal:      2_000,
				Discounts:     []Discount{{Source: SourcePromo, Label: "GIFT5000", Amount: 2_000}},
				DiscountTotal: 2_000,
				PromoStatus:   PromoApplied,
			},
		},
		{
			name:  "promo min subtotal not met",
			lines: []Line{{BrandID: payot, UnitPrice: 10_000, Quantity: 1}},
			promo: &Promo{Code: "WELCOME", Kind: Percent, Value: 5, MinSubtotal: 20_000},
			want:  Quote{Subtotal: 10_000, Discounts: []Discount{}, Total: 10_000, PromoStatus: PromoMinSubtotalNotMet},
		},
		{
			name:  "combinable promo applies after rules",
			lines: []Line{{BrandID: davines, UnitPrice: 50_000, Quantity: 1}},
			rules: []Rule{davinesRule},
			promo: &Promo{Code: "PLUS5", Kind: Percent, Value: 5, CombinableWithRules: true},
			want: Quote{
				Subtotal: 50_000,
				Discounts: []Discount{
					{Source: SourceRule, Label: davinesRule.Name, Amount: 5_000},
					{Source: SourcePromo, Label: "PLUS5", Amount: 2_250},
				},
				DiscountTotal: 7_250,
				Total:         42_750,
				PromoStatus:   PromoApplied,
			},
		},
		{
			name:  "non-combinable promo replaces weaker rules",
			lines: []Line{{BrandID: davines, UnitPrice: 50_000, Quantity: 1}},
			rules: []Rule{davinesRule},
			promo: &Promo{Code: "VIP20", Kind: Percent, Value: 20},
			want: Quote{
				Subtotal:      50_000,
				Discounts:     []Discount{{Source: SourcePromo, Label: "VIP20", Amount: 10_000}},
				DiscountTotal: 10_000,
				Total:         40_000,
				PromoStatus:   PromoApplied,
			},
		},
		{
			name:  "non-combinable promo loses to stronger rules",
			lines: []Line{{BrandID: davines, UnitPrice: 50_000, Quantity: 1}},
			rules: []Rule{davinesRule},
			promo: &Promo{Code: "HELLO", Kind: Fixed, Value: 1_000},
			want: Quote{
				Subtotal:      50_000,
				Discounts:     []Discount{{Source: SourceRule, Label: davinesRule.Name, Amount: 5_000}},
				DiscountTotal: 5_000,
				Total:         45_000,
				PromoStatus:   PromoNotBetter,
			},
		},
		{
			name:  "percent rounds down to whole tenge",
			lines: []Line{{BrandID: payot, UnitPrice: 9_999, Quantity: 1}},
			promo: &Promo{Code: "P7", Kind: Percent, Value: 7},
			want: Quote{
				Subtotal:      9_999,
				Discounts:     []Discount{{Source: SourcePromo, Label: "P7", Amount: 699}},
				DiscountTotal: 699,
				Total:         9_300,
				PromoStatus:   PromoApplied,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate(tt.lines, tt.rules, tt.promo)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Calculate() =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
