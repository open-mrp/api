//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests marked KNOWN BUG assert the behavior the endpoint should have and fail against the current
// implementation; the comment above each names the discrepancy.

func territoriesPath() string {
	return territoriesPathFor(SeedAccountID)
}

func territoriesPathFor(accountID string) string {
	return "/v1/sales/accounts/" + accountID + "/territories"
}

func territoryPath(territoryID string) string {
	return territoriesPath() + "/" + territoryID
}

// territoryIncludes is what the dashboard expands on every territory read and write: the rep, the rep's user
// (the rep's name comes from it), and the product line.
const territoryIncludes = "sales_rep,sales_rep.user,product_line"

func withTerritoryIncludes(path string) string {
	return path + "?include=" + territoryIncludes
}

// createTerritory creates a territory in the seed account, expanded as the dashboard asks, and deletes it
// when the test ends.
func createTerritory(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	status, resp, err := apiClient.Post(withTerritoryIncludes(territoriesPath()), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	created := parseJSON(resp)
	territoryID := jsonField(created, "id")
	require.NotEmpty(t, territoryID)
	t.Cleanup(func() { apiClient.Delete(territoryPath(territoryID)) })
	return created
}

func getTerritory(t *testing.T, territoryID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(territoryPath(territoryID), url.Values{"include": {territoryIncludes}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func patchTerritory(t *testing.T, territoryID string, body map[string]any) map[string]any {
	t.Helper()
	status, resp, err := apiClient.Patch(withTerritoryIncludes(territoryPath(territoryID)), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	return parseJSON(resp)
}

// territoryState is every value a territory holds, "" standing for null. Comparing whole states is what
// proves an edit to one field left the others alone.
type territoryState struct {
	state, start, end, salesRepID, productLineID string
}

func territoryStateOf(m map[string]any) territoryState {
	return territoryState{
		state:         jsonField(m, "state"),
		start:         jsonField(m, "start_zipcode"),
		end:           jsonField(m, "end_zipcode"),
		salesRepID:    jsonField(jsonObject(m, "sales_rep"), "id"),
		productLineID: jsonField(jsonObject(m, "product_line"), "id"),
	}
}

// createTerritoryProductLine creates a product line no other test's territories use, so searching its name
// scopes the territory list to this test's rows.
func createTerritoryProductLine(t *testing.T) (productLineID, name string) {
	t.Helper()
	name = uniqueName("e2e-terr-pl")
	created := createAndCleanup(t, productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	})
	return jsonField(created, "id"), name
}

// searchTerritoryIDs returns every territory a search for q finds, in list order.
func searchTerritoryIDs(t *testing.T, client *Client, path, q string) []string {
	t.Helper()
	list, _, err := client.GetList(path, url.Values{"q": {q}, "limit": {"1000"}})
	require.NoError(t, err)
	var found []string
	for page := 0; ; page++ {
		found = append(found, territoryIDsOf(list)...)
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil || page >= maxListScanPages {
			return found
		}
		list, _, err = client.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
	}
}

func territoryIDsOf(list *ListResponse) []string {
	found := make([]string, 0, len(list.Data))
	for _, item := range list.Data {
		found = append(found, DataItemField(item, "id"))
	}
	return found
}

// assertExpandedTerritoryRefs checks the expanded rep (with the user the dashboard names it from) and product line.
func assertExpandedTerritoryRefs(t *testing.T, territory map[string]any, repUserID, productLineName string) {
	t.Helper()
	rep := jsonObject(territory, "sales_rep")
	require.NotNil(t, rep, "sales_rep should be expanded: %v", territory)
	assert.Equal(t, "account_user", jsonField(rep, "object"))
	user := jsonObject(rep, "user")
	require.NotNil(t, user, "sales_rep.user should be expanded: %v", rep)
	assert.Equal(t, "user", jsonField(user, "object"))
	assert.Equal(t, repUserID, jsonField(user, "id"))

	pl := jsonObject(territory, "product_line")
	require.NotNil(t, pl, "product_line should be expanded: %v", territory)
	assert.Equal(t, "product_line", jsonField(pl, "object"))
	assert.Equal(t, productLineName, jsonField(pl, "name"))
}

// assertTerritoryErrorParam asserts a 400 whose error names one of params.
func assertTerritoryErrorParam(t *testing.T, status int, body []byte, params ...string) {
	t.Helper()
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assert.Contains(t, params, errObj["param"], "error.param should name the offending field: %s", body)
}

// ──────────────────────────────────────────────
// Create
// ──────────────────────────────────────────────

// The create form sends every field and the dashboard renders the territory from the response, so the
// response, a later read, and the list must all carry exactly what was sent.
func TestTerritories_CreateWithEveryFieldRoundTrips(t *testing.T) {
	t.Parallel()
	plID, plName := createTerritoryProductLine(t)
	state := uniqueName("e2e-terr-st")
	want := territoryState{state, "10001", "10999", SeedAccountUser2ID, plID}

	created := createTerritory(t, map[string]any{
		"state":           state,
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    SeedAccountUser2ID,
		"product_line_id": plID,
	})
	territoryID := jsonField(created, "id")
	assertIDFormat(t, territoryID, id.TerritoryIDPrefix)
	assertObjectField(t, created, "territory")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	assert.Equal(t, want, territoryStateOf(created))
	assertExpandedTerritoryRefs(t, created, SeedUser2ID, plName)

	got := getTerritory(t, territoryID)
	assert.Equal(t, want, territoryStateOf(got))
	assert.Equal(t, jsonField(created, "created_at"), jsonField(got, "created_at"))
	assert.Equal(t, jsonField(created, "updated_at"), jsonField(got, "updated_at"))
	assertExpandedTerritoryRefs(t, got, SeedUser2ID, plName)

	item := listFindByField(t, territoriesPath(), url.Values{"q": {state}, "include": {territoryIncludes}}, "id", territoryID)
	require.NotNil(t, item, "the new territory should be listed")
	listed := parseJSON(item)
	assert.Equal(t, want, territoryStateOf(listed))
	assertExpandedTerritoryRefs(t, listed, SeedUser2ID, plName)
}

// Without ?include the rep and product line are null, never fabricated, while every scalar is still there.
func TestTerritories_ReferencesAreNullWithoutInclude(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-st")
	territoryID := jsonField(createTerritory(t, map[string]any{
		"state":           state,
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    SeedAccountUserID,
		"product_line_id": SeedProductLineID,
	}), "id")

	status, body, err := apiClient.GetListRaw(territoryPath(territoryID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, territoryState{state: state, start: "10001", end: "10999"}, territoryStateOf(got))
	assertNilField(t, got, "sales_rep")
	assertNilField(t, got, "product_line")
}

// A territory with no ZIP codes covers its whole state and most have no product line; all three read back as
// null rather than zero values the dashboard would render as ZIP code 0.
func TestTerritories_CreateWithOnlyStateAndSalesRepCoversTheWholeState(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-st")
	created := createTerritory(t, map[string]any{"state": state, "sales_rep_id": SeedAccountUserID})
	want := territoryState{state: state, salesRepID: SeedAccountUserID}
	assert.Equal(t, want, territoryStateOf(created))
	for _, key := range []string{"start_zipcode", "end_zipcode", "product_line"} {
		v, present := created[key]
		assert.True(t, present, "%s should be present as null", key)
		assert.Nil(t, v, "%s should be null", key)
	}
	assert.Equal(t, want, territoryStateOf(getTerritory(t, jsonField(created, "id"))))
}

// The form tells users to leave the end ZIP code empty to cover a single ZIP code, and the endpoint documents
// that an end ZIP code without a start is dropped — on its own it would match nothing.
func TestTerritories_CreateWithOneZipcode(t *testing.T) {
	t.Parallel()

	t.Run("start alone covers that one ZIP code", func(t *testing.T) {
		t.Parallel()
		state := uniqueName("e2e-terr-st")
		created := createTerritory(t, map[string]any{"state": state, "start_zipcode": 10501, "sales_rep_id": SeedAccountUserID})
		want := territoryState{state: state, start: "10501", salesRepID: SeedAccountUserID}
		assert.Equal(t, want, territoryStateOf(created))
		assert.Equal(t, want, territoryStateOf(getTerritory(t, jsonField(created, "id"))))
	})

	t.Run("end alone is dropped", func(t *testing.T) {
		t.Parallel()
		state := uniqueName("e2e-terr-st")
		created := createTerritory(t, map[string]any{"state": state, "end_zipcode": 10999, "sales_rep_id": SeedAccountUserID})
		want := territoryState{state: state, salesRepID: SeedAccountUserID}
		assert.Equal(t, want, territoryStateOf(created))
		assert.Equal(t, want, territoryStateOf(getTerritory(t, jsonField(created, "id"))))
	})
}

// Each malformed field is rejected before anything is written, naming the field so the form can mark it.
func TestTerritories_CreateRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		edit  func(body map[string]any)
		param string
	}{
		{"missing state", func(b map[string]any) { delete(b, "state") }, "state"},
		{"blank state", func(b map[string]any) { b["state"] = "" }, "state"},
		{"state over 255 characters", func(b map[string]any) { b["state"] = strings.Repeat("A", 256) }, "state"},
		{"missing sales rep", func(b map[string]any) { delete(b, "sales_rep_id") }, "sales_rep_id"},
		{"blank sales rep", func(b map[string]any) { b["sales_rep_id"] = "" }, "sales_rep_id"},
		{"start ZIP code below 501", func(b map[string]any) { b["start_zipcode"] = 500 }, "start_zipcode"},
		{"start ZIP code above 99999", func(b map[string]any) { b["start_zipcode"] = 100000 }, "start_zipcode"},
		{"end ZIP code below 501", func(b map[string]any) { delete(b, "start_zipcode"); b["end_zipcode"] = 500 }, "end_zipcode"},
		{"end ZIP code above 99999", func(b map[string]any) { b["end_zipcode"] = 100000 }, "end_zipcode"},
		{"null start ZIP code", func(b map[string]any) { b["start_zipcode"] = nil }, "start_zipcode"},
		{"null end ZIP code", func(b map[string]any) { b["end_zipcode"] = nil }, "end_zipcode"},
		{"blank product line", func(b map[string]any) { b["product_line_id"] = "" }, "product_line_id"},
		{"null product line", func(b map[string]any) { b["product_line_id"] = nil }, "product_line_id"},
		{"unknown field", func(b map[string]any) { b[bogusE2EJSONField] = 1 }, bogusE2EJSONField},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]any{
				"state":         uniqueName("e2e-terr-bad"),
				"start_zipcode": 10001,
				"end_zipcode":   10999,
				"sales_rep_id":  SeedAccountUserID,
			}
			tc.edit(body)
			status, resp, err := apiClient.Post(territoriesPath(), body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				apiClient.Delete(territoryPath(jsonField(parseJSON(resp), "id")))
			}
			assertTerritoryErrorParam(t, status, resp, tc.param)
		})
	}
}

// A ZIP code of the wrong JSON type is refused naming start_zipcode / end_zipcode. Both are
// field.Optional, whose type error is raised inside its own UnmarshalJSON, where the decoder attaches no
// field; the gateway names it from the body.
func TestTerritories_CreateRejectsAMistypedZipcodeNamingTheField(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		field string
		value any
	}{
		{"start ZIP code as a string", "start_zipcode", "10001"},
		{"start ZIP code as text", "start_zipcode", "abc"},
		{"start ZIP code with a fraction", "start_zipcode", 10001.5},
		{"end ZIP code as a string", "end_zipcode", "10999"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]any{
				"state":         uniqueName("e2e-terr-bad"),
				"start_zipcode": 10001,
				"end_zipcode":   10999,
				"sales_rep_id":  SeedAccountUserID,
			}
			body[tc.field] = tc.value
			status, resp, err := apiClient.Post(territoriesPath(), body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				apiClient.Delete(territoryPath(jsonField(parseJSON(resp), "id")))
			}
			assertTerritoryErrorParam(t, status, resp, tc.field)
		})
	}
}

// A range that ends before it starts could never match an order — sales rep lookup needs start <= zip <= end
// (FindSalesRepByZipcode) — so it is refused naming a ZIP code field.
func TestTerritories_CreateRejectsARangeThatEndsBeforeItStarts(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(territoriesPath(), map[string]any{
		"state":         uniqueName("e2e-terr-bad"),
		"start_zipcode": 20000,
		"end_zipcode":   10000,
		"sales_rep_id":  SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		apiClient.Delete(territoryPath(jsonField(parseJSON(body), "id")))
	}
	assertTerritoryErrorParam(t, status, body, "start_zipcode", "end_zipcode")
}

// sales_rep_id must be an account user of the caller's account and product_line_id one of its product lines.
// The territory table has no foreign keys, so anything else would be stored: another tenant's or a customer's
// user would be auto-assigned as the rep on this account's orders, and another tenant's product line name would
// match the list's search. Each is refused with 400 naming the field, as the sales-order endpoints do.
func TestTerritories_CreateRejectsReferencesTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		field string
		value string
	}{
		{"unknown sales rep", "sales_rep_id", mustGenID(t, id.AccountUserIDPrefix)},
		{"a user id instead of an account user id", "sales_rep_id", SeedUserID},
		{"another tenant's account user", "sales_rep_id", SeedTenantBAccountUserID},
		{"a customer account's user", "sales_rep_id", SeedCustomerAccountUserID},
		{"unknown product line", "product_line_id", mustGenID(t, id.ProductLineIDPrefix)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]any{"state": uniqueName("e2e-terr-ref"), "sales_rep_id": SeedAccountUserID}
			body[tc.field] = tc.value
			status, resp, err := apiClient.Post(territoriesPath(), body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				apiClient.Delete(territoryPath(jsonField(parseJSON(resp), "id")))
			}
			assertValidationParam(t, status, resp, tc.field)
		})
	}

	t.Run("another tenant's product line", func(t *testing.T) {
		t.Parallel()
		clientB := getTenantBClient()
		pathB := territoriesPathFor(SeedTenantBAccountID)
		status, resp, err := clientB.Post(pathB, map[string]any{
			"state":           uniqueName("e2e-terr-ref"),
			"sales_rep_id":    SeedTenantBAccountUserID,
			"product_line_id": SeedProductLineID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		if status == 201 {
			clientB.Delete(pathB + "/" + jsonField(parseJSON(resp), "id"))
		}
		assertValidationParam(t, status, resp, "product_line_id")
	})
}

// A retried create (the dashboard retries on a timeout) must hand back the same territory rather than
// assigning the area twice, and a replayed edit must not be applied again.
func TestTerritories_ReplaysWithTheSameIdempotencyKey(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-idem")
	body := map[string]any{"state": state, "start_zipcode": 10001, "end_zipcode": 10999, "sales_rep_id": SeedAccountUserID}
	key := newIdempotencyKey()

	status, first, err := apiClient.Post(territoriesPath(), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, first)
	territoryID := jsonField(parseJSON(first), "id")
	t.Cleanup(func() { apiClient.Delete(territoryPath(territoryID)) })

	status, second, err := apiClient.Post(territoriesPath(), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, second)
	assert.Equal(t, territoryID, jsonField(parseJSON(second), "id"))
	assert.Equal(t, []string{territoryID}, searchTerritoryIDs(t, apiClient, territoriesPath(), state),
		"a replayed create must not store a second territory")

	renamed := uniqueName("e2e-terr-idem")
	patchKey := newIdempotencyKey()
	status, patched, err := apiClient.Patch(territoryPath(territoryID), map[string]any{"state": renamed}, patchKey)
	require.NoError(t, err)
	requireStatus(t, 200, status, patched)

	latest := uniqueName("e2e-terr-idem")
	patchTerritory(t, territoryID, map[string]any{"state": latest})

	status, replayed, err := apiClient.Patch(territoryPath(territoryID), map[string]any{"state": renamed}, patchKey)
	require.NoError(t, err)
	requireStatus(t, 200, status, replayed)
	assert.Equal(t, renamed, jsonField(parseJSON(replayed), "state"), "a replay returns the original response")
	assert.Equal(t, jsonField(parseJSON(patched), "updated_at"), jsonField(parseJSON(replayed), "updated_at"))
	assert.Equal(t, latest, jsonField(getTerritory(t, territoryID), "state"), "a replay must not reapply the edit")
}

// ──────────────────────────────────────────────
// Update
// ──────────────────────────────────────────────

// The details page saves one field at a time; an edit to one must leave every other field as it was.
func TestTerritories_UpdateChangesOnlyTheFieldsSent(t *testing.T) {
	t.Parallel()
	pl1, _ := createTerritoryProductLine(t)
	pl2, _ := createTerritoryProductLine(t)
	want := territoryState{uniqueName("e2e-terr-upd"), "10001", "10999", SeedAccountUserID, pl1}
	created := createTerritory(t, map[string]any{
		"state":           want.state,
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    want.salesRepID,
		"product_line_id": pl1,
	})
	territoryID := jsonField(created, "id")

	steps := []struct {
		name  string
		body  map[string]any
		apply func(*territoryState)
	}{
		{"state", map[string]any{"state": "e2e-renamed"}, func(s *territoryState) { s.state = "e2e-renamed" }},
		{"start ZIP code", map[string]any{"start_zipcode": 10100}, func(s *territoryState) { s.start = "10100" }},
		{"end ZIP code", map[string]any{"end_zipcode": 10900}, func(s *territoryState) { s.end = "10900" }},
		{"sales rep", map[string]any{"sales_rep_id": SeedAccountUser2ID}, func(s *territoryState) { s.salesRepID = SeedAccountUser2ID }},
		{"product line", map[string]any{"product_line_id": pl2}, func(s *territoryState) { s.productLineID = pl2 }},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			step.apply(&want)
			assert.Equal(t, want, territoryStateOf(patchTerritory(t, territoryID, step.body)))
			got := getTerritory(t, territoryID)
			assert.Equal(t, want, territoryStateOf(got))
			assert.Equal(t, jsonField(created, "created_at"), jsonField(got, "created_at"))
		})
	}
}

// The dashboard edits ZIP codes with the clear_* flags: an emptied start ZIP code sends both clears, an
// emptied end ZIP code sends clear_end_zipcode. Clearing must never touch the state, rep, or product line.
func TestTerritories_UpdateClearsZipcodesTheWayTheDashboardSendsThem(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-zip")
	created := createTerritory(t, map[string]any{
		"state":           state,
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    SeedAccountUserID,
		"product_line_id": SeedProductLineID,
	})
	territoryID := jsonField(created, "id")
	at := func(start, end string) territoryState {
		return territoryState{state, start, end, SeedAccountUserID, SeedProductLineID}
	}

	steps := []struct {
		name string
		body map[string]any
		want territoryState
	}{
		{"emptying the start ZIP code covers the whole state",
			map[string]any{"clear_start_zipcode": true, "clear_end_zipcode": true}, at("", "")},
		{"a range can be set again", map[string]any{"start_zipcode": 20001, "end_zipcode": 20999}, at("20001", "20999")},
		{"emptying the end ZIP code narrows it to the start ZIP code",
			map[string]any{"start_zipcode": 20500, "clear_end_zipcode": true}, at("20500", "")},
		{"an end ZIP code widens a single ZIP code into a range", map[string]any{"end_zipcode": 20999}, at("20500", "20999")},
		{"clearing the start ZIP code alone also clears the end", map[string]any{"clear_start_zipcode": true}, at("", "")},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			assert.Equal(t, step.want, territoryStateOf(patchTerritory(t, territoryID, step.body)))
			assert.Equal(t, step.want, territoryStateOf(getTerritory(t, territoryID)))
		})
	}
}

// The dashboard sends clear_product_line when the product line is emptied; false is not a clear.
func TestTerritories_UpdateClearsTheProductLine(t *testing.T) {
	t.Parallel()
	plID, _ := createTerritoryProductLine(t)
	state := uniqueName("e2e-terr-pl")
	created := createTerritory(t, map[string]any{
		"state":           state,
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    SeedAccountUserID,
		"product_line_id": plID,
	})
	territoryID := jsonField(created, "id")
	with := territoryState{state, "10001", "10999", SeedAccountUserID, plID}
	without := territoryState{state, "10001", "10999", SeedAccountUserID, ""}

	steps := []struct {
		name string
		body map[string]any
		want territoryState
	}{
		{"clearing removes it", map[string]any{"clear_product_line": true}, without},
		{"it can be set again", map[string]any{"product_line_id": plID}, with},
		{"false leaves it", map[string]any{"clear_product_line": false}, with},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			got := patchTerritory(t, territoryID, step.body)
			assert.Equal(t, step.want, territoryStateOf(got))
			if step.want.productLineID == "" {
				assertNilField(t, got, "product_line")
			}
			assert.Equal(t, step.want, territoryStateOf(getTerritory(t, territoryID)))
		})
	}
}

// A request that both clears a field and sets it is contradictory, so it is a 400 naming the field and changes
// nothing, rather than letting either silently win — as the gateway rejects other ambiguous input (unknown
// fields, explicit null). The dashboard never sends both.
func TestTerritories_UpdateRejectsClearingAndSettingTheSameField(t *testing.T) {
	t.Parallel()
	plID, _ := createTerritoryProductLine(t)

	cases := []struct {
		name   string
		body   map[string]any
		params []string
	}{
		{"start ZIP code", map[string]any{"clear_start_zipcode": true, "start_zipcode": 20001}, []string{"start_zipcode", "clear_start_zipcode"}},
		{"end ZIP code", map[string]any{"clear_end_zipcode": true, "end_zipcode": 20999}, []string{"end_zipcode", "clear_end_zipcode"}},
		{"product line", map[string]any{"clear_product_line": true, "product_line_id": plID}, []string{"product_line_id", "clear_product_line"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			created := createTerritory(t, map[string]any{
				"state":           uniqueName("e2e-terr-both"),
				"start_zipcode":   10001,
				"end_zipcode":     10999,
				"sales_rep_id":    SeedAccountUserID,
				"product_line_id": SeedProductLineID,
			})
			territoryID := jsonField(created, "id")

			status, body, err := apiClient.Patch(territoryPath(territoryID), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			assertTerritoryErrorParam(t, status, body, tc.params...)
			assert.Equal(t, territoryStateOf(created), territoryStateOf(getTerritory(t, territoryID)),
				"a rejected edit must change nothing")
		})
	}
}

// An end ZIP code on a territory without a start ZIP code would be a half range that matches no ZIP code yet
// shows an end ZIP code, so an update never stores one. Either a 400 naming end_zipcode or dropping it as
// create does is acceptable.
func TestTerritories_UpdateNeverStoresAnEndZipcodeWithoutAStart(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-half")
	created := createTerritory(t, map[string]any{"state": state, "sales_rep_id": SeedAccountUserID})
	territoryID := jsonField(created, "id")

	status, body, err := apiClient.Patch(territoryPath(territoryID), map[string]any{"end_zipcode": 10999}, newIdempotencyKey())
	require.NoError(t, err)
	if status != 200 {
		assertTerritoryErrorParam(t, status, body, "end_zipcode")
	}
	assert.Equal(t, territoryState{state: state, salesRepID: SeedAccountUserID}, territoryStateOf(getTerritory(t, territoryID)),
		"a territory without a start ZIP code must not hold an end ZIP code")
}

// An edit that leaves the range ending before it starts is refused, checked against the stored other end, naming
// the field that makes it invalid; see TestTerritories_CreateRejectsARangeThatEndsBeforeItStarts.
func TestTerritories_UpdateRejectsARangeThatEndsBeforeItStarts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		body   map[string]any
		params []string
	}{
		{"start moved past the end", map[string]any{"start_zipcode": 11000}, []string{"start_zipcode"}},
		{"end moved before the start", map[string]any{"end_zipcode": 10000}, []string{"end_zipcode"}},
		{"both sent reversed", map[string]any{"start_zipcode": 12000, "end_zipcode": 11000}, []string{"start_zipcode", "end_zipcode"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			created := createTerritory(t, map[string]any{
				"state":         uniqueName("e2e-terr-rev"),
				"start_zipcode": 10001,
				"end_zipcode":   10999,
				"sales_rep_id":  SeedAccountUserID,
			})
			territoryID := jsonField(created, "id")

			status, body, err := apiClient.Patch(territoryPath(territoryID), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			assertTerritoryErrorParam(t, status, body, tc.params...)
			assert.Equal(t, territoryStateOf(created), territoryStateOf(getTerritory(t, territoryID)),
				"a rejected edit must change nothing")
		})
	}
}

// Malformed edits are rejected naming the field and leave the territory as it was. A territory always has a
// rep, so sales_rep_id can be replaced but never emptied.
func TestTerritories_UpdateRejectsInvalidFields(t *testing.T) {
	t.Parallel()
	created := createTerritory(t, map[string]any{
		"state":           uniqueName("e2e-terr-bad"),
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    SeedAccountUserID,
		"product_line_id": SeedProductLineID,
	})
	territoryID := jsonField(created, "id")
	before := territoryStateOf(created)

	cases := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"blank state", map[string]any{"state": ""}, "state"},
		{"state over 255 characters", map[string]any{"state": strings.Repeat("A", 256)}, "state"},
		{"null state", map[string]any{"state": nil}, "state"},
		{"blank sales rep", map[string]any{"sales_rep_id": ""}, "sales_rep_id"},
		{"null sales rep", map[string]any{"sales_rep_id": nil}, "sales_rep_id"},
		{"start ZIP code below 501", map[string]any{"start_zipcode": 500}, "start_zipcode"},
		{"end ZIP code above 99999", map[string]any{"end_zipcode": 100000}, "end_zipcode"},
		{"null start ZIP code", map[string]any{"start_zipcode": nil}, "start_zipcode"},
		{"blank product line", map[string]any{"product_line_id": ""}, "product_line_id"},
		{"null product line", map[string]any{"product_line_id": nil}, "product_line_id"},
		{"null clear flag", map[string]any{"clear_product_line": nil}, "clear_product_line"},
		{"unknown field", map[string]any{bogusE2EJSONField: 1}, bogusE2EJSONField},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Patch(territoryPath(territoryID), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			assertTerritoryErrorParam(t, status, body, tc.param)
		})
	}

	t.Run("empty body", func(t *testing.T) {
		status, body, err := apiClient.Patch(territoryPath(territoryID), map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	})

	assert.Equal(t, before, territoryStateOf(getTerritory(t, territoryID)), "rejected edits must change nothing")
}

// An edit's sales_rep_id and product_line_id are held to the same account as a create's, and an unknown rep is a
// 400 naming the field rather than a 404 that reads as if the territory itself were missing. See
// TestTerritories_CreateRejectsReferencesTheAccountDoesNotHave.
func TestTerritories_UpdateRejectsReferencesTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		field string
		value string
	}{
		{"unknown sales rep", "sales_rep_id", mustGenID(t, id.AccountUserIDPrefix)},
		{"another tenant's account user", "sales_rep_id", SeedTenantBAccountUserID},
		{"a customer account's user", "sales_rep_id", SeedCustomerAccountUserID},
		{"unknown product line", "product_line_id", mustGenID(t, id.ProductLineIDPrefix)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			created := createTerritory(t, map[string]any{
				"state":           uniqueName("e2e-terr-ref"),
				"sales_rep_id":    SeedAccountUserID,
				"product_line_id": SeedProductLineID,
			})
			territoryID := jsonField(created, "id")

			status, body, err := apiClient.Patch(territoryPath(territoryID), map[string]any{tc.field: tc.value}, newIdempotencyKey())
			require.NoError(t, err)
			assertValidationParam(t, status, body, tc.field)
			assert.Equal(t, territoryStateOf(created), territoryStateOf(getTerritory(t, territoryID)),
				"a rejected edit must change nothing")
		})
	}
}

func TestTerritories_UpdateNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(territoryPath(mustGenID(t, id.TerritoryIDPrefix)), map[string]any{"state": "NY"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// List / Search / Pagination
// ──────────────────────────────────────────────

// The territories page searches by what users see on a row: the state, the rep's name or email, and the
// product line, case-insensitively and by substring.
func TestTerritories_SearchMatchesStateSalesRepAndProductLine(t *testing.T) {
	t.Parallel()
	plID, plName := createTerritoryProductLine(t)
	stateA := uniqueName("e2e-terr-srch")
	stateB := uniqueName("e2e-terr-srch")
	byState := jsonField(createTerritory(t, map[string]any{"state": stateA, "sales_rep_id": SeedAccountUserID}), "id")
	byRepAndLine := jsonField(createTerritory(t, map[string]any{
		"state":           stateB,
		"sales_rep_id":    SeedAccountUser2ID,
		"product_line_id": plID,
	}), "id")
	search := func(q string) []string { return searchTerritoryIDs(t, apiClient, territoriesPath(), q) }

	assert.Equal(t, []string{byState}, search(stateA), "state")
	assert.Equal(t, []string{byState}, search(stateA[len("e2e-terr-"):]), "part of the state")
	assert.Equal(t, []string{byRepAndLine}, search(plName), "product line name")
	assert.Equal(t, []string{byRepAndLine}, search(strings.ToUpper(plName)), "product line name in another case")

	for _, q := range []string{"Sarah Martinez", seedUser2Email} {
		found := search(q)
		assert.Contains(t, found, byRepAndLine, "searching %q should find the territory whose rep it names", q)
		assert.NotContains(t, found, byState, "searching %q should not find another rep's territory", q)
	}
}

func TestTerritories_SearchWithNoMatchReturnsAnEmptyList(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(territoriesPath(), url.Values{"q": {uniqueName("e2e-terr-none")}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
	assert.False(t, list.PageInfo.HasNextPage)
	assert.False(t, list.PageInfo.HasPrevPage)
}

// Typing a ZIP code lists the territories covering it, as the dashboard's Express endpoint did (its search OR-ed
// a ZIP code match with the text match), even though the ZIP code appears in no state, rep, or product line name.
func TestTerritories_SearchByZipcodeFindsTheTerritoriesCoveringIt(t *testing.T) {
	t.Parallel()
	create := func(body map[string]any) string {
		body["state"] = uniqueName("e2e-terr-zipq")
		body["sales_rep_id"] = SeedAccountUserID
		return jsonField(createTerritory(t, body), "id")
	}
	covering := create(map[string]any{"start_zipcode": 36000, "end_zipcode": 36999})
	exactly := create(map[string]any{"start_zipcode": 36500})
	elsewhere := create(map[string]any{"start_zipcode": 37000, "end_zipcode": 37999})
	wholeState := create(map[string]any{})

	found := searchTerritoryIDs(t, apiClient, territoriesPath(), "36500")
	assert.Contains(t, found, covering, "a range containing the ZIP code should match")
	assert.Contains(t, found, exactly, "a single-ZIP territory for that ZIP code should match")
	assert.NotContains(t, found, elsewhere, "a range not containing the ZIP code should not match")
	assert.NotContains(t, found, wholeState, "a ZIP code search should not match state-wide territories")
}

// The territories page pages by cursor both ways (Next, then Back), newest first. A page boundary must
// neither repeat nor skip a territory, and the search has to carry across pages.
func TestTerritories_ListWalksPagesForwardAndBackNewestFirst(t *testing.T) {
	t.Parallel()
	plID, plName := createTerritoryProductLine(t)
	var created []string
	for range 5 {
		created = append(created, jsonField(createTerritory(t, map[string]any{
			"state":           uniqueName("e2e-terr-page"),
			"sales_rep_id":    SeedAccountUserID,
			"product_line_id": plID,
		}), "id"))
	}

	full, _, err := apiClient.GetList(territoriesPath(), url.Values{"q": {plName}, "limit": {"100"}})
	require.NoError(t, err)
	order := territoryIDsOf(full)
	require.ElementsMatch(t, created, order)
	var previous time.Time
	for i, item := range full.Data {
		createdAt, err := time.Parse(time.RFC3339Nano, DataItemField(item, "created_at"))
		require.NoError(t, err)
		if i > 0 {
			assert.False(t, createdAt.After(previous), "territories should be listed newest first")
		}
		previous = createdAt
	}

	page, _, err := apiClient.GetList(territoriesPath(), url.Values{"q": {plName}, "limit": {"2"}})
	require.NoError(t, err)
	assert.False(t, page.PageInfo.HasPrevPage)
	assert.Nil(t, page.PageInfo.PreviousPageURL)
	pages := [][]string{territoryIDsOf(page)}
	for page.PageInfo.HasNextPage && len(pages) <= len(created) {
		require.NotNil(t, page.PageInfo.NextPageURL)
		page, _, err = apiClient.GetListFromPageURL(page.PageInfo.NextPageURL)
		require.NoError(t, err)
		pages = append(pages, territoryIDsOf(page))
	}
	require.Equal(t, [][]string{order[0:2], order[2:4], order[4:5]}, pages, "forward walk")
	assert.Nil(t, page.PageInfo.NextPageURL)
	assert.True(t, page.PageInfo.HasPrevPage)

	require.NotNil(t, page.PageInfo.PreviousPageURL)
	page, _, err = apiClient.GetListFromPageURL(page.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	assert.Equal(t, order[2:4], territoryIDsOf(page), "back from the last page")
	assert.True(t, page.PageInfo.HasNextPage)
	assert.True(t, page.PageInfo.HasPrevPage)

	require.NotNil(t, page.PageInfo.PreviousPageURL)
	page, _, err = apiClient.GetListFromPageURL(page.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	assert.Equal(t, order[0:2], territoryIDsOf(page), "back to the first page")
	assert.True(t, page.PageInfo.HasNextPage)
	assert.False(t, page.PageInfo.HasPrevPage, "nothing comes before the first page")
}

func TestTerritories_ListRejectsAMalformedCursor(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(territoriesPath(), url.Values{"cursor": {"not-a-cursor"}})
	require.NoError(t, err)
	assertTerritoryErrorParam(t, status, body, "cursor")
}

// ──────────────────────────────────────────────
// Delete
// ──────────────────────────────────────────────

// A deleted territory can't be read, edited, or found, and deleting it again says it is already gone, as the
// endpoint documents.
func TestTerritories_DeleteRemovesTheTerritory(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-del")
	territoryID := jsonField(createTerritory(t, map[string]any{"state": state, "sales_rep_id": SeedAccountUserID}), "id")

	status, body, err := apiClient.Delete(territoryPath(territoryID))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(territoryPath(territoryID), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")

	status, body, err = apiClient.Patch(territoryPath(territoryID), map[string]any{"state": "NY"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	assert.Empty(t, searchTerritoryIDs(t, apiClient, territoriesPath(), state), "a deleted territory should not be listed")

	status, body, err = apiClient.Delete(territoryPath(territoryID))
	require.NoError(t, err)
	requireStatus(t, 410, status, body)
	requireErrorResponse(t, body, "resource_gone", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Tenant isolation / auth
// ──────────────────────────────────────────────

// Territories decide who is paid commission, so another tenant must not see, edit, or delete them, and gets
// 404 rather than 403 so it cannot learn the id exists — whichever account its request path names.
func TestTerritories_AnotherTenantCannotReadChangeOrDeleteThem(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-iso")
	created := createTerritory(t, map[string]any{
		"state":           state,
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    SeedAccountUserID,
		"product_line_id": SeedProductLineID,
	})
	territoryID := jsonField(created, "id")
	clientB := getTenantBClient()

	for _, base := range []string{territoriesPathFor(SeedTenantBAccountID), territoriesPath()} {
		path := base + "/" + territoryID

		status, body, err := clientB.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)

		status, body, err = clientB.Patch(path, map[string]any{"state": "e2e-hijacked"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)

		status, body, err = clientB.Delete(path)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	}

	assert.Empty(t, searchTerritoryIDs(t, clientB, territoriesPathFor(SeedTenantBAccountID), state),
		"another tenant's list must not include the territory")
	assert.Equal(t, territoryStateOf(created), territoryStateOf(getTerritory(t, territoryID)),
		"the territory must be untouched")
}

// KNOWN BUG (low severity): once a territory is deleted, another tenant deleting the same id gets 410
// "already been deleted" instead of 404, confirming the id existed. DeleteTerritory falls back to
// DeletedRecordRepo.Exists(resourceType, id) (services/core-service/internal/service/territory_service.go
// DeleteTerritory), and deleted_record has no account column, so the check spans every tenant.
func TestTerritories_AnotherTenantCannotTellADeletedTerritoryExisted(t *testing.T) {
	t.Parallel()
	territoryID := jsonField(createTerritory(t, map[string]any{"state": uniqueName("e2e-terr-iso"), "sales_rep_id": SeedAccountUserID}), "id")

	status, body, err := apiClient.Delete(territoryPath(territoryID))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = getTenantBClient().Delete(territoriesPathFor(SeedTenantBAccountID) + "/" + territoryID)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// The {account_id} path parameter names the account that owns the territory, so a path naming any other account
// finds nothing even when the OpenMRP-Account header names the owner. The dashboard sends its own account in both.
func TestTerritories_PathNamingAnotherAccountFindsNothing(t *testing.T) {
	t.Parallel()
	created := createTerritory(t, map[string]any{"state": uniqueName("e2e-terr-path"), "sales_rep_id": SeedAccountUserID})
	territoryID := jsonField(created, "id")
	elsewhere := territoriesPathFor(SeedTenantBAccountID) + "/" + territoryID

	status, body, err := apiClient.GetListRaw(elsewhere, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "GET under another account's path: %s", body)

	status, body, err = apiClient.Patch(elsewhere, map[string]any{"state": "e2e-misrouted"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "PATCH under another account's path: %s", body)

	status, body, err = apiClient.Delete(elsewhere)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "DELETE under another account's path: %s", body)

	status, body, err = apiClient.GetListRaw(territoryPath(territoryID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, jsonField(created, "state"), jsonField(parseJSON(body), "state"))
}

func TestTerritories_RequireAuthentication(t *testing.T) {
	t.Parallel()
	territoryID := jsonField(createTerritory(t, map[string]any{"state": uniqueName("e2e-terr-auth"), "sales_rep_id": SeedAccountUserID}), "id")
	unauth := apiClient.WithBearerToken("", SeedAccountID)

	status, body, err := unauth.GetListRaw(territoriesPath(), nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")

	status, body, err = unauth.GetListRaw(territoryPath(territoryID), nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)

	status, body, err = unauth.Post(territoriesPath(), map[string]any{"state": "NY", "sales_rep_id": SeedAccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)

	status, body, err = unauth.Patch(territoryPath(territoryID), map[string]any{"state": "NY"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)

	status, body, err = unauth.Delete(territoryPath(territoryID))
	require.NoError(t, err)
	requireStatus(t, 401, status, body)

	getTerritory(t, territoryID)
}

// The dashboard calls these endpoints as the signed-in user rather than with an API key: list with search,
// create, read, save with the clear flags, and delete, each expanded as the pages ask.
func TestTerritories_DashboardSessionManagesTerritories(t *testing.T) {
	t.Parallel()
	session := loginAsSeedUser(t)
	plID, plName := createTerritoryProductLine(t)
	state := uniqueName("e2e-terr-sess")

	status, body, err := session.Post(withTerritoryIncludes(territoriesPath()), map[string]any{
		"state":           state,
		"start_zipcode":   10001,
		"end_zipcode":     10999,
		"sales_rep_id":    SeedAccountUser2ID,
		"product_line_id": plID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	territoryID := jsonField(created, "id")
	t.Cleanup(func() { apiClient.Delete(territoryPath(territoryID)) })
	assertExpandedTerritoryRefs(t, created, SeedUser2ID, plName)

	list, _, err := session.GetList(territoriesPath(), url.Values{"q": {plName}, "limit": {"10"}, "include": {territoryIncludes}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	listed := parseJSON(list.Data[0])
	assert.Equal(t, territoryID, jsonField(listed, "id"))
	assertExpandedTerritoryRefs(t, listed, SeedUser2ID, plName)

	status, body, err = session.GetListRaw(territoryPath(territoryID), url.Values{"include": {territoryIncludes}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, territoryStateOf(created), territoryStateOf(parseJSON(body)))

	// Saving the details form with the ZIP codes and product line emptied.
	status, body, err = session.Patch(territoryPath(territoryID), map[string]any{
		"state":               state,
		"sales_rep_id":        SeedAccountUserID,
		"clear_start_zipcode": true,
		"clear_end_zipcode":   true,
		"clear_product_line":  true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, territoryState{state: state, salesRepID: SeedAccountUserID}, territoryStateOf(getTerritory(t, territoryID)))

	status, body, err = session.Delete(territoryPath(territoryID))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = apiClient.GetListRaw(territoryPath(territoryID), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// Audit events
// ──────────────────────────────────────────────

// The territory page links to its audit timeline, so every create, edit, and delete has to land there with
// what changed.
func TestTerritories_WritesAreAudited(t *testing.T) {
	t.Parallel()
	state := uniqueName("e2e-terr-audit")
	territoryID := jsonField(createTerritory(t, map[string]any{
		"state":         state,
		"start_zipcode": 10001,
		"end_zipcode":   10999,
		"sales_rep_id":  SeedAccountUserID,
	}), "id")

	createChanges := jsonListData(expectAuditEventWithChanges(t, territoryID, "territory", "create"), "changes")
	change, ok := changeForField(createChanges, "state")
	require.True(t, ok, "create event should record the state: %v", createChanges)
	assert.Equal(t, state, jsonField(change, "new_value"))
	change, ok = changeForField(createChanges, "start_zipcode")
	require.True(t, ok, "create event should record the start ZIP code: %v", createChanges)
	assert.Equal(t, "10001", jsonField(change, "new_value"))

	renamed := uniqueName("e2e-terr-audit")
	patchTerritory(t, territoryID, map[string]any{"state": renamed, "clear_end_zipcode": true})
	updateChanges := jsonListData(expectAuditEventWithChanges(t, territoryID, "territory", "update"), "changes")
	change, ok = changeForField(updateChanges, "state")
	require.True(t, ok, "update event should record the state: %v", updateChanges)
	assert.Equal(t, state, jsonField(change, "old_value"))
	assert.Equal(t, renamed, jsonField(change, "new_value"))
	change, ok = changeForField(updateChanges, "end_zipcode")
	require.True(t, ok, "update event should record the cleared end ZIP code: %v", updateChanges)
	assert.Equal(t, "10999", jsonField(change, "old_value"))
	assert.Nil(t, change["new_value"])

	status, body, err := apiClient.Delete(territoryPath(territoryID))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	deleteChanges := jsonListData(expectAuditEventWithChanges(t, territoryID, "territory", "delete"), "changes")
	change, ok = changeForField(deleteChanges, "state")
	require.True(t, ok, "delete event should record the state: %v", deleteChanges)
	assert.Equal(t, renamed, jsonField(change, "old_value"))
}

// The audit trail records a territory's sales rep and product line: the create event names both, and an edit that
// only reassigns the rep or clears the product line still produces an update event for the timeline.
func TestTerritories_AuditTrailRecordsTheSalesRepAndProductLine(t *testing.T) {
	t.Parallel()
	territoryID := jsonField(createTerritory(t, map[string]any{
		"state":           uniqueName("e2e-terr-audit"),
		"sales_rep_id":    SeedAccountUserID,
		"product_line_id": SeedProductLineID,
	}), "id")

	createChanges := jsonListData(expectAuditEventWithChanges(t, territoryID, "territory", "create"), "changes")
	for _, field := range []string{"sales_rep_id", "product_line_id"} {
		change, ok := changeForField(createChanges, field)
		if assert.True(t, ok, "create event should record %s: %v", field, createChanges) {
			assert.NotNil(t, change["new_value"], "create event should record which %s was set", field)
		}
	}

	patchTerritory(t, territoryID, map[string]any{"sales_rep_id": SeedAccountUser2ID, "clear_product_line": true})
	updateChanges := jsonListData(expectAuditEventWithChanges(t, territoryID, "territory", "update"), "changes")
	for _, field := range []string{"sales_rep_id", "product_line_id"} {
		_, ok := changeForField(updateChanges, field)
		assert.True(t, ok, "update event should record the %s change: %v", field, updateChanges)
	}
}

// ──────────────────────────────────────────────
// Expandable Fields (seed territory)
// ──────────────────────────────────────────────

func TestTerritories_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(territoriesPath()+"/"+SeedTerritoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["sales_rep"], "sales_rep should be null without ?include=sales_rep")
	assert.Nil(t, got["product_line"], "product_line should be null without ?include=product_line")

	list, _, err := apiClient.GetList(territoriesPath(), nil)
	require.NoError(t, err)
	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["sales_rep"], "sales_rep should be null on list items without ?include=sales_rep")
		assert.Nil(t, m["product_line"], "product_line should be null on list items without ?include=product_line")
	}
}

func TestTerritories_IncludeSalesRep(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(territoriesPath()+"/"+SeedTerritoryID, url.Values{"include": {"sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["sales_rep"]
	assert.True(t, ok, "sales_rep key should be present with ?include=sales_rep")
	if sr := jsonObject(got, "sales_rep"); sr != nil {
		assert.Equal(t, "account_user", jsonField(sr, "object"))
	}
}

func TestTerritories_IncludeProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(territoriesPath()+"/"+SeedTerritoryID, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pl := jsonObject(got, "product_line")
	require.NotNil(t, pl, "product_line should be present with ?include=product_line")
	assert.Equal(t, "product_line", jsonField(pl, "object"))
}
