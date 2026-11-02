package pricing

import (
	"reflect"
	"testing"
)

const (
	platinum int64 = 10
	premium  int64 = 20
	headSpa  int64 = 30

	shampooPlatinum int64 = 1
	treatPlatinum   int64 = 2
	oilPlatinum     int64 = 3
	shampooPremium  int64 = 4
	treatPremium    int64 = 5
	sprayPremium    int64 = 6
	shampooHeadSpa  int64 = 7
	treatHeadSpa    int64 = 8
)

// posterTiers mirrors the real offer: shampoo + treatment, optionally with a leave-in item.
func posterTiers() []BundleTier {
	shampoos := []BundleProduct{{shampooPlatinum, platinum}, {shampooPremium, premium}, {shampooHeadSpa, headSpa}}
	treatments := []BundleProduct{{treatPlatinum, platinum}, {treatPremium, premium}, {treatHeadSpa, headSpa}}
	leaveIns := []BundleProduct{{oilPlatinum, platinum}, {sprayPremium, premium}}

	tier := func(percent int64, volume string, withLeaveIn bool) BundleTier {
		slots := []BundleSlot{
			{Group: 0, Name: "шампунь", Label: volume, Products: shampoos},
			{Group: 1, Name: "тритмент", Label: volume, Products: treatments},
		}
		if withLeaveIn {
			slots = append(slots, BundleSlot{Group: 2, Name: "несмываемый уход", Products: leaveIns})
		}
		return BundleTier{OfferID: 1, Percent: percent, Slots: slots}
	}
	return []BundleTier{tier(15, "200", false), tier(20, "200", true), tier(20, "400", false), tier(25, "400", true)}
}

func item(productID, lineID int64, lineName, label string, price int64, qty int) Line {
	return Line{
		ProductID: productID, LineID: lineID, LineName: lineName, Label: label,
		VariantID: productID*100 + int64(len(label)), UnitPrice: price, Quantity: qty,
	}
}

func platinumShampoo(label string, qty int) Line {
	prices := map[string]int64{"200": 27_900, "400": 44_900, "40": 13_500}
	return item(shampooPlatinum, platinum, "PLATINUM", label, prices[label], qty)
}

func platinumTreatment(label string, qty int) Line {
	prices := map[string]int64{"200": 29_900, "400": 49_900}
	return item(treatPlatinum, platinum, "PLATINUM", label, prices[label], qty)
}

func platinumOil(qty int) Line { return item(oilPlatinum, platinum, "PLATINUM", "", 27_900, qty) }

func TestBundleDiscounts(t *testing.T) {
	bundle := func(label string, amount int64) Discount {
		return Discount{Source: SourceBundle, Label: label, Amount: amount}
	}

	tests := []struct {
		name  string
		lines []Line
		want  []Discount
		hints []BundleHint
	}{
		{
			name:  "shampoo alone gets nothing",
			lines: []Line{platinumShampoo("200", 1)},
		},
		{
			name:  "200 ml pair: 15%",
			lines: []Line{platinumShampoo("200", 1), platinumTreatment("200", 1)},
			want:  []Discount{bundle("Набор PLATINUM: −15%", 8_670)},
			hints: []BundleHint{{Line: "PLATINUM", Item: "несмываемый уход", Percent: 20}},
		},
		{
			name:  "200 ml pair with leave-in: 20%",
			lines: []Line{platinumShampoo("200", 1), platinumTreatment("200", 1), platinumOil(1)},
			want:  []Discount{bundle("Набор PLATINUM: −20%", 17_140)},
		},
		{
			name:  "400 ml pair: 20%",
			lines: []Line{platinumShampoo("400", 1), platinumTreatment("400", 1)},
			want:  []Discount{bundle("Набор PLATINUM: −20%", 18_960)},
			hints: []BundleHint{{Line: "PLATINUM", Item: "несмываемый уход", Percent: 25}},
		},
		{
			name:  "400 ml pair with leave-in: 25%",
			lines: []Line{platinumShampoo("400", 1), platinumTreatment("400", 1), platinumOil(1)},
			want:  []Discount{bundle("Набор PLATINUM: −25%", 30_675)},
		},
		{
			name:  "mixed volumes do not form a set",
			lines: []Line{platinumShampoo("200", 1), platinumTreatment("400", 1)},
		},
		{
			name:  "40 ml is not part of the offer",
			lines: []Line{platinumShampoo("40", 1), platinumTreatment("200", 1)},
		},
		{
			name: "products of different lines do not form a set",
			lines: []Line{
				platinumShampoo("200", 1),
				item(treatPremium, premium, "PREMIUM", "200", 31_900, 1),
			},
		},
		{
			name:  "two sets, extra shampoo stays at full price",
			lines: []Line{platinumShampoo("200", 3), platinumTreatment("200", 2)},
			want:  []Discount{bundle("Набор PLATINUM: −15%", 17_340)},
			hints: []BundleHint{{Line: "PLATINUM", Item: "несмываемый уход", Percent: 20}},
		},
		{
			name: "the only leave-in goes to the bigger discount",
			lines: []Line{
				platinumShampoo("200", 1), platinumTreatment("200", 1),
				platinumShampoo("400", 1), platinumTreatment("400", 1),
				platinumOil(1),
			},
			want:  []Discount{bundle("Набор PLATINUM: −25%", 30_675), bundle("Набор PLATINUM: −15%", 8_670)},
			hints: []BundleHint{{Line: "PLATINUM", Item: "несмываемый уход", Percent: 20}},
		},
		{
			name: "each line is counted separately",
			lines: []Line{
				platinumShampoo("200", 1), platinumTreatment("200", 1),
				item(shampooPremium, premium, "PREMIUM", "400", 49_900, 1),
				item(treatPremium, premium, "PREMIUM", "400", 54_900, 1),
				item(sprayPremium, premium, "PREMIUM", "", 29_900, 1),
			},
			want:  []Discount{bundle("Набор PREMIUM: −25%", 33_675), bundle("Набор PLATINUM: −15%", 8_670)},
			hints: []BundleHint{{Line: "PLATINUM", Item: "несмываемый уход", Percent: 20}},
		},
		{
			name: "no hint when the line has no leave-in",
			lines: []Line{
				item(shampooHeadSpa, headSpa, "HEAD SPA", "200", 30_000, 1),
				item(treatHeadSpa, headSpa, "HEAD SPA", "200", 30_000, 1),
			},
			want: []Discount{bundle("Набор HEAD SPA: −15%", 9_000)},
		},
		{
			name: "reduced price does not join a set",
			lines: []Line{
				platinumTreatment("200", 1),
				func() Line { l := platinumShampoo("200", 1); l.OnSale = true; return l }(),
			},
		},
		{
			name: "product without a line does not join a set",
			lines: []Line{
				item(shampooPlatinum, 0, "", "200", 27_900, 1),
				item(treatPlatinum, 0, "", "200", 29_900, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, hints := bundleDiscounts(tt.lines, posterTiers())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("discounts = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(hints, tt.hints) {
				t.Errorf("hints = %+v, want %+v", hints, tt.hints)
			}
		})
	}
}

func TestCalculateWithBundles(t *testing.T) {
	pair := []Line{platinumShampoo("200", 1), platinumTreatment("200", 1)} // 57 800, bundle 8 670
	bundleDiscount := Discount{Source: SourceBundle, Label: "Набор PLATINUM: −15%", Amount: 8_670}
	hints := []BundleHint{{Line: "PLATINUM", Item: "несмываемый уход", Percent: 20}}
	rule := func(percent int64) []Rule {
		return []Rule{{ID: 1, Name: "Правило", Kind: Percent, Value: percent}}
	}

	tests := []struct {
		name  string
		rules []Rule
		promo *Promo
		want  Quote
	}{
		{
			name: "bundle alone",
			want: Quote{Subtotal: 57_800, Discounts: []Discount{bundleDiscount}, DiscountTotal: 8_670, Total: 49_130, BundleHints: hints},
		},
		{
			name:  "smaller rule is not added to the bundle",
			rules: rule(5),
			want:  Quote{Subtotal: 57_800, Discounts: []Discount{bundleDiscount}, DiscountTotal: 8_670, Total: 49_130, BundleHints: hints},
		},
		{
			name:  "bigger rule wins over the bundle",
			rules: rule(20),
			want: Quote{
				Subtotal: 57_800, Discounts: []Discount{{Source: SourceRule, Label: "Правило", Amount: 11_560}},
				DiscountTotal: 11_560, Total: 46_240,
			},
		},
		{
			name:  "smaller promo does not replace the bundle",
			promo: &Promo{Code: "P5", Kind: Percent, Value: 5},
			want: Quote{
				Subtotal: 57_800, Discounts: []Discount{bundleDiscount}, DiscountTotal: 8_670, Total: 49_130,
				PromoStatus: PromoNotBetter, BundleHints: hints,
			},
		},
		{
			name:  "bigger promo replaces the bundle and its hint",
			promo: &Promo{Code: "P30", Kind: Percent, Value: 30},
			want: Quote{
				Subtotal: 57_800, Discounts: []Discount{{Source: SourcePromo, Label: "P30", Amount: 17_340}},
				DiscountTotal: 17_340, Total: 40_460, PromoStatus: PromoApplied,
			},
		},
		{
			name:  "combinable promo applies after the bundle",
			promo: &Promo{Code: "C10", Kind: Percent, Value: 10, CombinableWithRules: true},
			want: Quote{
				Subtotal:      57_800,
				Discounts:     []Discount{bundleDiscount, {Source: SourcePromo, Label: "C10", Amount: 4_913}},
				DiscountTotal: 13_583, Total: 44_217, PromoStatus: PromoApplied, BundleHints: hints,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate(pair, tt.rules, posterTiers(), tt.promo)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Calculate() =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
