package pricing

import (
	"cmp"
	"fmt"
	"slices"
)

// BundleProduct is a product that can fill a slot.
type BundleProduct struct {
	ProductID int64
	LineID    int64
}

// BundleSlot is one position of a set. An empty Label accepts any variant of the products.
type BundleSlot struct {
	Group    int
	Name     string
	Label    string
	Products []BundleProduct
}

func (s BundleSlot) accepts(l Line) bool {
	if l.OnSale || (s.Label != "" && s.Label != l.Label) {
		return false
	}
	return slices.ContainsFunc(s.Products, func(p BundleProduct) bool { return p.ProductID == l.ProductID })
}

func (s BundleSlot) availableIn(lineID int64) bool {
	return slices.ContainsFunc(s.Products, func(p BundleProduct) bool { return p.LineID == lineID })
}

func (s BundleSlot) sameRequirement(o BundleSlot) bool {
	return s.Group == o.Group && s.Label == o.Label
}

// BundleTier is a set of slots that earns Percent off its items.
type BundleTier struct {
	OfferID int64
	Percent int64
	Slots   []BundleSlot
}

// BundleHint tells the customer which item raises the discount of a set already in the cart.
type BundleHint struct {
	Line    string `json:"line"`
	Item    string `json:"item"`
	Percent int64  `json:"percent"`
}

// bundleSet is a tier completed in one product line of the cart.
type bundleSet struct {
	tier   BundleTier
	lineID int64
}

// bundleDiscounts forms sets inside each product line, largest percent first. Every unit joins
// at most one set and only the units of a set are discounted. Sets with the same line and percent
// are reported as one discount.
func bundleDiscounts(lines []Line, tiers []BundleTier) ([]Discount, []BundleHint) {
	sorted := slices.Clone(tiers)
	slices.SortStableFunc(sorted, func(a, b BundleTier) int { return cmp.Compare(b.Percent, a.Percent) })

	var lineIDs []int64
	lineNames := make(map[int64]string)
	remaining := make([]int, len(lines))
	for i, l := range lines {
		remaining[i] = l.Quantity
		if _, seen := lineNames[l.LineID]; l.LineID != 0 && !seen {
			lineIDs = append(lineIDs, l.LineID)
			lineNames[l.LineID] = l.LineName
		}
	}

	var (
		labels []string
		bases  = make(map[string]int64)
		rates  = make(map[string]int64)
		sets   []bundleSet
	)
	for _, tier := range sorted {
		for _, lineID := range lineIDs {
			completed := false
			for {
				base, ok := takeSet(tier, lineID, lines, remaining)
				if !ok {
					break
				}
				completed = true
				label := fmt.Sprintf("Набор %s: −%d%%", lineNames[lineID], tier.Percent)
				if _, seen := bases[label]; !seen {
					labels = append(labels, label)
					rates[label] = tier.Percent
				}
				bases[label] += base
			}
			if completed {
				sets = append(sets, bundleSet{tier: tier, lineID: lineID})
			}
		}
	}

	var out []Discount
	for _, label := range labels {
		out = append(out, Discount{Source: SourceBundle, Label: label, Amount: discount(Percent, rates[label], bases[label])})
	}
	return out, bundleHints(sets, tiers, lineNames)
}

// takeSet reserves one unit per slot from the given product line and returns their total price.
// Nothing is reserved when any slot cannot be filled.
func takeSet(tier BundleTier, lineID int64, lines []Line, remaining []int) (int64, bool) {
	if len(tier.Slots) == 0 {
		return 0, false
	}
	picked := make([]int, 0, len(tier.Slots))
	var base int64
	for _, slot := range tier.Slots {
		i := -1
		for j, l := range lines {
			if remaining[j] > 0 && l.LineID == lineID && slot.accepts(l) {
				i = j
				break
			}
		}
		if i < 0 {
			for _, p := range picked {
				remaining[p]++
			}
			return 0, false
		}
		remaining[i]--
		picked = append(picked, i)
		base += lines[i].UnitPrice
	}
	return base, true
}

// bundleHints suggests the one extra item that upgrades a completed set to a better tier of its
// offer, if the product line has such an item at all. Only the best upgrade per line and item is kept.
func bundleHints(sets []bundleSet, tiers []BundleTier, lineNames map[int64]string) []BundleHint {
	var hints []BundleHint
	for _, set := range sets {
		for _, better := range tiers {
			if better.OfferID != set.tier.OfferID || better.Percent <= set.tier.Percent {
				continue
			}
			extra, ok := extraSlot(set.tier, better)
			if !ok || !extra.availableIn(set.lineID) {
				continue
			}
			line := lineNames[set.lineID]
			i := slices.IndexFunc(hints, func(h BundleHint) bool { return h.Line == line && h.Item == extra.Name })
			switch {
			case i < 0:
				hints = append(hints, BundleHint{Line: line, Item: extra.Name, Percent: better.Percent})
			case better.Percent > hints[i].Percent:
				hints[i].Percent = better.Percent
			}
		}
	}
	return hints
}

// extraSlot returns the only slot that better requires on top of everything in base.
func extraSlot(base, better BundleTier) (BundleSlot, bool) {
	if len(better.Slots) != len(base.Slots)+1 {
		return BundleSlot{}, false
	}
	var extra BundleSlot
	found := false
	for _, s := range better.Slots {
		if slices.ContainsFunc(base.Slots, s.sameRequirement) {
			continue
		}
		if found {
			return BundleSlot{}, false
		}
		extra, found = s, true
	}
	return extra, found
}
