//go:build plans

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The production cost corpus is a plant's last two years of scans: 60,000 batches across six stations in
// four departments, one of them a single quiet station, three in four scanned and nearly all at one of 120
// production steps, a tenth recording waste and some seconds, a few counted in dozens. The steps make
// chains of parts and twenty finished goods on two product lines, and the account keeps four times as many
// steps that scan nothing, as a plant accumulates them. The items sit in seven busy categories and one held
// by a single rare part with three batches.
const (
	planCostAccount   = "ac_plancost"
	planCostBatches   = 60_000
	planCostSteps     = 120
	planCostIdleSteps = 480
	planCostParts     = 100
	planCostProducts  = 20
	planCostMaterials = 20
	planCostSpan      = 2 * 365 * 24 * time.Hour

	// planCostCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planCostCorpusVersion = "Plan Production Costs v2"

	// Items: parts first, then finished goods, then materials, then the rare part.
	planCostFirstProduct  = planCostParts
	planCostFirstMaterial = planCostFirstProduct + planCostProducts
	planCostRareItem      = planCostFirstMaterial + planCostMaterials
	planCostItems         = planCostRareItem + 1
	planCostRareCategory  = 7
)

var (
	planCostOrigin = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
	// planCostStationShares is each station's share of batches, in thousandths; the rest are on no station.
	planCostStationShares = []int{370, 240, 170, 160, 40, 15}
	// planCostStationDepartment is each station's department: the fourth holds only the quietest station.
	planCostStationDepartment = []int{0, 1, 2, 0, 1, 3}
	planCostRareItemBatches   = map[int]bool{500: true, 30_000: true, 59_500: true}
)

func planCostItemID(n int) string       { return fmt.Sprintf("0000%04d-plan-4cit-8000-%012d", n, n) }
func planCostStepID(s int) string       { return fmt.Sprintf("0000%04d-plan-4cps-8000-%012d", s, s) }
func planCostStationID(s int) string    { return fmt.Sprintf("0000%04d-plan-4css-8000-%012d", s, s) }
func planCostDepartmentID(d int) string { return fmt.Sprintf("dp_plancost_%d", d) }
func planCostCategoryID(c int) string   { return fmt.Sprintf("ic_plancost_%d", c) }
func planCostLineID(l int) string       { return fmt.Sprintf("pl_plancost_%d", l) }

const planCostEmptyDepartment = "dp_plancost_empty"

func planCostCreatedAt(i int) time.Time {
	return planCostOrigin.Add(time.Duration(i) * (planCostSpan / planCostBatches))
}

func planCostStation(i int) int {
	slot := (i * 7) % 1000
	for s, share := range planCostStationShares {
		if slot < share {
			return s
		}
		slot -= share
	}
	return -1
}

// planCostItem gives one finished good a tenth of the batches and spreads the rest over everything a step makes.
func planCostItem(i int) int {
	switch {
	case planCostRareItemBatches[i]:
		return planCostRareItem
	case i%10 == 1:
		return planCostFirstProduct
	}
	made := planHash(i, 41) % (planCostParts + planCostProducts)
	if made < planCostParts {
		return made
	}
	return planCostFirstProduct + made - planCostParts
}

// planCostStepFor is the step that makes item n; the rare part is made at the first step but no step produces it.
func planCostStepFor(n int) int {
	if n >= planCostFirstProduct && n < planCostFirstMaterial {
		return planCostParts + n - planCostFirstProduct
	}
	if n == planCostRareItem {
		return 0
	}
	return n
}

// planCostStepItem is what step s produces; an idle step makes a part again.
func planCostStepItem(s int) int {
	switch {
	case s < planCostParts:
		return s
	case s < planCostSteps:
		return planCostFirstProduct + s - planCostParts
	}
	return s % planCostParts
}

func planCostCategory(n int) int {
	if n == planCostRareItem {
		return planCostRareCategory
	}
	return n % planCostRareCategory
}

var planCostCorpusOnce sync.Once

func ensureProductionCostCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planCostCorpusOnce.Do(func() {
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planCostAccount).Scan(&version)
		if version == planCostCorpusVersion {
			return
		}
		t.Logf("seeding the production cost plan corpus (%d batches); it is kept for later runs", planCostBatches)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM batch WHERE account_id = ?", planCostAccount)
		exec("DELETE FROM quantity WHERE id LIKE 'qu\\_plancost\\_%'")
		exec("DELETE FROM quantity WHERE id LIKE 'prd\\_plancost\\_%' OR id LIKE 'csm\\_plancost\\_%'")
		exec("DELETE FROM rate WHERE id LIKE 'rt\\_plancost\\_%'")
		exec("DELETE FROM production WHERE id LIKE 'prd\\_plancost\\_%'")
		exec("DELETE FROM consumption WHERE id LIKE 'csm\\_plancost\\_%'")
		exec("DELETE FROM production_step WHERE account_id = ?", planCostAccount)
		exec("DELETE FROM product WHERE id LIKE 'pd\\_plancost\\_%'")
		exec("DELETE FROM product_line WHERE account_id = ?", planCostAccount)
		exec("DELETE FROM item WHERE account_id = ?", planCostAccount)
		exec("DELETE FROM item_category WHERE account_id = ?", planCostAccount)
		exec("DELETE FROM scanning_station WHERE account_id = ?", planCostAccount)
		exec("DELETE FROM department WHERE account_id = ?", planCostAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, 'pending', 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = 'pending'`, planCostAccount)
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_plancost', 'plancost each', 'un_plancost', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3)),
		             ('un_plancost_dz', 'plancost dozen', 'un_plancost_dz', 'quantity', NULL, 12, 1, 0, NOW(3), NOW(3))`)

		for d := range 4 {
			exec("INSERT INTO department (id, name, account_id) VALUES (?, ?, ?)", planCostDepartmentID(d), fmt.Sprintf("Plan Cost Dept %d", d), planCostAccount)
		}
		exec("INSERT INTO department (id, name, account_id) VALUES (?, 'Plan Cost Empty Dept', ?)", planCostEmptyDepartment, planCostAccount)
		for s, d := range planCostStationDepartment {
			exec(`INSERT INTO scanning_station (id, name, department_id, account_id, scanning_station_type_code) VALUES (?, ?, ?, ?, 'production')`,
				planCostStationID(s), fmt.Sprintf("Plan Cost Station %d", s), planCostDepartmentID(d), planCostAccount)
		}
		for c := range planCostRareCategory + 1 {
			exec(`INSERT INTO item_category (id, name, account_id, item_category_type_code, unit_group_id) VALUES (?, ?, ?, 'product_category', 'ungp_plancost')`,
				planCostCategoryID(c), fmt.Sprintf("Plan Cost Category %d", c), planCostAccount)
		}

		var itemVals, rateVals []string
		var itemArgs, rateArgs []any
		for n := range planCostItems {
			itemType := "part"
			switch {
			case n >= planCostFirstProduct && n < planCostFirstMaterial:
				itemType = "product"
			case n >= planCostFirstMaterial && n < planCostRareItem:
				itemType = "material"
			}
			unitCost := fmt.Sprintf("rt_plancost_uc_%04d", n)
			itemVals = append(itemVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			itemArgs = append(itemArgs, planCostItemID(n), fmt.Sprintf("PLANCOST-%04d", n), fmt.Sprintf("uv_plancost_%04d", n),
				fmt.Sprintf("br_plancost_%04d", n), unitCost, planCostAccount, itemType, planCostCategoryID(planCostCategory(n)), planCostOrigin, planCostOrigin)
			rateVals = append(rateVals, "(?, '0.5', 'dollar', 'un_plancost')")
			rateArgs = append(rateArgs, unitCost)
		}
		exec(`INSERT INTO item (id, sku, unit_value_id, burn_rate_id, unit_cost_id, account_id, item_type_code, item_category_id,
		      created_at, updated_at) VALUES `+strings.Join(itemVals, ","), itemArgs...)

		for l := range 2 {
			exec("INSERT INTO product_line (id, name, account_id, unit_group_id) VALUES (?, ?, ?, 'ungp_plancost')", planCostLineID(l), fmt.Sprintf("Plan Cost Line %d", l), planCostAccount)
		}
		for p := range planCostProducts {
			exec("INSERT INTO product (id, item_id, product_type_code, product_line_id) VALUES (?, ?, 'sale', ?)",
				fmt.Sprintf("pd_plancost_%02d", p), planCostItemID(planCostFirstProduct+p), planCostLineID(p/10))
		}

		var stepVals, prodVals, consVals, qVals []string
		var stepArgs, prodArgs, consArgs, qArgs []any
		quantity := func(id, value, unit string) {
			qVals = append(qVals, "(?, ?, ?)")
			qArgs = append(qArgs, id, value, unit)
		}
		consume := func(s, k, item int, value string) {
			id := fmt.Sprintf("csm_plancost_%03d_%d", s, k)
			quantity(id+"_q", value, "un_plancost")
			quantity(id+"_w", "0.1", "un_plancost")
			consVals = append(consVals, "(?, ?, ?, ?, ?)")
			consArgs = append(consArgs, id, planCostItemID(item), id+"_q", planCostStepID(s), id+"_w")
		}
		for s := range planCostSteps + planCostIdleSteps {
			lt, lr, oh := fmt.Sprintf("rt_plancost_lt_%03d", s), fmt.Sprintf("rt_plancost_lr_%03d", s), fmt.Sprintf("rt_plancost_oh_%03d", s)
			rateVals = append(rateVals, "(?, '30', 'second', 'un_plancost')", "(?, '25', 'dollar', 'hour')", "(?, '20', 'dollar', 'hour')")
			rateArgs = append(rateArgs, lt, lr, oh)
			stepVals = append(stepVals, "(?, ?, ?, ?, ?, ?, '0.1', '0.05')")
			stepArgs = append(stepArgs, planCostStepID(s), fmt.Sprintf("Plan Cost Step %03d", s), planCostAccount, lr, lt, oh)

			// Every tenth step was given a second production later, which costing must not pick.
			for k, created := range []time.Time{planCostOrigin, planCostOrigin.AddDate(0, 0, 1)} {
				if k == 1 && s%10 != 5 {
					continue
				}
				id := fmt.Sprintf("prd_plancost_%03d_%d", s, k)
				quantity(id+"_q", fmt.Sprintf("%d", 12/(k+1)), "un_plancost")
				prodVals = append(prodVals, "(?, ?, ?, ?, ?, ?)")
				prodArgs = append(prodArgs, id, planCostItemID(planCostStepItem(s)), id+"_q", planCostStepID(s), created, created)
			}

			consume(s, 0, planCostFirstMaterial+s%planCostMaterials, "2")
			switch {
			case s >= planCostSteps:
			case s >= planCostParts:
				consume(s, 1, (s-planCostParts)*5, "1")
			case s%10 != 0:
				consume(s, 1, s-1, "1")
			}
		}
		exec("INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES "+strings.Join(rateVals, ","), rateArgs...)
		exec(`INSERT INTO production_step (id, name, account_id, labor_rate_id, labor_time_id, overhead_rate_id, leveling_factor, allowances)
		      VALUES `+strings.Join(stepVals, ","), stepArgs...)
		exec("INSERT INTO quantity (id, value, unit_id) VALUES "+strings.Join(qVals, ","), qArgs...)
		exec("INSERT INTO production (id, item_id, quantity_id, production_step_id, created_at, updated_at) VALUES "+strings.Join(prodVals, ","), prodArgs...)
		exec("INSERT INTO consumption (id, item_id, quantity_id, production_step_id, waste_quantity_id) VALUES "+strings.Join(consVals, ","), consArgs...)

		const chunk = 1_000
		for start := 0; start < planCostBatches; start += chunk {
			var qVals, bVals []string
			var qArgs, bArgs []any
			for i := start; i < start+chunk; i++ {
				createdAt := planCostCreatedAt(i)
				var station, scanned, step, waste, seconds any
				if s := planCostStation(i); s >= 0 {
					station = planCostStationID(s)
				}
				if i%4 != 3 {
					scanned = createdAt.Add(30 * time.Minute)
				}
				item := planCostItem(i)
				if i%20 != 7 {
					step = planCostStepID(planCostStepFor(item))
				}
				qID := fmt.Sprintf("qu_plancost_%06d", i)
				value, unit := "24", "un_plancost"
				if i%20 == 9 {
					value, unit = "2", "un_plancost_dz"
				}
				qVals = append(qVals, "(?, ?, ?)")
				qArgs = append(qArgs, qID, value, unit)
				if i%10 == 3 {
					waste = qID + "_w"
					qVals = append(qVals, "(?, '1', 'un_plancost')")
					qArgs = append(qArgs, waste)
				}
				if i%12 == 5 {
					seconds = qID + "_s"
					qVals = append(qVals, "(?, '2', 'un_plancost')")
					qArgs = append(qArgs, seconds)
				}
				bVals = append(bVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
				bArgs = append(bArgs, fmt.Sprintf("btch_plancost_%06d", i), planCostAccount, planCostItemID(item), qID, waste, seconds,
					station, step, scanned, createdAt, createdAt)
			}
			exec("INSERT INTO quantity (id, value, unit_id) VALUES "+strings.Join(qVals, ","), qArgs...)
			exec(`INSERT INTO batch (id, account_id, item_id, quantity_id, waste_quantity_id, seconds_quantity_id, scanning_station_id,
			      production_step_id, scanned_at, created_at, updated_at) VALUES `+strings.Join(bVals, ","), bArgs...)
		}
		exec("UPDATE account SET name = ? WHERE id = ?", planCostCorpusVersion, planCostAccount)
		exec("ANALYZE TABLE batch, production_step, production, consumption, item, scanning_station")
	})
}

func productionCostPlanCases() []planCase[domain.AnalyzeProductionCostsParams] {
	last := planCostCreatedAt(planCostBatches - 1)
	window := func(start, end time.Time) func(*domain.AnalyzeProductionCostsParams) {
		return func(p *domain.AnalyzeProductionCostsParams) { p.StartDate, p.EndDate = start, end }
	}
	return planCases(domain.AnalyzeProductionCostsParams{AccountID: planCostAccount}, []planDim[domain.AnalyzeProductionCostsParams]{
		{"departments", []planValue[domain.AnalyzeProductionCostsParams]{
			{"departments=busiest", func(p *domain.AnalyzeProductionCostsParams) { p.DepartmentIDs = []string{planCostDepartmentID(0)} }},
			{"departments=quiet", func(p *domain.AnalyzeProductionCostsParams) { p.DepartmentIDs = []string{planCostDepartmentID(3)} }},
			{"departments=no-stations", func(p *domain.AnalyzeProductionCostsParams) { p.DepartmentIDs = []string{planCostEmptyDepartment} }},
		}},
		{"categories", []planValue[domain.AnalyzeProductionCostsParams]{
			{"categories=busy", func(p *domain.AnalyzeProductionCostsParams) { p.CategoryIDs = []string{planCostCategoryID(0)} }},
			{"categories=rare", func(p *domain.AnalyzeProductionCostsParams) {
				p.CategoryIDs = []string{planCostCategoryID(planCostRareCategory)}
			}},
		}},
		{"items", []planValue[domain.AnalyzeProductionCostsParams]{
			{"items=busiest-good", func(p *domain.AnalyzeProductionCostsParams) {
				p.ItemIDs = []string{planCostItemID(planCostFirstProduct)}
			}},
			{"items=rare", func(p *domain.AnalyzeProductionCostsParams) { p.ItemIDs = []string{planCostItemID(planCostRareItem)} }},
		}},
		{"product_lines", []planValue[domain.AnalyzeProductionCostsParams]{
			{"product_lines=first", func(p *domain.AnalyzeProductionCostsParams) { p.ProductLineIDs = []string{planCostLineID(0)} }},
		}},
	}, []planValue[domain.AnalyzeProductionCostsParams]{
		{"week", window(last.AddDate(0, 0, -7), last)},
		{"month", window(last.AddDate(0, -1, 0), last)},
		{"quarter", window(last.AddDate(0, -3, 0), last)},
		{"1y", window(last.AddDate(-1, 0, 0), last)},
		{"old-month", window(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 31, 23, 59, 59, 999_000_000, time.UTC))},
	})
}

// readProductionCosts runs the reads the service makes for one report.
func readProductionCosts(ctx context.Context, q *sqlc.Queries, p domain.AnalyzeProductionCostsParams) ([]domain.ProductionCostRow, map[string]domain.ProductionCostStep, error) {
	repo := NewAnalyticsRepo(q)
	rows, apiErr := repo.GetProductionCostRows(ctx, p)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	seen := map[string]bool{}
	var stepIDs []string
	for _, r := range rows {
		if !seen[r.ProductionStepID] {
			seen[r.ProductionStepID] = true
			stepIDs = append(stepIDs, r.ProductionStepID)
		}
	}
	steps, apiErr := repo.GetProductionCostSteps(ctx, p.AccountID, stepIDs)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	if _, apiErr := repo.GetBaseUnitIDsByDimension(ctx); apiErr != nil {
		return nil, nil, apiErr
	}
	return rows, steps, nil
}

// productionCostPlanFloor is a report's scope: the scans its key ranges, the account's in the window or
// only the selected departments' stations'. Items and categories are checked on the scans the key yields:
// no account-led key orders a batch by its item.
func productionCostPlanFloor(t *testing.T, db *sql.DB, p domain.AnalyzeProductionCostsParams) map[string]float64 {
	t.Helper()
	where, args := productionCostPlanScope(p)
	var scans float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM batch WHERE "+where, args...).Scan(&scans))
	return map[string]float64{"b": scans}
}

func productionCostPlanScope(p domain.AnalyzeProductionCostsParams) (string, []any) {
	where := "account_id = ? AND scanned_at >= ? AND scanned_at <= ?"
	args := []any{p.AccountID, p.StartDate, p.EndDate}
	if len(p.DepartmentIDs) > 0 {
		where += " AND scanning_station_id IN (SELECT id FROM scanning_station WHERE account_id = ? AND department_id IN (" + placeholders(len(p.DepartmentIDs)) + "))"
		args = append(append(args, p.AccountID), stringsToAny(p.DepartmentIDs)...)
	}
	return where, args
}

// TestProductionCosts_ReadsItsScope holds a production cost report to reading only the scans its window
// (and departments) cover, through an account-led key.
func TestProductionCosts_ReadsItsScope(t *testing.T) {
	ensureProductionCostCorpus(t)
	aggregatePlanSuite[domain.AnalyzeProductionCostsParams]{
		tables: []aggregateTable{{table: "batch", scopeColumn: "account_id", from: "FROM batch b", alias: "b"}},
		cases:  productionCostPlanCases(),
		report: func(ctx context.Context, q *sqlc.Queries, p domain.AnalyzeProductionCostsParams) error {
			_, _, err := readProductionCosts(ctx, q, p)
			return err
		},
		floor: productionCostPlanFloor,
	}.run(t)
}

// TestProductionCosts_StepsReadWhatTheyReturn holds the step, production and consumption reads to the
// steps a window's scans name: by id, not by reading the account's every step, most of which scan nothing.
func TestProductionCosts_StepsReadWhatTheyReturn(t *testing.T) {
	ensureProductionCostCorpus(t)
	db := planDB(t)
	var cases []lookupPlanCase
	for _, c := range productionCostPlanCases() {
		if strings.Contains(c.name, ",") || !strings.HasPrefix(c.name, "unfiltered/") && !strings.HasPrefix(c.name, "departments=quiet/") {
			continue
		}
		where, args := productionCostPlanScope(c.params)
		stepIDs, apiErr := queryStringColumn(context.Background(), db, "SELECT DISTINCT production_step_id FROM batch WHERE production_step_id IS NOT NULL AND "+where, args...)
		require.Nil(t, apiErr)
		cases = append(cases, lookupPlanCase{name: c.name, run: func(ctx context.Context, q *sqlc.Queries) error {
			_, apiErr := NewAnalyticsRepo(q).GetProductionCostSteps(ctx, planCostAccount, stepIDs)
			if apiErr != nil {
				return apiErr
			}
			return nil
		}})
	}
	require.NotEmpty(t, cases)
	lookupPlanSuite{tables: []string{"production_step"}, cases: cases}.run(t)
}

// TestProductionCosts_ResultsUnchanged pins what every plan-tested production cost read returns.
func TestProductionCosts_ResultsUnchanged(t *testing.T) {
	ensureProductionCostCorpus(t)
	q := sqlc.New(planDB(t))
	ctx := context.Background()
	got := map[string]string{}
	for _, c := range productionCostPlanCases() {
		rows, steps, err := readProductionCosts(ctx, q, c.params)
		require.NoError(t, err)
		for id, s := range steps {
			sort.Slice(s.Consumptions, func(i, j int) bool {
				a, _ := json.Marshal(s.Consumptions[i])
				b, _ := json.Marshal(s.Consumptions[j])
				return string(a) < string(b)
			})
			steps[id] = s
		}
		got[c.name] = planDigest(t, struct{ Rows, Steps any }{unorderedRows(t, rows), steps})
	}
	checkPlanResults(t, "production_costs.json", got)
}
