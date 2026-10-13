package scheduling

import (
	"sort"
	"strings"
)

// CommittedStock is downstream stock that has passed a branch point: dyed, cut or packed far enough that it can only become some of the finished goods its constraint item feeds. It still counts in the echelon total, but it can only answer for the demand of the finished goods it can still become — black socks cannot cover an order for blue ones.
type CommittedStock struct {
	ItemID string
	// OnHand is in the constraint item's unit, the same as the echelon it is part of.
	OnHand float64
	// FinishedItemIDs are the finished goods this stock can still become. Empty means it can become none of them, so it covers no demand at all.
	FinishedItemIDs []string
}

// FinishedDemand is one finished good's own demand at the constraint, used to decide which committed stock a week's demand can be served from.
type FinishedDemand struct {
	ItemID string
	// WeeklyForecast is this SKU's share of the pooled weekly forecast.
	WeeklyForecast float64
	// FirmByWeek is the order book placed for this SKU, dated the same way the pooled order book is.
	FirmByWeek []float64
}

// forWeek applies the same forecast-consumption rule as the pooled demand: the greater of the forecast and the orders on the book.
func (d FinishedDemand) forWeek(week int) float64 {
	if week < 0 || week >= len(d.FirmByWeek) {
		return d.WeeklyForecast
	}
	return max(d.WeeklyForecast, d.FirmByWeek[week])
}

// committedPool is committed stock grouped by the finished goods it can still become, so two stages committed to the same colorway are one supply.
type committedPool struct {
	finished []string
	onHand   float64
}

// newCommittedPools groups committed stock by the finished goods it can become, most specific first.
//
// Most specific first is the order demand draws on them: a finished black sock can only ever serve black, while dyed-black WIP might still become black in either size, so the narrow stock is used before the flexible one.
func newCommittedPools(stock []CommittedStock) []committedPool {
	byKey := map[string]*committedPool{}
	for _, s := range stock {
		finished := append([]string(nil), s.FinishedItemIDs...)
		sort.Strings(finished)
		key := strings.Join(finished, "|")
		if byKey[key] == nil {
			byKey[key] = &committedPool{finished: finished}
		}
		byKey[key].onHand += s.OnHand
	}

	out := make([]committedPool, 0, len(byKey))
	for _, pool := range byKey {
		out = append(out, *pool)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].finished) != len(out[j].finished) {
			return len(out[i].finished) < len(out[j].finished)
		}
		return strings.Join(out[i].finished, "|") < strings.Join(out[j].finished, "|")
	})
	return out
}

func committedTotal(pools []committedPool) float64 {
	var total float64
	for _, pool := range pools {
		total += pool.onHand
	}
	return total
}

// coverable is how much of the pools can answer for the given per-SKU needs, without consuming anything.
func coverable(pools []committedPool, need map[string]float64) float64 {
	remaining := make(map[string]float64, len(need))
	for id, units := range need {
		remaining[id] = units
	}

	var covered float64
	for _, pool := range pools {
		available := pool.onHand
		for _, id := range pool.finished {
			if available <= 0 {
				break
			}
			used := min(available, remaining[id])
			if used <= 0 {
				continue
			}
			remaining[id] -= used
			available -= used
			covered += used
		}
	}
	return covered
}

// consume draws each SKU's demand from the pools that can serve it, most specific first, and returns how much they served.
func consume(pools []committedPool, demand []FinishedDemand, week int) float64 {
	var served float64
	for _, d := range demand {
		remaining := d.forWeek(week)
		for i := range pools {
			if remaining <= 0 {
				break
			}
			if pools[i].onHand <= 0 || !containsSorted(pools[i].finished, d.ItemID) {
				continue
			}
			used := min(pools[i].onHand, remaining)
			pools[i].onHand -= used
			remaining -= used
			served += used
		}
	}
	return served
}

func containsSorted(sorted []string, id string) bool {
	i := sort.SearchStrings(sorted, id)
	return i < len(sorted) && sorted[i] == id
}

// buildFinishedDemand splits one constraint item's demand back out to the finished goods its committed stock can become.
//
// The forecast share is each SKU's slice of the pooled history over the same window, applied to the pooled weekly rate so overrides and the seasonal basis carry through. The order book is read per SKU from the requirements, so a week holding a large order for one colorway draws on that colorway's stock and no other.
func buildFinishedDemand(itemID string, policy ItemPolicy, committed []CommittedStock, pooled []MonthlyDemand, downstream []FinishedGood, firm FirmSchedule, horizonWeeks int) []FinishedDemand {
	if len(committed) == 0 {
		return nil
	}

	wanted := map[string]bool{}
	for _, s := range committed {
		for _, id := range s.FinishedItemIDs {
			wanted[id] = true
		}
	}
	if len(wanted) == 0 {
		return nil
	}

	var pooledTotal float64
	for _, m := range pooled {
		pooledTotal += m.Quantity
	}

	byID := make(map[string]*FinishedDemand, len(wanted))
	for id := range wanted {
		byID[id] = &FinishedDemand{ItemID: id}
	}

	if pooledTotal > 0 {
		for _, fg := range downstream {
			d, ok := byID[fg.ItemID]
			if !ok {
				continue
			}
			var total float64
			for _, m := range fg.Monthly {
				total += m.Quantity
			}
			d.WeeklyForecast = policy.WeeklyDemand * total / pooledTotal
		}
	}

	for _, req := range firm.Requirements {
		if req.ItemID != itemID {
			continue
		}
		d, ok := byID[req.FinishedItemID]
		if !ok || req.DueWeek < 0 || req.DueWeek >= horizonWeeks {
			continue
		}
		if d.FirmByWeek == nil {
			d.FirmByWeek = make([]float64, horizonWeeks)
		}
		d.FirmByWeek[req.DueWeek] += req.Units
	}

	out := make([]FinishedDemand, 0, len(byID))
	for _, d := range byID {
		out = append(out, *d)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ItemID < out[j].ItemID })
	return out
}
