//go:build e2e

package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Commitment bases: an order commits through exactly one of four rules — the
// customer's standing lead time, a lead time set on this order alone, a promised
// delivery date, or a pinned ship date. The first is inherited; the other three
// are mutually exclusive inputs, because they are alternative answers to the same
// question and combining them has no meaning.
//
// Only a promised delivery date has carrier transit subtracted from it. Every
// other basis already names a ship date or a ship lead time, so deducting the
// journey again would charge for it twice.

const quoteCommitmentPath = salesOrdersPath + "/actions/quote-commitment"

// dateOnly takes the calendar day out of an API timestamp. ship_by_date is a date, but it serializes as a full RFC3339 instant, so comparing the whole string against a day would only ever match by accident.
func dateOnly(timestamp string) string {
	if len(timestamp) < 10 {
		return timestamp
	}
	return timestamp[:10]
}

// futureWeekday returns a date comfortably ahead that is not a weekend, so a case testing one rule is not silently also testing the weekend snap.
func futureWeekday(daysOut int) time.Time {
	d := time.Now().UTC().AddDate(0, 0, daysOut)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func createOrderWithBasis(t *testing.T, customerID string, basis map[string]any) (int, []byte) {
	t.Helper()

	body := minimalSalesOrderCreateBody(t, customerID)
	for k, v := range basis {
		body[k] = v
	}

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "order create must not 5xx: %s", string(respBody))
	if status == 201 {
		deleteOrder(t, jsonField(parseJSON(respBody), "id"))
	}
	return status, respBody
}

func TestCommitmentBasis_LeadTimeOverrideReplacesTheCustomerChain(t *testing.T) {
	t.Parallel()

	status, body := createOrderWithBasis(t, SeedCustomerAccountID, map[string]any{
		"lead_time_override_days": 7,
	})
	requireStatus(t, 201, status, body)

	orderID := jsonField(parseJSON(body), "id")
	assert.Equal(t, "7", jsonField(parseJSON(body), "lead_time_override_days"))

	issued := issueOrder(t, orderID)
	assert.Equal(t, "order_lead_time", jsonField(issued, "lead_time_source"),
		"an order-level lead time must win the chain: %v", issued)
	assert.NotEmpty(t, jsonField(issued, "ship_by_date"))
	// A lead time is already a ship lead time; subtracting transit would deduct the journey twice.
	assert.Empty(t, jsonField(issued, "transit_days"), "a lead-time basis takes no transit")
}

func TestCommitmentBasis_ShipByPinIsTakenAsGiven(t *testing.T) {
	t.Parallel()

	pinned := futureWeekday(30)

	status, body := createOrderWithBasis(t, SeedCustomerAccountID, map[string]any{
		"ship_by_override_date": pinned.Format("2006-01-02") + "T00:00:00Z",
	})
	requireStatus(t, 201, status, body)

	issued := issueOrder(t, jsonField(parseJSON(body), "id"))
	assert.Equal(t, "order_ship_by", jsonField(issued, "lead_time_source"))
	assert.Equal(t, pinned.Format("2006-01-02"), dateOnly(jsonField(issued, "ship_by_date")),
		"a pinned ship date on an open day must be taken as given: %v", issued)
	assert.Empty(t, jsonField(issued, "transit_days"), "a pinned ship date takes no transit")
}

// A pinned Saturday resolves to the Friday before rather than standing as a date nobody can ship on.
func TestCommitmentBasis_ShipByPinSnapsOffAClosedDay(t *testing.T) {
	t.Parallel()

	saturday := time.Now().UTC().AddDate(0, 0, 30)
	for saturday.Weekday() != time.Saturday {
		saturday = saturday.AddDate(0, 0, 1)
	}
	saturday = time.Date(saturday.Year(), saturday.Month(), saturday.Day(), 0, 0, 0, 0, time.UTC)

	status, body := createOrderWithBasis(t, SeedCustomerAccountID, map[string]any{
		"ship_by_override_date": saturday.Format("2006-01-02") + "T00:00:00Z",
	})
	requireStatus(t, 201, status, body)

	issued := issueOrder(t, jsonField(parseJSON(body), "id"))
	want := saturday.AddDate(0, 0, -1).Format("2006-01-02")
	assert.Equal(t, want, dateOnly(jsonField(issued, "ship_by_date")),
		"a Saturday pin must resolve back to Friday: %v", issued)
	assert.Equal(t, "1", jsonField(issued, "calendar_adjustment_days"),
		"the day lost to the snap must be reported: %v", issued)
}

func TestCommitmentBasis_SettingTwoBasesIsRejected(t *testing.T) {
	t.Parallel()

	when := futureWeekday(30).Format("2006-01-02") + "T00:00:00Z"

	for name, basis := range map[string]map[string]any{
		"promised date and lead time": {"promised_at": when, "lead_time_override_days": 7},
		"promised date and ship-by":   {"promised_at": when, "ship_by_override_date": when},
		"lead time and ship-by":       {"lead_time_override_days": 7, "ship_by_override_date": when},
	} {
		t.Run(name, func(t *testing.T) {
			status, body := createOrderWithBasis(t, SeedCustomerAccountID, basis)
			assert.Equal(t, 400, status, "%s must be rejected: %s", name, string(body))
		})
	}
}

// The conflict is judged on what the order will hold, not just on what one request named: adding a second basis to an order that already carries one is the same conflict.
func TestCommitmentBasis_AddingASecondBasisByPatchIsRejected(t *testing.T) {
	t.Parallel()

	status, body := createOrderWithBasis(t, SeedCustomerAccountID, map[string]any{
		"lead_time_override_days": 7,
	})
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")

	when := futureWeekday(30).Format("2006-01-02") + "T00:00:00Z"
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{
		"promised_at": when,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "patch must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "adding a second basis must be rejected: %s", string(body))
}

// Clearing one basis while setting another in the same request is how a caller switches, and must succeed.
func TestCommitmentBasis_SwitchingBasisInOnePatchSucceeds(t *testing.T) {
	t.Parallel()

	status, body := createOrderWithBasis(t, SeedCustomerAccountID, map[string]any{
		"lead_time_override_days": 7,
	})
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")

	when := futureWeekday(30).Format("2006-01-02") + "T00:00:00Z"
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{
		"lead_time_override_days": nil,
		"promised_at":             when,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "patch must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Empty(t, jsonField(updated, "lead_time_override_days"))
	assert.NotEmpty(t, jsonField(updated, "promised_at"))
}

// Moving a basis on a live order re-stamps its commitment, and the response says so.
//
// The stamp writes the new ship-by straight to the row, behind the copy the update read, so a
// response built from that copy answers a renegotiated date with the date it replaced. A caller
// that trusts the response — the order page does, writing it straight into its cache — shows the
// old date until something else refetches. All three bases are re-stamped through the one re-read,
// so all three are checked here rather than only the one that was reported.
func TestCommitmentBasis_PatchingABasisMovesTheStampedShipBy(t *testing.T) {
	t.Parallel()

	// Far enough out that no basis here can land on the date the seed account's standing lead time
	// already committed to, which would make a stale response indistinguishable from a fresh one.
	when := futureWeekday(60)
	whenDay := when.Format("2006-01-02")

	cases := map[string]struct {
		patch      map[string]any
		wantSource string
		// The date the basis resolves to, where that is knowable from the request alone. A promised
		// delivery date is not: carrier transit is subtracted from it, and how long the seed lane
		// takes is the carrier's answer rather than this test's. Those cases assert the response
		// agrees with the stored order, which is the property under test either way.
		wantShipBy func(t *testing.T, patched map[string]any) string
	}{
		"a lead time for this order": {
			patch:      map[string]any{"lead_time_override_days": 33},
			wantSource: "order_lead_time",
			wantShipBy: func(t *testing.T, patched map[string]any) string { return expectedShipBy(t, patched, 33) },
		},
		"a pinned ship date": {
			patch:      map[string]any{"ship_by_override_date": whenDay + "T00:00:00Z"},
			wantSource: "order_ship_by",
			wantShipBy: func(*testing.T, map[string]any) string { return whenDay },
		},
		"a promised delivery date": {
			patch:      map[string]any{"promised_at": whenDay + "T00:00:00Z"},
			wantSource: "manual",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			issued := issueOrderForCustomer(t, SeedCustomerAccountID, nil)
			orderID := jsonField(issued, "id")
			require.NotEmpty(t, shipByDate(t, issued), "precondition: the issued order carries a commitment")

			status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, tc.patch, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "patch must not 5xx: %s", string(body))
			requireStatus(t, 200, status, body)

			patched := parseJSON(body)
			assert.Equal(t, tc.wantSource, jsonField(patched, "lead_time_source"))
			assert.NotEqual(t, shipByDate(t, issued), shipByDate(t, patched),
				"the response must not still carry the ship-by the basis replaced: %v", patched)
			if tc.wantShipBy != nil {
				assert.Equal(t, tc.wantShipBy(t, patched), shipByDate(t, patched),
					"the patch response must carry the re-stamped ship-by: %v", patched)
			}

			resp, err := apiClient.GetFull(salesOrdersPath+"/"+orderID, nil)
			require.NoError(t, err)
			requireStatus(t, 200, resp.StatusCode, resp.Body)

			fetched := parseJSON(resp.Body)
			assert.Equal(t, shipByDate(t, patched), shipByDate(t, fetched),
				"the patch response must match what a read of the order returns")
			assert.Equal(t, jsonField(patched, "lead_time_days"), jsonField(fetched, "lead_time_days"))
			assert.Equal(t, jsonField(patched, "lead_time_source"), jsonField(fetched, "lead_time_source"))
		})
	}
}

func TestQuoteCommitment_NamesTheDateAndExplainsIt(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(quoteCommitmentPath, map[string]any{
		"buyer_account_id":        SeedCustomerAccountID,
		"lead_time_override_days": 10,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "quote must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	quote := parseJSON(body)
	assert.Equal(t, "sales_order_commitment_quote", jsonField(quote, "object"))
	assert.NotEmpty(t, jsonField(quote, "ship_by_date"), "a quote must name a date: %s", string(body))
	assert.Equal(t, "order_lead_time", jsonField(quote, "lead_time_source"))
	// The derivation is what lets a form explain the date rather than restate the rules.
	assert.NotEmpty(t, jsonArray(quote, "steps"), "a quote must explain itself: %s", string(body))
}

// The whole point of the preview: it runs the same resolution the issue path runs, so what a rep is shown is what the order gets.
func TestQuoteCommitment_MatchesWhatIssueStamps(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(quoteCommitmentPath, map[string]any{
		"buyer_account_id":        SeedCustomerAccountID,
		"lead_time_override_days": 12,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	quoted := jsonField(parseJSON(body), "ship_by_date")

	createStatus, createBody := createOrderWithBasis(t, SeedCustomerAccountID, map[string]any{
		"lead_time_override_days": 12,
	})
	requireStatus(t, 201, createStatus, createBody)

	issued := issueOrder(t, jsonField(parseJSON(createBody), "id"))
	assert.Equal(t, dateOnly(quoted), dateOnly(jsonField(issued, "ship_by_date")),
		"the preview must agree with what issue stamped: %v", issued)
}

func TestQuoteCommitment_RejectsTwoBases(t *testing.T) {
	t.Parallel()

	when := futureWeekday(30).Format("2006-01-02") + "T00:00:00Z"

	status, body, err := apiClient.Post(quoteCommitmentPath, map[string]any{
		"buyer_account_id":        SeedCustomerAccountID,
		"promised_at":             when,
		"lead_time_override_days": 7,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "quote must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a quote naming two bases must be rejected: %s", string(body))
}

// Unissuing hands the order back to the customer's standing lead time, so no stamped commitment survives.
func TestCommitmentBasis_UnissueClearsTheCommitment(t *testing.T) {
	t.Parallel()

	status, body := createOrderWithBasis(t, SeedCustomerAccountID, map[string]any{
		"lead_time_override_days": 7,
	})
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")

	issued := issueOrder(t, orderID)
	require.NotEmpty(t, jsonField(issued, "ship_by_date"))

	unissueOrder(t, orderID)

	resp, err := apiClient.GetFull(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	after := parseJSON(resp.Body)
	assert.Empty(t, jsonField(after, "ship_by_date"), "unissue must clear the commitment: %v", after)
	// The basis is an input, not a commitment, so it survives for the next issue.
	assert.Equal(t, "7", jsonField(after, "lead_time_override_days"))
}
