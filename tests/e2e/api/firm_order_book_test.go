//go:build e2e

package api_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The firm order book: the solver reading orders that have already been placed,
// rather than only the history it forecasts from.
//
// These drive the preview endpoint because it is the solve without the write —
// the same path a generated schedule takes, so what is asserted here is what a
// real plan is built from. Solves take the read side of planningMu so a settings
// write elsewhere cannot move the plan mid-test.

// previewPlan runs a solve and returns the parsed preview, skipping when the
// environment has no constraint department configured to plan against.
func previewPlan(t *testing.T) map[string]any {
	t.Helper()
	defer lockPlanningRead()()

	status, body, err := apiClient.Put(productionSchedulePreviewPath, map[string]any{})
	require.NoError(t, err)
	require.Less(t, status, 500, "preview must not 5xx: %s", string(body))
	if status == 400 {
		t.Skip("no constraint department configured in this environment")
	}
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func previewDiagnostics(t *testing.T, preview map[string]any) map[string]any {
	t.Helper()
	d, ok := preview["diagnostics"].(map[string]any)
	require.True(t, ok, "preview should carry diagnostics: %v", preview)
	return d
}

// The order book has to reach the solver at all. Every seeded environment has open
// orders, so a solve that reports none is the wiring being broken rather than a
// tenant with an empty book — and the diagnostics have to serialize as real values,
// not as absent keys a client would read as zero.
func TestFirmOrderBook_SolverReportsTheOrderBook(t *testing.T) {
	t.Parallel()

	diagnostics := previewDiagnostics(t, previewPlan(t))

	require.Contains(t, diagnostics, "firm_demand_units",
		"the solver must report the order book it planned against")
	require.Contains(t, diagnostics, "undated_firm_order_count")
	require.Contains(t, diagnostics, "at_risk_orders")

	for _, raw := range atRiskOrders(t, diagnostics) {
		order, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "schedule_at_risk_order", jsonField(order, "object"))
		assert.Contains(t, []string{"past_due", "undated", "short"}, jsonField(order, "reason"),
			"every at-risk order must say why")
		assert.NotEmpty(t, jsonField(order, "sales_order"), "an at-risk order must name the order")
	}
}

// The order book is walked through maps on the way in, and Go randomizes map
// iteration, so its output has to be ordered explicitly. At-risk orders come back
// sorted by due week — soonest first, which is also the order a planner works them.
//
// Asserted as an invariant of one response rather than by comparing two solves: the
// suite issues and unissues orders in parallel throughout, so two solves taken
// seconds apart legitimately see different order books. True run-to-run determinism
// is pinned where it can be held still — TestBuildFirmSchedule_Deterministic and
// TestSolve_Deterministic, over a fixed input.
func TestFirmOrderBook_AtRiskOrdersComeBackOrdered(t *testing.T) {
	t.Parallel()

	orders := atRiskOrders(t, previewDiagnostics(t, previewPlan(t)))

	previous := -1
	for _, raw := range orders {
		row, ok := raw.(map[string]any)
		require.True(t, ok)
		week, ok := row["due_week"].(float64)
		require.True(t, ok, "due_week should be numeric: %v", row["due_week"])
		assert.GreaterOrEqual(t, int(week), previous,
			"at-risk orders must be sorted by due week, soonest first")
		previous = int(week)
	}
}

// The point of the phase: an order placed today has to move the plan today, rather
// than waiting a month to age into the trailing-twelve window.
//
// A large order for a seeded product, due inside the horizon, must increase the
// order book the solver reports. That number is what the sweep draws down against,
// so a change in it is a change in the plan.
func TestFirmOrderBook_ANewOrderMovesThePlanImmediately(t *testing.T) {
	t.Parallel()
	lockOrderBook(t)

	before := previewDiagnostics(t, previewPlan(t))
	beforeUnits, ok := before["firm_demand_units"].(float64)
	require.True(t, ok, "firm_demand_units should be a number: %v", before["firm_demand_units"])

	// Due inside the horizon but far enough out that the finishing lead time still
	// leaves a constraint week inside it. Sized far above anything else the suite
	// orders, so a concurrent unissue elsewhere cannot mask the increase.
	customerID := leadTimeCustomer(t, "e2e-firm-order", ptrInt(75), "")
	body := minimalSalesOrderCreateBody(t, customerID)
	setOrderLineQuantity(body, firmOrderProbeUnits)
	order := issueOrderWithBody(t, body)
	require.NotEmpty(t, shipByDate(t, order), "the order must carry a commitment to be datable")

	after := previewDiagnostics(t, previewPlan(t))
	afterUnits, ok := after["firm_demand_units"].(float64)
	require.True(t, ok)

	assert.Greater(t, afterUnits-beforeUnits, float64(firmOrderProbeUnits)/2,
		"issuing a %d-unit order should be visible in the order book the solver plans against (before %v, after %v)",
		firmOrderProbeUnits, beforeUnits, afterUnits)
}

// An order that is not yet issued is not demand. This is the same rule the
// historical demand query applies to estimates, and it has to hold for the order
// book too or a quote nobody accepted would consume machine time.
//
// The assertion is a tolerance rather than an equality because the suite runs in
// parallel and other tests issue their own orders throughout. The estimate is
// deliberately enormous so its absence is unambiguous: a few units of concurrent
// churn cannot be mistaken for 50,000 having leaked in.
func TestFirmOrderBook_AnEstimateIsNotDemand(t *testing.T) {
	t.Parallel()
	lockOrderBook(t)

	before := previewDiagnostics(t, previewPlan(t))
	beforeUnits, _ := before["firm_demand_units"].(float64)

	customerID := leadTimeCustomer(t, "e2e-firm-estimate", ptrInt(75), "")
	body := minimalSalesOrderCreateBody(t, customerID)
	setOrderLineQuantity(body, firmOrderProbeUnits)

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	deleteOrder(t, jsonField(created, "id"))
	require.Equal(t, "estimate", jsonField(created, "status"), "precondition: the order is still an estimate")

	after := previewDiagnostics(t, previewPlan(t))
	afterUnits, _ := after["firm_demand_units"].(float64)

	assert.Less(t, afterUnits-beforeUnits, float64(firmOrderProbeUnits)/2,
		"an unissued estimate must not enter the order book (before %v, after %v)", beforeUnits, afterUnits)
}

// Unissuing gives the demand back. An order pulled back is no longer owed, and
// leaving it in the book would keep the plan building for it forever.
//
// Measured as the size of the swing rather than an absolute, so orders other tests
// issue in parallel cannot decide the result.
func TestFirmOrderBook_UnissuingReturnsTheDemand(t *testing.T) {
	t.Parallel()
	lockOrderBook(t)

	customerID := leadTimeCustomer(t, "e2e-firm-unissue", ptrInt(75), "")

	baseline := previewDiagnostics(t, previewPlan(t))
	baselineUnits, _ := baseline["firm_demand_units"].(float64)

	body := minimalSalesOrderCreateBody(t, customerID)
	setOrderLineQuantity(body, firmOrderProbeUnits)
	order := issueOrderWithBody(t, body)

	issued := previewDiagnostics(t, previewPlan(t))
	issuedUnits, _ := issued["firm_demand_units"].(float64)
	require.Greater(t, issuedUnits-baselineUnits, float64(firmOrderProbeUnits)/2,
		"precondition: issuing a %d-unit order should be visible in the book", firmOrderProbeUnits)

	status, unissueBody, err := apiClient.Put(salesOrdersPath+"/"+jsonField(order, "id")+"/actions/unissue", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, unissueBody)

	after := previewDiagnostics(t, previewPlan(t))
	afterUnits, _ := after["firm_demand_units"].(float64)

	assert.Less(t, afterUnits-baselineUnits, float64(firmOrderProbeUnits)/2,
		"unissuing should give the demand back (baseline %v, issued %v, after %v)",
		baselineUnits, issuedUnits, afterUnits)
}

// An order due sooner than the plant can produce it is the most useful thing the
// solver can say, so it must be reported rather than quietly scheduled as if it
// were achievable.
func TestFirmOrderBook_AnImpossibleCommitmentIsReportedAtRisk(t *testing.T) {
	t.Parallel()

	// One day out: well inside the finishing lead time, so the constraint stage would
	// have had to start before the horizon.
	promised := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02") + "T00:00:00Z"
	customerID := leadTimeCustomer(t, "e2e-firm-atrisk", ptrInt(1), "")
	order := issueOrderForCustomer(t, customerID, map[string]any{"promised_at": promised})
	orderNumber := jsonField(order, "number")
	require.NotEmpty(t, orderNumber)

	diagnostics := previewDiagnostics(t, previewPlan(t))
	found := false
	for _, raw := range atRiskOrders(t, diagnostics) {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		salesOrder, ok := row["sales_order"].(map[string]any)
		if !ok {
			continue
		}
		if jsonField(salesOrder, "id") == jsonField(order, "id") {
			found = true
			assert.Equal(t, "past_due", jsonField(row, "reason"),
				"an order due inside the production lead time is past due at the constraint")
			assert.Equal(t, "0", jsonField(row, "due_week"),
				"a past-due requirement clamps to the front of the horizon")
		}
	}

	if !found {
		// The seeded product may not descend from any constraint item, in which case
		// nothing in the plan produces it and it correctly cannot be at risk.
		t.Skipf("order %s is not produced by the constraint department in this environment", orderNumber)
	}
}

// firmOrderProbeUnits is far larger than the seeded order book (~415 units at the
// constraint) and than anything the rest of the suite orders, so a probe's presence
// or absence cannot be confused with concurrent churn. Not larger than it needs to
// be: the order book drives campaign sizing, and an absurd probe makes every
// concurrent solve slower for no extra signal.
const firmOrderProbeUnits = 5000

// orderBookMu serializes the tests that measure the account-wide order book.
//
// firm_demand_units is a single number for the whole account, so two tests each
// adding a probe and then asserting on the total will read each other's probe. They
// stay parallel with the rest of the suite — only with each other are they exclusive.
var orderBookMu sync.Mutex

// lockOrderBook takes the order-book measurement slot for the duration of one test.
func lockOrderBook(t *testing.T) {
	t.Helper()
	orderBookMu.Lock()
	t.Cleanup(orderBookMu.Unlock)
}

func setOrderLineQuantity(body map[string]any, units int) {
	lines, ok := body["lines"].([]map[string]any)
	if !ok || len(lines) == 0 {
		return
	}
	quantity, ok := lines[0]["quantity"].(map[string]any)
	if !ok {
		return
	}
	quantity["value"] = strconv.Itoa(units)
}

// issueOrderWithBody creates and issues an order from a prepared body.
func issueOrderWithBody(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)

	issueStatus, issueBody, err := apiClient.Put(salesOrdersPath+"/"+orderID+"/actions/issue", nil)
	require.NoError(t, err)
	require.Less(t, issueStatus, 500, "issue must not 5xx: %s", string(issueBody))
	requireStatus(t, 200, issueStatus, issueBody)
	return parseJSON(issueBody)
}

// atRiskOrders unwraps the at_risk_orders list. It is a List resource like every
// other collection the API serves, not a bare array.
func atRiskOrders(t *testing.T, diagnostics map[string]any) []any {
	t.Helper()
	list, ok := diagnostics["at_risk_orders"].(map[string]any)
	require.True(t, ok, "at_risk_orders must be a list resource: %v", diagnostics["at_risk_orders"])
	assert.Equal(t, "list", jsonField(list, "object"))
	data, ok := list["data"].([]any)
	require.True(t, ok, "at_risk_orders.data must be an array, not null: %v", list["data"])
	return data
}
