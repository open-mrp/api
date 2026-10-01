package service

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/pricing"
)

// How much of a purchase order line is still to come is the ordered quantity less everything booked against it on the receiving order, stocked or not. Each receiving line can count in any unit of the item's group, so every figure here is taken into the ordered unit before it is compared or added; the dashboard's receiving screen does the same sum.

// progressByOrderLine groups receiving lines by the purchase order line they receive against, oldest first within each group.
func progressByOrderLine(lines []domain.ReceivingProgressLine) map[string][]domain.ReceivingProgressLine {
	groups := make(map[string][]domain.ReceivingProgressLine)
	for _, l := range lines {
		groups[l.OrderLineID] = append(groups[l.OrderLineID], l)
	}
	for _, g := range groups {
		sort.SliceStable(g, func(i, j int) bool {
			if !g[i].CreatedAt.Equal(g[j].CreatedAt) {
				return g[i].CreatedAt.Before(g[j].CreatedAt)
			}
			return g[i].ID < g[j].ID
		})
	}
	return groups
}

// inOrderedUnit is a receiving line's quantity counted in the unit its purchase order line was ordered in.
func inOrderedUnit(l domain.ReceivingProgressLine) decimal.Decimal {
	if l.UnitID == l.OrderedUnitID {
		return l.Value
	}
	return pricing.ConvertQuantity(l.Value, l.UnitRatio, l.OrderedUnitRatio)
}

// receiveTarget is the quantity, in the ordered unit, that the target line must hold for its purchase order line to be fully received: what was ordered, less what the order line's other receiving lines already hold.
//
// It reports false when there is nothing to do: the other lines already cover the order, or the target already holds at least that much. Finishing a line never takes away what was counted on it.
func receiveTarget(group []domain.ReceivingProgressLine, targetID string) (decimal.Decimal, bool) {
	var target *domain.ReceivingProgressLine
	others := decimal.Zero
	for i := range group {
		if group[i].ID == targetID {
			target = &group[i]
			continue
		}
		others = others.Add(inOrderedUnit(group[i]))
	}
	if target == nil {
		return decimal.Zero, false
	}

	want := target.OrderedValue.Sub(others)
	if !want.IsPositive() || !want.GreaterThan(inOrderedUnit(*target)) {
		return decimal.Zero, false
	}
	return want, true
}

// followUpLine is a purchase order line that needs a fresh unstocked line, at zero in the ordered unit, for the rest of it to be received against.
type followUpLine struct {
	OrderLineID string
	UnitID      string
}

// followUpLines names the purchase order lines a stocking left short: everything booked against them is now stocked, and together it is still less than was ordered.
//
// A purchase order line that still has an unstocked line needs nothing; that line is where the rest is received. The new line starts at zero, as a line does when an order is issued, so nothing is stocked until someone counts it.
func followUpLines(groups map[string][]domain.ReceivingProgressLine) []followUpLine {
	var out []followUpLine
	for orderLineID, group := range groups {
		if len(group) == 0 {
			continue
		}
		received := decimal.Zero
		open := false
		for _, l := range group {
			if l.StockedAt == nil {
				open = true
				break
			}
			received = received.Add(inOrderedUnit(l))
		}
		if open || !group[0].OrderedValue.GreaterThan(received) {
			continue
		}
		out = append(out, followUpLine{OrderLineID: orderLineID, UnitID: group[0].OrderedUnitID})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OrderLineID < out[j].OrderLineID })
	return out
}
