// Package pricing calculates cart totals with automatic discount rules and promo codes.
// It is pure: callers load only valid (active, in date range) rules and promos.
package pricing

// Kind is how a discount value is interpreted.
type Kind string

const (
	// Percent takes Value percent (1..100) off the base.
	Percent Kind = "percent"
	// Fixed takes Value tenge off the base, never more than the base itself.
	Fixed Kind = "fixed"
)

// Line is a cart position. Prices are whole tenge.
type Line struct {
	VariantID int64
	BrandID   int64
	UnitPrice int64
	Quantity  int
}

// Rule is an automatic discount. A nil BrandID applies to the whole cart,
// otherwise only to lines of that brand, and MinSubtotal is checked against that brand's subtotal.
type Rule struct {
	ID          int64
	Name        string
	BrandID     *int64
	MinSubtotal int64
	Kind        Kind
	Value       int64
}

// Promo is a customer-entered code.
type Promo struct {
	Code                string
	Kind                Kind
	Value               int64
	MinSubtotal         int64
	CombinableWithRules bool
}

// PromoStatus explains what happened to the entered promo code.
type PromoStatus string

const (
	PromoNone              PromoStatus = ""
	PromoApplied           PromoStatus = "applied"
	PromoMinSubtotalNotMet PromoStatus = "min_subtotal_not_met"
	// PromoNotBetter means the code cannot combine with rules and the rules give a larger discount.
	PromoNotBetter PromoStatus = "not_better"
)

// Source tells where a discount came from.
type Source string

const (
	SourceRule  Source = "rule"
	SourcePromo Source = "promo"
)

// Discount is one applied discount.
type Discount struct {
	Source Source `json:"source"`
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

// Quote is the calculated cart.
type Quote struct {
	Subtotal      int64       `json:"subtotal"`
	Discounts     []Discount  `json:"discounts"`
	DiscountTotal int64       `json:"discountTotal"`
	Total         int64       `json:"total"`
	PromoStatus   PromoStatus `json:"promoStatus,omitempty"`
}

// Calculate applies the best rule per scope (each brand and the whole cart), then the promo code.
// A promo that cannot combine with rules replaces them only when it gives a bigger discount.
func Calculate(lines []Line, rules []Rule, promo *Promo) Quote {
	var subtotal int64
	brandSubtotals := make(map[int64]int64)
	for _, l := range lines {
		amount := l.UnitPrice * int64(l.Quantity)
		subtotal += amount
		brandSubtotals[l.BrandID] += amount
	}

	ruleDiscounts := bestRules(rules, subtotal, brandSubtotals)
	var ruleTotal int64
	for _, d := range ruleDiscounts {
		ruleTotal += d.Amount
	}
	ruleTotal = min(ruleTotal, subtotal)

	q := Quote{Subtotal: subtotal, Discounts: ruleDiscounts}

	if promo != nil {
		switch {
		case subtotal < promo.MinSubtotal:
			q.PromoStatus = PromoMinSubtotalNotMet
		case promo.CombinableWithRules:
			amount := discount(promo.Kind, promo.Value, subtotal-ruleTotal)
			q.Discounts = append(q.Discounts, Discount{Source: SourcePromo, Label: promo.Code, Amount: amount})
			q.PromoStatus = PromoApplied
		default:
			amount := discount(promo.Kind, promo.Value, subtotal)
			if amount > ruleTotal {
				q.Discounts = []Discount{{Source: SourcePromo, Label: promo.Code, Amount: amount}}
				q.PromoStatus = PromoApplied
			} else {
				q.PromoStatus = PromoNotBetter
			}
		}
	}

	for _, d := range q.Discounts {
		q.DiscountTotal += d.Amount
	}
	q.DiscountTotal = min(q.DiscountTotal, subtotal)
	q.Total = subtotal - q.DiscountTotal
	if q.Discounts == nil {
		q.Discounts = []Discount{}
	}
	return q
}

// bestRules picks the largest qualifying rule per scope, keeping input order for stable output.
func bestRules(rules []Rule, subtotal int64, brandSubtotals map[int64]int64) []Discount {
	const cartScope int64 = 0 // brand IDs are identity values starting at 1

	best := make(map[int64]Discount)
	var order []int64
	for _, r := range rules {
		scope, base := cartScope, subtotal
		if r.BrandID != nil {
			scope, base = *r.BrandID, brandSubtotals[*r.BrandID]
		}
		if base == 0 || base < r.MinSubtotal {
			continue
		}
		amount := discount(r.Kind, r.Value, base)
		cur, seen := best[scope]
		if !seen {
			order = append(order, scope)
		}
		if !seen || amount > cur.Amount {
			best[scope] = Discount{Source: SourceRule, Label: r.Name, Amount: amount}
		}
	}

	out := make([]Discount, 0, len(order))
	for _, scope := range order {
		out = append(out, best[scope])
	}
	return out
}

func discount(kind Kind, value, base int64) int64 {
	if base <= 0 || value <= 0 {
		return 0
	}
	switch kind {
	case Percent:
		return base * min(value, 100) / 100
	case Fixed:
		return min(value, base)
	default:
		return 0
	}
}
