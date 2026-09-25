//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Covers the filter set the picking index page sends (pick.api.ts fetchPicks): q, status,
// customer_ids, product_line_ids, customer_group_ids and the date window.
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
		"searching a full pick number should surface that pick")
	// The ngram search matches a substring, not just a prefix or whole token: "CK-002" sits in the
	// middle of "PICK-002".
	assert.Contains(t, pickIDsFiltered(t, url.Values{"q": {"CK-002"}}), seedClosedPickID,
		"searching a substring of a pick number should surface that pick")
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

// Reaches through the pick's order lines to the product behind them.
func TestPicksList_FiltersByProductLine(t *testing.T) {
	t.Parallel()

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

// A 1-2 character term is too common as a substring to page quickly, so it matches pick numbers by
// prefix only; 3+ characters keep substring search. Scoped to a fresh customer so the result is
// exactly one pick or none.
func TestPicksList_ShortSearchMatchesPickNumberPrefix(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-prefix", nil, "")
	_, pickID := issuedOrderAndPick(t, customerID, nil)
	number := jsonField(retrievePick(t, pickID), "number")
	require.GreaterOrEqual(t, len(number), 4, "the pick number must be long enough to split into a prefix and a non-prefix")

	search := func(q string) []string {
		return pickIDsFiltered(t, url.Values{"q": {q}, "customer_ids": {customerID}})
	}

	assert.Equal(t, []string{pickID}, search(number[:1]), "a one-character prefix of the number should match")
	assert.Equal(t, []string{pickID}, search(number[:2]), "a two-character prefix of the number should match")

	tail := number[len(number)-2:]
	if !strings.HasPrefix(strings.ToLower(number), strings.ToLower(tail)) {
		assert.Empty(t, search(tail), "a two-character term that is not a prefix must not match")
	}
	assert.Equal(t, []string{pickID}, search(number[len(number)-3:]), "three characters match anywhere in the number")
}

// The customer filter reads the pick's own copy of its order's buyer, so a merge that moves the
// order to another customer has to move the pick with it.
func TestPicksList_CustomerFilterFollowsAMerge(t *testing.T) {
	t.Parallel()

	targetID := leadTimeCustomer(t, "e2e-pick-merge-target", nil, "")
	sourceID := leadTimeCustomer(t, "e2e-pick-merge-source", nil, "")
	_, pickID := issuedOrderAndPick(t, sourceID, nil)
	require.Equal(t, []string{pickID}, pickIDsFiltered(t, url.Values{"customer_ids": {sourceID}}))

	status, body, err := apiClient.Post(customersPath+"/"+targetID+"/actions/merge", map[string]any{
		"source_customer_ids": []string{sourceID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assert.Equal(t, []string{pickID}, pickIDsFiltered(t, url.Values{"customer_ids": {targetID}}),
		"the merged customer's pick should list under the target")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"customer_ids": {sourceID}}),
		"nothing should list under the merged-away customer")
}
