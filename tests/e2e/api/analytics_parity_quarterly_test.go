//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Quarterly orders total the ordered value of sale lines on sales orders by the UTC year and quarter they were
// issued: each line's quantity in each times its price per each, whatever the order's status now.

func quarterlyOrders(t *testing.T, client *Client, body map[string]any) map[string]any {
	t.Helper()
	got := mustPutAnalytics(t, client, quarterlyOrdersPath, nil, body)
	assert.Equal(t, "analyze_quarterly_orders_response", jsonField(got, "object"))
	return jsonObject(got, "data")
}

// thisQuarter is the data key and quarter field an order issued now is booked under.
func thisQuarter() (year, quarter string) {
	now := time.Now().UTC()
	return fmt.Sprint(now.Year()), fmt.Sprintf("q%d", (int(now.Month())-1)/3+1)
}

func assertOnlyThisQuarter(t *testing.T, data map[string]any, total string) {
	t.Helper()
	year, quarter := thisQuarter()
	require.Len(t, data, 1, "only this year has orders: %v", data)
	y := jsonObject(data, year)
	require.NotNil(t, y, "the year issued: %v", data)
	for _, q := range []string{"q1", "q2", "q3", "q4"} {
		want := "0"
		if q == quarter {
			want = total
		}
		assert.Equal(t, want, jsonField(y, q), q)
	}
	assert.Equal(t, total, jsonField(y, "total"))
}

func TestAnalyticsParityQuarterlyOrders_TotalsIssuedSalesOrders(t *testing.T) {
	t.Parallel()
	customer := parityCustomer(t, "")
	// $21.25 + $36: five pairs at $4.25 and two dozen (12 pr) at $3.00.
	issueParityOrder(t, customer, "",
		parityLine(SeedProductID, "5", SeedUnitID, "4.25"),
		parityLine(parityProductSCK002, "2", seedDozenUnitID, "3.00"))
	// Shipped and completed, it still counts: $3.
	done := issueParityOrder(t, customer, "", parityLine(parityProductSCK003, "3", SeedUnitID, "1.00"))
	shipWholeOrder(t, jsonField(done, "id"))
	// An estimate has no issue date: it neither counts nor breaks the report.
	createParityEstimate(t, customer, parityLine(parityProductSCK004, "7", SeedUnitID, "1.00"))

	body := map[string]any{"customer_ids": []string{customer}}
	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, body), "60.25")

	body["years_back"] = 1
	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, body), "60.25")

	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, map[string]any{"customer_ids": []string{customer}, "item_ids": []string{parityItemSCK002}}), "36")
	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, map[string]any{"customer_ids": []string{customer}, "product_line_ids": []string{SeedProductLineID}}), "60.25")
}

// The whole account holds estimates (no issue date), which once failed the report outright; it still answers,
// within its default five years.
func TestAnalyticsParityQuarterlyOrders_AccountWithEstimatesAnswers(t *testing.T) {
	t.Parallel()
	data := quarterlyOrders(t, apiClient, nil)
	earliest := time.Now().UTC().Year() - 4
	for year := range data {
		var y int
		_, err := fmt.Sscan(year, &y)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, y, earliest, "the default window is the last five calendar years")
	}
}

func TestAnalyticsParityQuarterlyOrders_ChildrenGroupsAndRepsScopeIt(t *testing.T) {
	t.Parallel()
	group := newAccountGroup(t)
	parent := parityCustomer(t, "")
	child := parityCustomer(t, group)
	makeChildCustomer(t, parent, child)
	rep, repID := paritySalesRep(t, "invoices")
	issueParityOrder(t, child, repID, parityLine(SeedProductID, "2", SeedUnitID, "2.00")) // $4, the rep's
	issueParityOrder(t, child, "", parityLine(SeedProductID, "1", SeedUnitID, "10.00"))   // $10

	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, map[string]any{"customer_ids": []string{parent}}), "14")
	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, map[string]any{"customer_group_ids": []string{group}}), "14")
	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, map[string]any{"customer_ids": []string{child}, "sales_rep_ids": []string{repID}}), "4")
	assert.Empty(t, quarterlyOrders(t, apiClient, map[string]any{"customer_ids": []string{parent}, "customer_group_ids": []string{newAccountGroup(t)}}))

	// A sales rep sees only their own orders, whatever they ask for.
	assertOnlyThisQuarter(t, quarterlyOrders(t, rep, map[string]any{"customer_ids": []string{child}, "sales_rep_ids": []string{SeedAccountUserID}}), "4")
}

func TestAnalyticsParityQuarterlyOrders_RejectsAnImpossibleWindow(t *testing.T) {
	t.Parallel()
	for _, yearsBack := range []any{101, -1, "five"} {
		status, _, raw := putAnalytics(t, apiClient, quarterlyOrdersPath, url.Values{}, map[string]any{"years_back": yearsBack})
		assert.Contains(t, []int{400, 422}, status, "years_back=%v: %s", yearsBack, string(raw))
	}
}
