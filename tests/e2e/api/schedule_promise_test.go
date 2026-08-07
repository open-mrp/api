//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Promises on the plan: which campaign is building which order, what the plan does
// not build in time, and the earliest date it could ship something new.

const quotePromiseDatePath = "/v1/operations/production-schedules/actions/quote-promise-date"

// The at-risk report has to describe the version it belongs to, not a fresh solve.
func TestSchedulePromises_AtRiskOrdersDescribeTheVersion(t *testing.T) {
	t.Parallel()

	schedule := generateSchedule(t, map[string]any{})
	scheduleID := jsonField(schedule, "id")
	cleanupSchedule(t, scheduleID)

	status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/"+scheduleID+"/at-risk-orders", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "at-risk orders must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	data, ok := parseJSON(body)["data"].([]any)
	require.True(t, ok, "at-risk orders must serialize as a list: %s", string(body))

	validReasons := []string{"past_due", "undated", "short"}
	previousWeek := -1
	for _, raw := range data {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "schedule_order_coverage", jsonField(row, "object"))
		assert.Contains(t, validReasons, jsonField(row, "reason"), "every at-risk order must say why")

		order, ok := row["sales_order"].(map[string]any)
		require.True(t, ok, "an at-risk order must name the order: %v", row)
		assert.NotEmpty(t, jsonField(order, "id"))

		// The quantity at risk must be the shortfall, not a placeholder.
		units, ok := row["units_at_risk"].(float64)
		require.True(t, ok)
		assert.Positive(t, units, "an order reported at risk must have a quantity at risk")

		// Soonest first: that is the order they have to be worked.
		week, ok := row["due_week"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, int(week), previousWeek, "at-risk orders must be sorted by due week")
		previousWeek = int(week)

		covering, ok := row["covering_lines"].(map[string]any)
		require.True(t, ok, "covering_lines must be a list resource, not null: %v", row["covering_lines"])
		lines, ok := covering["data"].([]any)
		require.True(t, ok)
		for _, rawLine := range lines {
			line, ok := rawLine.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "schedule_order_coverage_line", jsonField(line, "object"))
			allocated, ok := line["allocated_quantity"].(float64)
			require.True(t, ok)
			assert.Positive(t, allocated, "a campaign earmarked for an order must be earmarked for some quantity")
		}
	}
}

// A version that made no commitments cannot break any, so an empty answer is correct
// rather than an error.
func TestSchedulePromises_AtRiskOrdersRejectsUnknownSchedule(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/pnsc_doesnotexist00/at-risk-orders", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown schedule must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "body: %s", string(body))
}

// Capable-to-promise: a date drawn from the published plan, allowing for finishing.
//
// Publishes its own version rather than hoping one exists, so the quote path is
// actually exercised — a test that takes the "nothing published" branch proves only
// that the refusal works. Both quotes run against that one version: generating and
// publishing is a full solve, and doing it twice would hold the publishing lock
// twice as long for no extra coverage.
func TestSchedulePromises_QuoteComesFromThePublishedPlan(t *testing.T) {
	lockPublishing(t)

	schedule := ownedScheduleLocked(t, uniqueName("e2e-promise"))
	scheduleID := jsonField(schedule, "id")

	pubStatus, pubBody, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, pubStatus, pubBody)
	require.Equal(t, "published", jsonField(parseJSON(pubBody), "status"))

	t.Run("quotes against the published version", func(t *testing.T) {
		status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
			"item_id":  SeedGreigeItemID,
			"quantity": 1,
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "quoting must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		quote := parseJSON(body)
		assert.Equal(t, "promise_date_quote", jsonField(quote, "object"))
		assert.Equal(t, "1", jsonField(quote, "quantity"))

		quotedFrom, ok := quote["production_schedule"].(map[string]any)
		require.True(t, ok, "a quote must name the version it came from: %v", quote)
		assert.Equal(t, scheduleID, jsonField(quotedFrom, "id"),
			"the quote must come from the published version, not a draft")

		promisable, ok := quote["is_promisable"].(bool)
		require.True(t, ok)
		if promisable {
			assert.NotEmpty(t, jsonField(quote, "earliest_ship_date"),
				"a promisable quantity must carry a date")
		} else {
			assert.Empty(t, jsonField(quote, "earliest_ship_date"),
				"a quantity the horizon cannot supply must not be given a date")
		}
	})

	// A quantity far beyond what any plan holds is not promisable, and must come back
	// without a date rather than with an invented one.
	t.Run("an unreachable quantity gets no date", func(t *testing.T) {
		status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
			"item_id":  SeedGreigeItemID,
			"quantity": 100000000,
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		quote := parseJSON(body)
		assert.Equal(t, false, quote["is_promisable"],
			"a hundred million units is beyond any horizon")
		assert.Empty(t, jsonField(quote, "earliest_ship_date"))
	})
}

func TestSchedulePromises_QuoteValidatesItsInput(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"zero quantity", map[string]any{"item_id": SeedGreigeItemID, "quantity": 0}},
		{"negative quantity", map[string]any{"item_id": SeedGreigeItemID, "quantity": -5}},
		{"missing item", map[string]any{"quantity": 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(quotePromiseDatePath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Equal(t, 400, status, "body: %s", string(body))
		})
	}
}

// The at-risk report is a collection like any other, and a version that made no
// commitments it could break must come back as an empty list rather than a null a
// client has to guard against.
func TestSchedulePromises_AtRiskOrdersIsAWellFormedList(t *testing.T) {
	t.Parallel()

	schedule := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	scheduleID := jsonField(schedule, "id")
	cleanupSchedule(t, scheduleID)

	status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/"+scheduleID+"/at-risk-orders", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"))
	data, ok := parsed["data"].([]any)
	require.True(t, ok, "data must be an array, not null: %v", parsed["data"])

	// Every row names an order that is genuinely short of what it needs, and carries
	// the commitment behind it when the order has one.
	for _, raw := range data {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		item, ok := row["item"].(map[string]any)
		require.True(t, ok, "an at-risk order must name the item that has to be built: %v", row)
		assert.NotEmpty(t, jsonField(item, "id"))
		assert.NotEmpty(t, jsonField(row, "sku"))

		if jsonField(row, "reason") == "undated" {
			assert.Empty(t, jsonField(row, "ship_by_date"),
				"an order reported undated must not also carry a date")
		}
	}
}

// A version belongs to the account that generated it, so another tenant asking
// about its commitments gets the same answer as for a version that does not exist.
func TestSchedulePromises_AtRiskOrdersTenantIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	schedule := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	scheduleID := jsonField(schedule, "id")
	cleanupSchedule(t, scheduleID)

	status, body, err := clientB.GetListRaw(productionSchedulesPath+"/"+scheduleID+"/at-risk-orders", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "cross-tenant read must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "tenant B must not read tenant A's commitments: %s", string(body))
}

// Quoting is a read of the published plan and reserves nothing, so two quotes taken
// back to back must agree. A date that moved because somebody asked twice would be
// a date nobody could act on.
func TestSchedulePromises_QuotingReservesNothing(t *testing.T) {
	lockPublishing(t)

	schedule := ownedScheduleLocked(t, uniqueName("e2e-promise-idem"))
	scheduleID := jsonField(schedule, "id")

	pubStatus, pubBody, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, pubStatus, pubBody)

	quote := func() map[string]any {
		status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
			"item_id":  SeedGreigeItemID,
			"quantity": 1,
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "quoting must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)
		return parseJSON(body)
	}

	first, second := quote(), quote()
	assert.Equal(t, first["is_promisable"], second["is_promisable"],
		"asking twice must not change whether the plan can supply the quantity")
	assert.Equal(t, jsonField(first, "earliest_ship_date"), jsonField(second, "earliest_ship_date"),
		"a quote does not consume supply, so the second quote gets the same date")
	assert.Equal(t, jsonField(first, "earliest_week_index"), jsonField(second, "earliest_week_index"))
	assert.Equal(t, scheduleID, jsonField(jsonObject(second, "production_schedule"), "id"))
}

// A quantity the plan already holds is promisable; a larger one is quoted no
// sooner, because supply the plan does not have cannot arrive earlier than supply
// it does.
func TestSchedulePromises_QuoteIsMonotonicInQuantity(t *testing.T) {
	lockPublishing(t)

	schedule := ownedScheduleLocked(t, uniqueName("e2e-promise-monotonic"))
	scheduleID := jsonField(schedule, "id")

	pubStatus, pubBody, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, pubStatus, pubBody)

	weekFor := func(quantity float64) (int, bool) {
		status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
			"item_id":  SeedGreigeItemID,
			"quantity": quantity,
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "quoting must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		quote := parseJSON(body)
		promisable, _ := quote["is_promisable"].(bool)
		if !promisable {
			return 0, false
		}
		week, ok := quote["earliest_week_index"].(float64)
		require.True(t, ok, "a promisable quote must say which week the constraint finishes in: %v", quote)
		return int(week), true
	}

	smallWeek, smallOK := weekFor(1)
	if !smallOK {
		t.Skip("the published plan builds none of this item, so there is nothing to compare")
	}

	largeWeek, largeOK := weekFor(100000)
	if !largeOK {
		// Not promisable at all is a valid answer for the larger quantity, and a
		// stronger one than a later week.
		return
	}
	assert.GreaterOrEqual(t, largeWeek, smallWeek,
		"a larger quantity cannot be ready before a smaller one")
}

func TestSchedulePromises_QuoteRejectsMalformedRequests(t *testing.T) {
	t.Parallel()

	t.Run("missing quantity", func(t *testing.T) {
		status, body, err := apiClient.Post(quotePromiseDatePath,
			map[string]any{"item_id": SeedGreigeItemID}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		assert.Equal(t, 400, status, "body: %s", string(body))
	})

	t.Run("empty item", func(t *testing.T) {
		status, body, err := apiClient.Post(quotePromiseDatePath,
			map[string]any{"item_id": "", "quantity": 10}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		assert.Equal(t, 400, status, "body: %s", string(body))
	})

	t.Run("unknown field", func(t *testing.T) {
		status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
			"item_id":         SeedGreigeItemID,
			"quantity":        10,
			bogusE2EJSONField: "x",
		}, newIdempotencyKey())
		require.NoError(t, err)
		assertJSONUnknownFieldRejected(t, "POST", quotePromiseDatePath, status, body)
	})
}

// An item id that does not exist is currently answered rather than rejected: the
// quote comes back promisable=false, which is the same answer a real item the plan
// does not build would get.
//
// Pinned rather than left implicit, because the two answers are not the same
// question — a typo in a SKU reads as a capacity problem — and because the outcome
// depends on there being a published plan at all. Without one the request fails on
// "nothing is published" and the item is never looked at, so this publishes first
// and asserts the branch that actually runs.
func TestSchedulePromises_QuoteAnswersAnUnknownItemAsUnpromisable(t *testing.T) {
	lockPublishing(t)

	schedule := ownedScheduleLocked(t, uniqueName("e2e-promise-unknown"))
	pubStatus, pubBody, err := apiClient.Put(schedulePath(jsonField(schedule, "id"))+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, pubStatus, pubBody)

	status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
		"item_id":  "it_doesnotexist0000",
		"quantity": 10,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	quote := parseJSON(body)
	assert.Equal(t, false, quote["is_promisable"],
		"nothing in any plan builds an item that does not exist")
	assert.Empty(t, jsonField(quote, "earliest_ship_date"),
		"and no date may be offered for it")
}
