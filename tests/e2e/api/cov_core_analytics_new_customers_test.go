//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const newCustomersPath = "/v1/core/analytics/new-customers-table"

// newCustomersToday lists the customers added around now, with any extra body fields.
func newCustomersToday(t *testing.T, params url.Values, extra map[string]any) (int, map[string]any, []byte) {
	t.Helper()
	now := time.Now().UTC()
	body := map[string]any{"starts_at": rfc3339(now.Add(-time.Hour)), "ends_at": rfc3339(now.Add(time.Hour))}
	for k, v := range extra {
		body[k] = v
	}
	return putSales(t, newCustomersPath, params, body)
}

// newCustomerRow finds the customer in a list of new customers, or nil.
func newCustomerRow(list map[string]any, customerID string) map[string]any {
	for _, row := range jsonArray(list, "data") {
		if m := row.(map[string]any); jsonField(m, "id") == customerID {
			return m
		}
	}
	return nil
}

// awaitNewCustomer waits until the customer's first sale reaches its summary (a few seconds after the invoice) and returns its row.
func awaitNewCustomer(t *testing.T, customerID string) map[string]any {
	t.Helper()
	var row map[string]any
	eventually(t, 45*time.Second, time.Second, func() error {
		// 503 means the buyer summaries are still being backfilled on a fresh stack.
		status, list, body := newCustomersToday(t, url.Values{"limit": {"1000"}}, map[string]any{"customer_group_ids": []string{SeedCustomerGroupID}})
		if status != 200 {
			return fmt.Errorf("new customers answered %d: %s", status, string(body))
		}
		if row = newCustomerRow(list, customerID); row == nil {
			return fmt.Errorf("customer %s not listed yet", customerID)
		}
		return nil
	})
	return row
}

func TestNewCustomers_ListsACustomerOnceItHasOrdered(t *testing.T) {
	t.Parallel()
	addedBefore := time.Now().UTC().Add(-time.Second)
	sale := shipSaleToNewCustomer(t)
	row := awaitNewCustomer(t, sale.customerID)

	assert.Equal(t, "new_customer", jsonField(row, "object"))
	assert.NotEmpty(t, jsonField(row, "number"))
	assert.NotEmpty(t, jsonField(row, "name"))
	assert.NotEmpty(t, jsonField(row, "customer_group_name"), "the customer's group")
	assertNilField(t, row, "sales_rep_name")
	assert.True(t, decimal.RequireFromString(shippedSaleRevenue).Equal(decimal.RequireFromString(computedValue(t, row, "lifetime_revenue"))),
		"lifetime revenue is the whole sale: %v", row["lifetime_revenue"])
	first := mustTime(t, jsonField(row, "first_ordered_at"))
	added := mustTime(t, jsonField(row, "added_at"))
	assert.False(t, added.Before(addedBefore), "added when the test created it")
	assert.False(t, first.Before(added), "ordered after it was added")
}

func TestNewCustomers_LeavesOutACustomerThatHasNotOrdered(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	// Another customer's sale proves the summaries are current before asserting the absence.
	awaitNewCustomer(t, shipSaleToNewCustomer(t).customerID)

	status, list, body := newCustomersToday(t, url.Values{"limit": {"1000"}}, nil)
	requireStatus(t, 200, status, body)
	assert.Nil(t, newCustomerRow(list, customerID), "a customer with no orders is not a new customer yet")
}

func TestNewCustomers_Filters(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitNewCustomer(t, sale.customerID)

	for name, tt := range map[string]struct {
		body   map[string]any
		listed bool
	}{
		"its customer group":           {map[string]any{"customer_group_ids": []string{SeedCustomerGroupID}}, true},
		"another customer group":       {map[string]any{"customer_group_ids": []string{"acgp_none"}}, false},
		"a sales rep it does not have": {map[string]any{"sales_rep_ids": []string{SeedAccountUserID}}, false},
		"added after the window": {map[string]any{
			"starts_at": rfc3339(time.Now().UTC().AddDate(0, 0, -30)), "ends_at": rfc3339(time.Now().UTC().AddDate(0, 0, -29)),
		}, false},
	} {
		t.Run(name, func(t *testing.T) {
			status, list, body := newCustomersToday(t, url.Values{"limit": {"1000"}}, tt.body)
			requireStatus(t, 200, status, body)
			assert.Equal(t, tt.listed, newCustomerRow(list, sale.customerID) != nil)
		})
	}
}

func TestNewCustomers_PagesByCursorBothWays(t *testing.T) {
	t.Parallel()
	// Customers of a group of its own, so parallel tests' new customers cannot join mid-walk.
	groupID := newAccountGroup(t)
	var mine []string
	for range 3 {
		customerID := shipSaleToNewCustomer(t).customerID
		status, body, err := apiClient.Patch(customersPath+"/"+customerID, map[string]any{"customer_type_group_id": groupID}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		mine = append(mine, customerID)
	}
	eventually(t, 45*time.Second, time.Second, func() error {
		status, list, body := newCustomersToday(t, url.Values{"limit": {"1000"}}, map[string]any{"customer_group_ids": []string{groupID}})
		if status != 200 {
			return fmt.Errorf("new customers answered %d: %s", status, string(body))
		}
		if n := len(jsonArray(list, "data")); n != len(mine) {
			return fmt.Errorf("%d of %d customers listed", n, len(mine))
		}
		return nil
	})

	page := func(params url.Values) ([]string, map[string]any) {
		status, list, body := newCustomersToday(t, params, map[string]any{"customer_group_ids": []string{groupID}})
		requireStatus(t, 200, status, body)
		var ids []string
		for _, row := range jsonArray(list, "data") {
			ids = append(ids, jsonField(row.(map[string]any), "id"))
		}
		return ids, jsonObject(list, "page_info")
	}

	var forward []string
	params := url.Values{"limit": {"1"}}
	var info map[string]any
	for i := 0; i < 200; i++ {
		var ids []string
		ids, info = page(params)
		forward = append(forward, ids...)
		if jsonField(info, "has_next_page") != "true" {
			break
		}
		params = url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "next_page_url"))}}
	}
	require.ElementsMatch(t, mine, forward, "one page per customer of the group")
	all, _ := page(url.Values{"limit": {"1000"}})
	require.Equal(t, all, forward, "one at a time, the same customers in the same order")

	var backward []string
	for i := 0; i < 200 && jsonField(info, "has_prev_page") == "true"; i++ {
		var ids []string
		ids, info = page(url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "previous_page_url"))}})
		backward = append(ids, backward...)
	}
	require.Equal(t, forward[:len(forward)-1], backward, "paging back retraces the pages before the last")
}

func TestNewCustomers_RejectsABadWindow(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	for name, body := range map[string]map[string]any{
		"no start":              {"ends_at": rfc3339(now)},
		"no end":                {"starts_at": rfc3339(now)},
		"ends before it starts": {"starts_at": rfc3339(now), "ends_at": rfc3339(now.Add(-time.Hour))},
	} {
		t.Run(name, func(t *testing.T) {
			status, _, raw := putSales(t, newCustomersPath, nil, body)
			assert.Equal(t, 400, status, string(raw))
		})
	}
}

// An invoice spanning two product lines, filtered to both, is read from the per-line rollups: its money
// adds up line by line, and it counts as one invoice, not one per line.
func TestSalesAnalytics_AnInvoiceSpanningTwoProductLinesCountsOnce(t *testing.T) {
	t.Parallel()
	const otherLineID, otherProductID = "pdln_01k0a735ypfjva933tg57wfx0t", "pd_01seedfcebad000000"

	status, body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-two-lines")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	customerID := jsonField(parseJSON(body), "id")
	status, body, err = apiClient.Post(productLineAccessPath, map[string]any{
		"customer_id": customerID, "product_line_ids": []string{SeedProductLineID, otherLineID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(productLineAccessPath + "/" + customerID)
		_, _, _ = apiClient.Delete(customersPath + "/" + customerID)
	})

	// 5 pr at $4.25 and 3 each at $2.00: $27.25 on one invoice.
	shipOrder(t, customerID, []map[string]any{
		{
			"product_id": SeedProductID,
			"quantity":   map[string]any{"value": "5", "unit_id": SeedUnitID},
			"unit_price": map[string]any{"value": "4.25", "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
		},
		{
			"product_id": otherProductID,
			"quantity":   map[string]any{"value": "3", "unit_id": "each"},
			"unit_price": map[string]any{"value": "2.00", "numerator_unit_id": dollarUnitID, "denominator_unit_id": "each"},
		},
	})

	now := time.Now().UTC()
	breakdown := func(productLines ...string) map[string]any {
		body := map[string]any{
			"group_by": "customer", "customer_ids": []string{customerID}, "product_line_ids": productLines,
			"starts_at": rfc3339(now.Add(-24 * time.Hour)), "ends_at": rfc3339(now.Add(24 * time.Hour)),
		}
		status, list, raw := putSales(t, salesBreakdownPath, nil, body)
		requireStatus(t, 200, status, raw)
		groups := jsonArray(list, "data")
		if len(groups) == 0 {
			return nil
		}
		require.Len(t, groups, 1)
		return jsonObject(groups[0].(map[string]any), "totals")
	}
	// Facts are written a few seconds after the invoice.
	var both map[string]any
	eventually(t, 45*time.Second, time.Second, func() error {
		both = breakdown(SeedProductLineID, otherLineID)
		if both == nil || jsonField(both, "line_count") != "2" {
			return fmt.Errorf("the invoice's lines have not reached the report yet: %v", both)
		}
		return nil
	})

	assert.True(t, decimal.RequireFromString("27.25").Equal(decimal.RequireFromString(computedValue(t, both, "revenue"))), "revenue: %v", both["revenue"])
	assert.Equal(t, "1", jsonField(both, "invoice_count"), "one invoice, though it spans both lines")
	socks, other := breakdown(SeedProductLineID), breakdown(otherLineID)
	assert.Equal(t, "1", jsonField(socks, "invoice_count"))
	assert.Equal(t, "1", jsonField(other, "invoice_count"))
	assert.True(t, decimal.RequireFromString("21.25").Equal(decimal.RequireFromString(computedValue(t, socks, "revenue"))))
	assert.True(t, decimal.RequireFromString("6").Equal(decimal.RequireFromString(computedValue(t, other, "revenue"))))
}
