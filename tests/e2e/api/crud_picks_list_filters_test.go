//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Covers the filter set the picking index page sends (pick.api.ts fetchPicks): q, status,
// customer_ids, product_line_ids, customer_group_ids, department_ids and the date window.
//
// A filter the server silently ignores still returns 200 with a full page, so every case pairs a
// positive match with a nonsense id that must narrow the list to nothing.
const (
	// PICK-001 is open and seeded today; PICK-002 and PICK-003 are finished, 2 and 4 days back.
	seedOpenPickID      = "pk_01k0a5tsn7f7psgagr1732fxqa"
	seedClosedPickID    = "pk_01k0a5tsn7ejfrwg5dnshzfwsx"
	seedOldClosedPickID = "pk_01k0a5tsn7eeht162chb2jcknc"
)

// Collects the ids returned by the pick list under the given filters.
func pickIDsFiltered(t *testing.T, params url.Values) []string {
	t.Helper()
	return listIDs(t, picksPath, params)
}

func TestPicksList_SearchMatchesPickNumber(t *testing.T) {
	t.Parallel()

	assert.Contains(t, pickIDsFiltered(t, url.Values{"q": {"PICK-002"}}), seedClosedPickID,
		"searching a pick number should surface that pick")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"q": {"zzz-no-such-pick-zzz"}}),
		"a search matching nothing must return nothing")
}

// `open` is a pick that has not been finished; `closed` is one that has.
func TestPicksList_StatusSplitsOpenFromClosed(t *testing.T) {
	t.Parallel()

	open := pickIDsFiltered(t, url.Values{"status": {"open"}})
	assert.Contains(t, open, seedOpenPickID)
	assert.NotContains(t, open, seedClosedPickID, "a finished pick must not appear under open")

	closed := pickIDsFiltered(t, url.Values{"status": {"closed"}})
	assert.Contains(t, closed, seedClosedPickID)
	assert.NotContains(t, closed, seedOpenPickID, "an unfinished pick must not appear under closed")
}

func TestPicksList_FiltersByCustomerAndGroup(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, pickIDsFiltered(t, url.Values{"customer_ids": {SeedCustomerAccountID}}),
		"the seeded customer has picks")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"customer_ids": {"ac_01nosuchcustomer000"}}),
		"an unknown customer must narrow the list to nothing rather than be ignored")

	assert.NotEmpty(t, pickIDsFiltered(t, url.Values{"customer_group_ids": {SeedCustomerGroupID}}),
		"the seeded customer belongs to the seeded group")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"customer_group_ids": {"acgp_01nosuchgroup0000"}}))
}

// Both reach through the pick's order lines to the product behind them.
func TestPicksList_FiltersByDepartmentAndProductLine(t *testing.T) {
	t.Parallel()

	assert.Contains(t, pickIDsFiltered(t, url.Values{"department_ids": {SeedDepartmentID}}), seedOpenPickID,
		"the seeded department owns the seeded picks")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"department_ids": {"dp_01nosuchdepartment"}}))

	assert.NotEmpty(t, pickIDsFiltered(t, url.Values{"product_line_ids": {SeedProductLineID}}),
		"the seeded product line is picked on at least one pick")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"product_line_ids": {"pdln_01nosuchline00000"}}))
}

// The window filters on creation. An end date is inclusive of that whole day, so a pick created
// today still matches ends_at=today rather than being cut off at midnight.
func TestPicksList_FiltersByCreatedDateWindow(t *testing.T) {
	t.Parallel()

	today := time.Now().UTC().Format("2006-01-02")
	inWindow := pickIDsFiltered(t, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {today}})
	assert.Contains(t, inWindow, seedOpenPickID, "a window ending today must include a pick created today")

	// PICK-003 is four days old, so a window opening yesterday leaves it behind.
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	recent := pickIDsFiltered(t, url.Values{"starts_at": {yesterday}})
	assert.Contains(t, recent, seedOpenPickID)
	assert.NotContains(t, recent, seedOldClosedPickID, "a pick created four days ago is outside the window")

	assert.Empty(t, pickIDsFiltered(t, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {"2000-01-02"}}),
		"a window that closed decades ago must exclude every pick")
}
