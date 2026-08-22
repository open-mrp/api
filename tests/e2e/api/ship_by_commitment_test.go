//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ship-by commitments: the lead-time chain (customer -> account group -> account),
// the promised-date override, stamping at issue, clearing on unissue, and the
// immutability rule that makes a commitment worth having.
//
// These deliberately do NOT touch the account-level default via the settings
// endpoint. Settings are account-wide and a write would race every other test in
// the package (cov_operations_production-schedule-settings_test.go serializes its
// own writes behind planningMu for the same reason). The account level is covered
// by reading whatever default is configured rather than by setting one.

const customerLeadTimePathSuffix = "/lead-time"

// leadTimeCustomer creates a customer with product-line access, optionally with its
// own lead time and/or an account group.
func leadTimeCustomer(t *testing.T, namePrefix string, leadTimeDays *int, accountGroupID string) string {
	t.Helper()

	body := validCustomerBody(uniqueName(namePrefix))
	if leadTimeDays != nil {
		body["lead_time_days"] = *leadTimeDays
	}
	if accountGroupID != "" {
		body["customer_type_group_id"] = accountGroupID
	}

	status, respBody, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	customerID := jsonField(parseJSON(respBody), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + customerID) })

	plStatus, plBody, err := apiClient.Post(productLineAccessPath, map[string]any{
		"customer_id":      customerID,
		"product_line_ids": []string{SeedProductLineID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, plStatus, plBody)

	return customerID
}

// leadTimeAccountGroup creates an account group, optionally with its own lead time.
func leadTimeAccountGroup(t *testing.T, namePrefix string, leadTimeDays *int) string {
	t.Helper()

	body := map[string]any{
		"name": uniqueName(namePrefix),
		"type": "type_group",
	}
	if leadTimeDays != nil {
		body["default_lead_time_days"] = *leadTimeDays
	}
	created := createAndCleanup(t, accountGroupsPath, body)
	return jsonField(created, "id")
}

// issueOrderForCustomer creates and issues an order, returning the issued order body.
func issueOrderForCustomer(t *testing.T, customerID string, extra map[string]any) map[string]any {
	t.Helper()

	body := minimalSalesOrderCreateBody(t, customerID)
	for k, v := range extra {
		body[k] = v
	}

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

// The date a lead-time *rule* commits to, less whatever the calendars pulled it back onto an open
// day. A span the order reported has that taken out already — use issuedPlusDays for those.
func expectedShipBy(t *testing.T, order map[string]any, ruleDays int) string {
	t.Helper()
	return issuedPlusDays(t, order, ruleDays-calendarAdjustmentDays(order))
}

// The issue date plus a span, with no calendar reasoning of its own.
func issuedPlusDays(t *testing.T, order map[string]any, days int) string {
	t.Helper()
	issuedAt := jsonField(order, "issued_at")
	require.NotEmpty(t, issuedAt, "issued order must carry issued_at")
	parsed, err := time.Parse(time.RFC3339, issuedAt)
	require.NoError(t, err)
	return parsed.UTC().AddDate(0, 0, days).Format("2006-01-02")
}

// The lead-time rule an order resolved to: the span it committed to, plus whatever the calendars
// took back out of it. Assert against this rather than lead_time_days, which moves with the date.
func committedRuleDays(t *testing.T, order map[string]any) int {
	t.Helper()
	days, err := strconv.Atoi(jsonField(order, "lead_time_days"))
	require.NoError(t, err, "an issued order must carry the days it committed to")
	return days + calendarAdjustmentDays(order)
}

// How many days the receiving and shipping calendars pulled an order's ship-by back. Zero when the
// field is absent, which is what an order whose dates all landed on open days reports.
func calendarAdjustmentDays(order map[string]any) int {
	adjustment, ok := order["calendar_adjustment_days"].(float64)
	if !ok {
		return 0
	}
	return int(adjustment)
}

// shipByDate normalizes the ship_by_date field, which serializes as a timestamp.
func shipByDate(t *testing.T, order map[string]any) string {
	t.Helper()
	raw := jsonField(order, "ship_by_date")
	if raw == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	require.NoError(t, err, "ship_by_date should be RFC3339: %q", raw)
	return parsed.UTC().Format("2006-01-02")
}

func TestShipByCommitment_CustomerLeadTimeWinsTheChain(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-shipby-grp", ptrInt(21))
	customerID := leadTimeCustomer(t, "e2e-shipby-cust", ptrInt(14), groupID)

	order := issueOrderForCustomer(t, customerID, nil)

	assert.Equal(t, "customer", jsonField(order, "lead_time_source"))
	assert.Equal(t, 14, committedRuleDays(t, order))
	assert.Equal(t, expectedShipBy(t, order, 14), shipByDate(t, order))
}

func TestShipByCommitment_InheritsAccountGroupLeadTime(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-shipby-grp-inherit", ptrInt(21))
	customerID := leadTimeCustomer(t, "e2e-shipby-cust-inherit", nil, groupID)

	order := issueOrderForCustomer(t, customerID, nil)

	assert.Equal(t, "account_group", jsonField(order, "lead_time_source"))
	assert.Equal(t, 21, committedRuleDays(t, order))
	assert.Equal(t, expectedShipBy(t, order, 21), shipByDate(t, order))
}

// With neither the customer nor its group configured, the account default applies.
// The number itself is whatever the account is set to, so this asserts the source
// and that the date is consistent with the days it reported, rather than pinning a
// value another test could legitimately change.
func TestShipByCommitment_FallsBackToAccountDefault(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-shipby-grp-none", nil)
	customerID := leadTimeCustomer(t, "e2e-shipby-cust-none", nil, groupID)

	order := issueOrderForCustomer(t, customerID, nil)

	assert.Equal(t, "account", jsonField(order, "lead_time_source"))
	days := jsonField(order, "lead_time_days")
	require.NotEmpty(t, days, "an issued order must carry the days it committed to")

	parsedDays, err := strconv.Atoi(days)
	require.NoError(t, err)
	assert.Equal(t, issuedPlusDays(t, order, parsedDays), shipByDate(t, order))
}

// A promised date is a negotiation about one order, so it beats every standing rule
// and records the span actually committed to rather than the customer's.
func TestShipByCommitment_PromisedDateOverridesTheChain(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-shipby-promised", ptrInt(45), "")
	// Landed on a weekday so the date is unchanged by the receiving calendar. A promise to deliver at a weekend resolves back to the Friday, which is correct but would make this test about the calendar rather than about the promise beating the standing rule. That case has its own coverage in commitment_basis_test.go.
	promisedDay := time.Now().UTC().AddDate(0, 0, 3)
	for promisedDay.Weekday() == time.Saturday || promisedDay.Weekday() == time.Sunday {
		promisedDay = promisedDay.AddDate(0, 0, 1)
	}
	promised := promisedDay.Format("2006-01-02") + "T00:00:00Z"

	order := issueOrderForCustomer(t, customerID, map[string]any{"promised_at": promised})

	assert.Equal(t, "manual", jsonField(order, "lead_time_source"))
	assert.Equal(t, promised[:10], shipByDate(t, order))
	assert.NotEqual(t, "45", jsonField(order, "lead_time_days"),
		"the committed span should be the promised one, not the customer's standing rule")
}

// An order that is no longer issued carries no promise: leaving the commitment
// behind would keep it in the past-due queue for a date nobody is working to.
func TestShipByCommitment_UnissueClearsTheCommitment(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-shipby-unissue", ptrInt(10), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderID := jsonField(order, "id")
	require.NotEmpty(t, shipByDate(t, order), "precondition: the issued order has a commitment")

	status, body, err := apiClient.Put(salesOrdersPath+"/"+orderID+"/actions/unissue", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "unissue must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	after := parseJSON(body)
	assert.Empty(t, shipByDate(t, after), "ship_by_date should be cleared")
	assert.Empty(t, jsonField(after, "lead_time_days"))
	assert.Empty(t, jsonField(after, "lead_time_source"))
}

// The rule the whole design rests on: a commitment is a fact about the moment it
// was made. Renegotiating a customer must never rewrite what was already promised.
func TestShipByCommitment_ChangingCustomerLeadTimeLeavesIssuedOrdersAlone(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-shipby-immutable", ptrInt(7), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderID := jsonField(order, "id")

	originalShipBy := shipByDate(t, order)
	require.NotEmpty(t, originalShipBy)
	require.Equal(t, 7, committedRuleDays(t, order))

	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+customerID,
		map[string]any{"lead_time_days": 90}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, patchStatus, 500, "customer update must not 5xx: %s", string(patchBody))
	requireStatus(t, 200, patchStatus, patchBody)

	getStatus, getBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	after := parseJSON(getBody)
	assert.Equal(t, originalShipBy, shipByDate(t, after),
		"an existing commitment must not move when the customer's lead time changes")
	assert.Equal(t, 7, committedRuleDays(t, after))
	assert.Equal(t, "customer", jsonField(after, "lead_time_source"))
}

func TestCustomerLeadTime_ResolvesChainWithoutAnOrder(t *testing.T) {
	t.Parallel()

	t.Run("customer", func(t *testing.T) {
		groupID := leadTimeAccountGroup(t, "e2e-lt-grp-a", ptrInt(21))
		customerID := leadTimeCustomer(t, "e2e-lt-cust-a", ptrInt(5), groupID)

		status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "lead-time lookup must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		got := parseJSON(body)
		assert.Equal(t, "customer_lead_time", jsonField(got, "object"))
		assert.Equal(t, "5", jsonField(got, "days"))
		assert.Equal(t, "customer", jsonField(got, "source"))
		assert.Nil(t, got["account_group"], "the group did not decide, so it should not be named")
	})

	t.Run("account_group", func(t *testing.T) {
		groupID := leadTimeAccountGroup(t, "e2e-lt-grp-b", ptrInt(21))
		customerID := leadTimeCustomer(t, "e2e-lt-cust-b", nil, groupID)

		// The group is an expandable sub-object, so the resolution names it only when asked to.
		status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix+"?include=account_group", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		got := parseJSON(body)
		assert.Equal(t, "21", jsonField(got, "days"))
		assert.Equal(t, "account_group", jsonField(got, "source"))
		group, ok := got["account_group"].(map[string]any)
		require.True(t, ok, "the group that decided should be named: %s", string(body))
		assert.Equal(t, groupID, jsonField(group, "id"))
	})

	t.Run("account", func(t *testing.T) {
		groupID := leadTimeAccountGroup(t, "e2e-lt-grp-c", nil)
		customerID := leadTimeCustomer(t, "e2e-lt-cust-c", nil, groupID)

		status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		got := parseJSON(body)
		assert.Equal(t, "account", jsonField(got, "source"))
		assert.NotEmpty(t, jsonField(got, "days"))
	})

	t.Run("unknown customer", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(customersPath+"/ac_doesnotexist000/lead-time", nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "an unknown customer must not 5xx: %s", string(body))
		assert.Equal(t, 404, status, "body: %s", string(body))
	})
}

// A cleared lead time hands the customer back to its group, rather than leaving the
// old value in place or falling all the way through to the account.
func TestCustomerLeadTime_ClearingReturnsCustomerToItsGroup(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-lt-clear-grp", ptrInt(21))
	customerID := leadTimeCustomer(t, "e2e-lt-clear-cust", ptrInt(5), groupID)

	status, body, err := apiClient.Patch(customersPath+"/"+customerID,
		map[string]any{"lead_time_days": nil}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "clearing must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, "account_group", jsonField(got, "source"))
	assert.Equal(t, "21", jsonField(got, "days"))
}

// Omitting the field on an update must not be read as clearing it — that would wipe
// a contractual lead time on any unrelated edit to the customer.
func TestCustomerLeadTime_OmittingOnUpdatePreservesIt(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-lt-preserve", ptrInt(12), "")

	status, body, err := apiClient.Patch(customersPath+"/"+customerID,
		map[string]any{"note": "unrelated edit"}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "update must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, "customer", jsonField(got, "source"))
	assert.Equal(t, "12", jsonField(got, "days"))
}

// Zero is a real commitment — ship same day — and must not be mistaken for "unset"
// and fall through to a laxer rule.
func TestCustomerLeadTime_ZeroIsSameDayNotUnset(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-lt-zero-grp", ptrInt(30))
	customerID := leadTimeCustomer(t, "e2e-lt-zero-cust", ptrInt(0), groupID)

	status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "customer", jsonField(got, "source"))
	assert.Equal(t, "0", jsonField(got, "days"))
}

func TestSalesOrders_ShipByFilters(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-shipby-filter", ptrInt(30), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderID := jsonField(order, "id")
	shipBy := shipByDate(t, order)
	require.NotEmpty(t, shipBy)

	parsed, err := time.Parse("2006-01-02", shipBy)
	require.NoError(t, err)
	dayBefore := parsed.AddDate(0, 0, -1).Format("2006-01-02")
	dayAfter := parsed.AddDate(0, 0, 1).Format("2006-01-02")

	t.Run("ship_by_after includes its own date", func(t *testing.T) {
		assert.True(t, orderAppearsInFilteredList(t, orderID, url.Values{"ship_by_after": {shipBy}}))
	})

	t.Run("ship_by_after excludes earlier commitments", func(t *testing.T) {
		assert.False(t, orderAppearsInFilteredList(t, orderID, url.Values{"ship_by_after": {dayAfter}}))
	})

	t.Run("ship_by_before includes its own date", func(t *testing.T) {
		assert.True(t, orderAppearsInFilteredList(t, orderID, url.Values{"ship_by_before": {shipBy}}))
	})

	t.Run("ship_by_before excludes later commitments", func(t *testing.T) {
		assert.False(t, orderAppearsInFilteredList(t, orderID, url.Values{"ship_by_before": {dayBefore}}))
	})

	// The order is due 30 days out, so it is not past due and must be absent from
	// past_due=true and present in past_due=false.
	t.Run("past_due excludes an order still within its window", func(t *testing.T) {
		assert.False(t, orderAppearsInFilteredList(t, orderID, url.Values{"past_due": {"true"}}))
		assert.True(t, orderAppearsInFilteredList(t, orderID, url.Values{"past_due": {"false"}}))
	})
}

// orderAppearsInFilteredList pages the sales-order list under the given filter,
// looking for one order. Every page is walked rather than only the first, because a
// parallel suite is creating and deleting orders throughout (see
// [[project_list_hydration_race]]).
func orderAppearsInFilteredList(t *testing.T, orderID string, filters url.Values) bool {
	t.Helper()

	params := url.Values{}
	for k, v := range filters {
		params[k] = v
	}
	params.Set("limit", "100")

	for range 20 {
		status, body, err := apiClient.GetListRaw(salesOrdersPath, params)
		require.NoError(t, err)
		require.Less(t, status, 500, "list must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		parsed := parseJSON(body)
		data, _ := parsed["data"].([]any)
		for _, raw := range data {
			row, ok := raw.(map[string]any)
			if ok && jsonField(row, "id") == orderID {
				return true
			}
		}

		pageInfo, _ := parsed["page_info"].(map[string]any)
		next := jsonField(pageInfo, "next_cursor")
		if next == "" {
			return false
		}
		params.Set("cursor", next)
	}
	return false
}

func ptrInt(v int) *int { return &v }

// past_due is the backlog question: an order whose date has passed and which is
// still owed. An order issued against a date already behind us has to appear in it,
// or the queue a planner works from is empty while orders are late.
func TestSalesOrders_PastDueFilterFindsALateOrder(t *testing.T) {
	t.Parallel()

	// Promised a week ago, so the commitment is already behind us the moment it is
	// issued. The promised date beats the customer's standing lead time.
	promised := time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02") + "T00:00:00Z"
	customerID := leadTimeCustomer(t, "e2e-pastdue", ptrInt(30), "")
	order := issueOrderForCustomer(t, customerID, map[string]any{"promised_at": promised})
	orderID := jsonField(order, "id")
	// The shipping calendar can only pull a ship-by date back onto an earlier open day, never
	// forward, so a promise a week old stays in the past whichever weekday the suite runs on.
	committed := shipByDate(t, order)
	require.NotEmpty(t, committed, "precondition: the order carries a commitment")
	require.LessOrEqual(t, committed, promised[:10], "precondition: the order is committed to a past date")

	assert.True(t, orderAppearsInFilteredList(t, orderID, url.Values{"past_due": {"true"}}),
		"an issued order past its ship-by date is past due")
	assert.False(t, orderAppearsInFilteredList(t, orderID, url.Values{"past_due": {"false"}}),
		"and must not also come back as not past due")

	// Unissuing withdraws the promise, so there is nothing left to be late for.
	status, body, err := apiClient.Put(salesOrdersPath+"/"+orderID+"/actions/unissue", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assert.False(t, orderAppearsInFilteredList(t, orderID, url.Values{"past_due": {"true"}}),
		"an order pulled back off the book is no longer late")
}

// An order with no commitment cannot be answered either way, so it must not be
// swept into the backlog by a filter that only knows about dates.
func TestSalesOrders_PastDueExcludesAnUncommittedOrder(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pastdue-estimate", ptrInt(30), "")
	body := minimalSalesOrderCreateBody(t, customerID)

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	orderID := jsonField(created, "id")
	deleteOrder(t, orderID)
	require.Equal(t, "estimate", jsonField(created, "status"), "precondition: nothing has been promised yet")
	require.Empty(t, shipByDate(t, created))

	assert.False(t, orderAppearsInFilteredList(t, orderID, url.Values{"past_due": {"true"}}),
		"an estimate carries no promise, so it cannot be past due")
}

// past_due is a boolean, and a value that is not one is refused rather than
// quietly read as false — which would answer a question about the backlog with the
// whole order book.
func TestSalesOrders_PastDueFilterValidatesItsInput(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath, url.Values{"past_due": {"maybe"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "a malformed filter must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "body: %s", string(body))
}

// A ship-by bound that is not a date is currently ignored rather than rejected,
// inherited from how starts_at/ends_at have always parsed: anything that does not
// match a date layout produces no filter at all.
//
// Pinned rather than left implicit because the failure mode is silent — a client
// sending a malformed date gets the unfiltered list back and reads it as an answer.
// The day this is tightened into a 400 this test fails, which is the point: it
// should be a deliberate change rather than one nobody noticed.
func TestSalesOrders_MalformedShipByFilterIsIgnored(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-shipby-malformed", ptrInt(30), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderID := jsonField(order, "id")
	require.NotEmpty(t, shipByDate(t, order))

	for _, tc := range []struct {
		name   string
		params url.Values
	}{
		{"ship_by_after not a date", url.Values{"ship_by_after": {"last-tuesday"}}},
		{"ship_by_before not a date", url.Values{"ship_by_before": {"2026-13-45"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.GetListRaw(salesOrdersPath, tc.params)
			require.NoError(t, err)
			require.Less(t, status, 500, "a malformed filter must not 5xx: %s", string(body))
			requireStatus(t, 200, status, body)

			// Ignored, not applied: the order is still there, exactly as it would be with
			// no filter at all.
			assert.True(t, orderAppearsInFilteredList(t, orderID, tc.params),
				"an unparseable bound narrows nothing, so the list comes back unfiltered")
		})
	}
}

// The two date bounds have to compose: a window that brackets a commitment
// includes it, and one that excludes it on either side does not.
func TestSalesOrders_ShipByFiltersCompose(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-shipby-window", ptrInt(30), "")
	order := issueOrderForCustomer(t, customerID, nil)
	orderID := jsonField(order, "id")
	shipBy := shipByDate(t, order)
	require.NotEmpty(t, shipBy)

	parsed, err := time.Parse("2006-01-02", shipBy)
	require.NoError(t, err)
	weekBefore := parsed.AddDate(0, 0, -7).Format("2006-01-02")
	weekAfter := parsed.AddDate(0, 0, 7).Format("2006-01-02")

	assert.True(t, orderAppearsInFilteredList(t, orderID, url.Values{
		"ship_by_after": {weekBefore}, "ship_by_before": {weekAfter},
	}), "a window that brackets the commitment must include it")

	assert.False(t, orderAppearsInFilteredList(t, orderID, url.Values{
		"ship_by_after": {weekAfter}, "ship_by_before": {parsed.AddDate(0, 0, 14).Format("2006-01-02")},
	}), "a window entirely after the commitment must not include it")
}
