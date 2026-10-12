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
	assert.Contains(t, pickIDsFiltered(t, url.Values{"q": {"pick-00"}}), seedClosedPickID,
		"searching the start of a pick number, in any case, should surface that pick")
	// "CK-002" sits in the middle of "PICK-002": a prefix search passes it over, a contains search finds it.
	assert.NotContains(t, pickIDsFiltered(t, url.Values{"q": {"CK-002"}}), seedClosedPickID,
		"the default prefix search must not match the middle of a pick number")
	assert.Contains(t, pickIDsFiltered(t, url.Values{"q": {"CK-002"}, "q_match": {"contains"}}), seedClosedPickID,
		"a contains search should match the middle of a pick number")
	assert.Contains(t, pickIDsFiltered(t, url.Values{"q": {"PICK-002"}, "q_match": {"prefix"}}), seedClosedPickID,
		"prefix named explicitly is the default")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"q": {"zzz-no-such-pick-zzz"}}),
		"a search matching nothing must return nothing")
	assert.Empty(t, pickIDsFiltered(t, url.Values{"q": {"zzz-no-such-pick-zzz"}, "q_match": {"contains"}}),
		"a contains search matching nothing must return nothing")
}

// q_match is an enum: an unknown mode is rejected rather than read as the default.
func TestPicksList_UnknownSearchMatchIsRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath, url.Values{"q": {"PICK"}, "q_match": {"fuzzy"}})
	require.NoError(t, err)
	requireErrorParam(t, 400, status, body, "q_match")
}

// A customer PO number matches by its start under the default prefix search and anywhere under contains;
// the customer's name still matches anywhere in either mode.
func TestPicksList_SearchMatchModesOnPONumberAndCustomer(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-match", nil, "")
	po := uniqueName("PO-MATCH")
	_, pickID := issuedOrderAndPick(t, customerID, map[string]any{"customer_purchase_order_number": po})
	name := jsonField(retrieveCustomerForPickSearch(t, customerID), "name")
	require.Greater(t, len(po), 8, "the PO number must be long enough to split")
	require.Greater(t, len(name), 8, "the customer name must be long enough to split")

	search := func(q, match string) []string {
		params := url.Values{"q": {q}, "customer_ids": {customerID}}
		if match != "" {
			params.Set("q_match", match)
		}
		return pickIDsFiltered(t, params)
	}

	assert.Equal(t, []string{pickID}, search(po[:8], ""), "the start of the PO number matches by default")
	assert.Equal(t, []string{pickID}, search(strings.ToLower(po[:8]), ""), "a PO prefix matches in any case")
	assert.Empty(t, search(po[3:], ""), "the middle of the PO number does not match a prefix search")
	assert.Equal(t, []string{pickID}, search(po[3:], "contains"), "the middle of the PO number matches a contains search")
	assert.Equal(t, []string{pickID}, search(name[4:], ""), "the middle of the customer's name matches under prefix")
	assert.Equal(t, []string{pickID}, search(name[4:], "contains"), "the middle of the customer's name matches under contains")
}

// The FULLTEXT index holds no token with an "i" (stopwords), so "pic" and "tio" have none to search by;
// they still match, the start of the pick number (anywhere in it under contains) and the middle of the
// customer's name respectively. Both terms match every pick the suite makes for the seed customer, so
// the seed pick is looked for on every page, not just the newest hundred.
func TestPicksList_SearchMatchesATermTheIndexHoldsNoTokenFor(t *testing.T) {
	t.Parallel()

	assert.Contains(t, dashPicksAllIDs(t, apiClient, picksPath, url.Values{"q": {"pic"}}), seedClosedPickID,
		`"pic" is the start of PICK-002`)
	assert.Contains(t, dashPicksAllIDs(t, apiClient, picksPath, url.Values{"q": {"pic"}, "q_match": {"contains"}}), seedClosedPickID,
		`"pic" is a substring of PICK-002`)
	assert.Contains(t, dashPicksAllIDs(t, apiClient, picksPath, url.Values{"q": {"tio"}}), seedClosedPickID,
		`"tio" is a substring of PICK-002's customer, Global Manufacturing Solutions`)
}

// `open` is a pick that has not been finished; `closed` is one that has.
func TestPicksList_StatusSplitsOpenFromClosed(t *testing.T) {
	t.Parallel()

	open := url.Values{"status": {"open"}}
	assertListContainsID(t, picksPath, open, seedOpenPickID)
	assert.Nil(t, listFindByField(t, picksPath, open, "id", seedClosedPickID), "a finished pick must not appear under open")

	closed := url.Values{"status": {"closed"}}
	assertListContainsID(t, picksPath, closed, seedClosedPickID)
	assert.Nil(t, listFindByField(t, picksPath, closed, "id", seedOpenPickID), "an unfinished pick must not appear under closed")
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
	assert.NotNil(t, listFindByField(t, picksPath, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {today}}, "id", seedOpenPickID),
		"a window ending today must include a pick created today")

	// PICK-003 is four days old, so a window opening yesterday leaves it behind.
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	recent := url.Values{"starts_at": {yesterday}}
	assertListContainsID(t, picksPath, recent, seedOpenPickID)
	assert.Nil(t, listFindByField(t, picksPath, recent, "id", seedOldClosedPickID), "a pick created four days ago is outside the window")

	assert.Empty(t, pickIDsFiltered(t, url.Values{"starts_at": {"2000-01-01"}, "ends_at": {"2000-01-02"}}),
		"a window that closed decades ago must exclude every pick")
}

// A 1-2 character term is too common as a substring to page quickly, so it matches pick numbers by
// prefix only, whichever q_match is sent; 3+ characters match anywhere under contains. Scoped to a
// fresh customer so the result is exactly one pick or none.
func TestPicksList_ShortSearchMatchesPickNumberPrefix(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-prefix", nil, "")
	_, pickID := issuedOrderAndPick(t, customerID, nil)
	number := jsonField(retrievePick(t, pickID), "number")
	require.GreaterOrEqual(t, len(number), 4, "the pick number must be long enough to split into a prefix and a non-prefix")

	search := func(q string, match string) []string {
		params := url.Values{"q": {q}, "customer_ids": {customerID}}
		if match != "" {
			params.Set("q_match", match)
		}
		return pickIDsFiltered(t, params)
	}

	assert.Equal(t, []string{pickID}, search(number[:1], ""), "a one-character prefix of the number should match")
	assert.Equal(t, []string{pickID}, search(number[:2], ""), "a two-character prefix of the number should match")

	tail := number[len(number)-2:]
	if !strings.HasPrefix(strings.ToLower(number), strings.ToLower(tail)) {
		assert.Empty(t, search(tail, ""), "a two-character term that is not a prefix must not match")
		assert.Empty(t, search(tail, "contains"), "a two-character term matches only as a prefix, even under contains")
	}
	assert.Equal(t, []string{pickID}, search(number[:3], ""), "three characters match the start of the number")
	assert.Equal(t, []string{pickID}, search(number[len(number)-3:], "contains"), "three characters match anywhere in the number under contains")
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
