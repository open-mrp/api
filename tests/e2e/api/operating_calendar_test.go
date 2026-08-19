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

// Operating calendars: the days a plant tenders freight and a customer's dock
// accepts it. Every ship-by date is resolved against them, so an order is never
// committed to a day nobody can act on.
//
// The account under test is seeded with Monday-to-Friday defaults at
// registration, which is why the pre-existing ship-by tests are unaffected — that
// is exactly the week they were written against. These tests change the calendars
// and watch the dates move.

const operatingCalendarsPath = "/v1/operations/operating-calendars"

// createCalendar makes a calendar and cleans it up. Codes are unique per account, so every test gets its own.
func createCalendar(t *testing.T, kind, daysOfWeek string, extra map[string]any) string {
	t.Helper()

	body := map[string]any{
		"code":         uniqueName("e2e-cal"),
		"name":         "E2E " + kind + " calendar",
		"kind":         kind,
		"days_of_week": daysOfWeek,
	}
	for k, v := range extra {
		body[k] = v
	}

	status, respBody, err := apiClient.Post(operatingCalendarsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "calendar create must not 5xx: %s", string(respBody))
	requireStatus(t, 201, status, respBody)

	id := jsonField(parseJSON(respBody), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(operatingCalendarsPath + "/" + id) })
	return id
}

func TestOperatingCalendar_CreateReadUpdate(t *testing.T) {
	t.Parallel()

	calendarID := createCalendar(t, "ship", "1111000", map[string]any{
		"cutoff_at": "15:00",
		"timezone":  "America/Chicago",
	})

	resp, err := apiClient.GetFull(operatingCalendarsPath+"/"+calendarID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	assert.Equal(t, "operating_calendar", jsonField(got, "object"))
	assert.Equal(t, "ship", jsonField(got, "kind"))
	assert.Equal(t, "1111000", jsonField(got, "days_of_week"))
	assert.Equal(t, "15:00", jsonField(got, "cutoff_at"))
	assert.Equal(t, "America/Chicago", jsonField(got, "timezone"))

	status, body, err := apiClient.Patch(operatingCalendarsPath+"/"+calendarID, map[string]any{
		"days_of_week": "1111100",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "1111100", jsonField(parseJSON(body), "days_of_week"))
}

// A mask nothing can ship on would send every commitment resolved against it to the snap-back limit before failing, so it is refused at the door.
func TestOperatingCalendar_RejectsACalendarThatNeverRuns(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(operatingCalendarsPath, map[string]any{
		"code":         uniqueName("e2e-cal-closed"),
		"name":         "Never runs",
		"kind":         "ship",
		"days_of_week": "0000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(body))
	assert.Equal(t, 400, status, "an all-closed mask must be rejected: %s", string(body))
}

func TestOperatingCalendar_RejectsCutoffOnAReceivingCalendar(t *testing.T) {
	t.Parallel()

	// A cutoff is when a plant hands freight over. A customer's dock has no equivalent, and accepting one would imply a rule nothing applies.
	status, body, err := apiClient.Post(operatingCalendarsPath, map[string]any{
		"code":         uniqueName("e2e-cal-recv"),
		"name":         "Receiving with a cutoff",
		"kind":         "receive",
		"days_of_week": "1111100",
		"cutoff_at":    "15:00",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a receiving calendar must reject a cutoff: %s", string(body))
}

func TestOperatingCalendar_RejectsUnknownTimezone(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(operatingCalendarsPath, map[string]any{
		"code":         uniqueName("e2e-cal-tz"),
		"name":         "Bad zone",
		"kind":         "ship",
		"days_of_week": "1111100",
		"timezone":     "Mars/Olympus_Mons",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(body))
	assert.Equal(t, 400, status, "an unloadable zone must be rejected: %s", string(body))
}

func TestOperatingCalendar_ClosuresRoundTrip(t *testing.T) {
	t.Parallel()

	calendarID := createCalendar(t, "ship", "1111100", nil)
	closuresPath := operatingCalendarsPath + "/" + calendarID + "/closures"
	closedOn := time.Now().UTC().AddDate(0, 2, 0).Format("2006-01-02")

	status, body, err := apiClient.Post(closuresPath, map[string]any{
		"closed_on": closedOn + "T00:00:00Z",
		"name":      "E2E shutdown",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "closure create must not 5xx: %s", string(body))
	requireStatus(t, 201, status, body)

	closureID := jsonField(parseJSON(body), "id")
	assert.Equal(t, "operating_calendar_closure", jsonField(parseJSON(body), "object"))

	listStatus, listBody, err := apiClient.GetListRaw(closuresPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	assert.Contains(t, string(listBody), "E2E shutdown")

	status, body, err = apiClient.Delete(closuresPath + "/" + closureID)
	require.NoError(t, err)
	require.Less(t, status, 500, "closure delete must not 5xx: %s", string(body))
	requireStatus(t, 204, status, body)
}

// Closing the same date twice is a no-op, which is what makes re-seeding a year safe and keeps a relabelled holiday relabelled.
func TestOperatingCalendar_ClosingTheSameDateTwiceIsIdempotent(t *testing.T) {
	t.Parallel()

	calendarID := createCalendar(t, "ship", "1111100", nil)
	closuresPath := operatingCalendarsPath + "/" + calendarID + "/closures"
	closedOn := time.Now().UTC().AddDate(0, 3, 0).Format("2006-01-02") + "T00:00:00Z"

	for i := range 2 {
		status, body, err := apiClient.Post(closuresPath, map[string]any{
			"closed_on": closedOn,
			"name":      fmt.Sprintf("E2E repeat %d", i),
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "closure create must not 5xx: %s", string(body))
		requireStatus(t, 201, status, body)
	}
}

// Deleting a calendar out from under its references would quietly return every affected order to a plain Monday-to-Friday week, which reads as the feature breaking rather than as a decision anybody made.
func TestOperatingCalendar_DeleteRefusedWhileReferenced(t *testing.T) {
	t.Parallel()

	calendarID := createCalendar(t, "receive", "1111100", nil)

	status, body, err := apiClient.Patch(customersPath+"/"+SeedCustomerAccountID, map[string]any{
		"receive_calendar_id": calendarID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "linking must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	t.Cleanup(func() {
		_, _, _ = apiClient.Patch(customersPath+"/"+SeedCustomerAccountID, map[string]any{"receive_calendar_id": nil}, newIdempotencyKey())
	})

	status, body, err = apiClient.Delete(operatingCalendarsPath + "/" + calendarID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	assert.Equal(t, 409, status, "a referenced calendar must not be deletable: %s", string(body))
}

func TestOperatingCalendar_ListFiltersByKind(t *testing.T) {
	t.Parallel()

	shipID := createCalendar(t, "ship", "1111000", nil)

	status, body, err := apiClient.GetListRaw(operatingCalendarsPath, url.Values{"kind": {"ship"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Contains(t, string(body), shipID)

	status, body, err = apiClient.GetListRaw(operatingCalendarsPath, url.Values{"kind": {"receive"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotContains(t, string(body), shipID, "a ship calendar must not appear under kind=receive")
}
