//go:build e2e

package api_test

import (
	"fmt"
	"math/rand/v2"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sysPropertiesPath = "/v1/settings/properties"

// The counters the dashboard reads a next number from, to prefill a new customer, supplier or
// purchase order.
var dashboardNumberCounters = []string{"customer_number", "supplier_number", "purchase_order_number"}

// listSysProperties pages through the account's counters as client and returns every one, in the
// order the API returns them.
func listSysProperties(t *testing.T, client *Client, params url.Values) []map[string]any {
	t.Helper()
	merged := url.Values{"limit": {"100"}}
	for k, v := range params {
		merged[k] = v
	}

	list, _, err := client.GetList(sysPropertiesPath, merged)
	require.NoError(t, err, "listing %s", sysPropertiesPath)
	var out []map[string]any
	for page := 0; page < maxListScanPages; page++ {
		for _, raw := range list.Data {
			out = append(out, parseJSON(raw))
		}
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil {
			return out
		}
		list, _, err = client.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err, "paging %s", sysPropertiesPath)
	}
	t.Fatalf("%s did not end within %d pages", sysPropertiesPath, maxListScanPages)
	return nil
}

func sysPropertyCode(p map[string]any) string {
	return jsonField(jsonObject(p, "type"), "code")
}

func sysPropertyIDs(props []map[string]any) []string {
	ids := make([]string, 0, len(props))
	for _, p := range props {
		ids = append(ids, jsonField(p, "id"))
	}
	return ids
}

// sysPropertyByCode returns the client's account's counter of the given type.
func sysPropertyByCode(t *testing.T, client *Client, code string) map[string]any {
	t.Helper()
	for _, p := range listSysProperties(t, client, nil) {
		if sysPropertyCode(p) == code {
			return p
		}
	}
	t.Fatalf("the account has no %s counter", code)
	return nil
}

func intField(t *testing.T, m map[string]any, key string) int {
	t.Helper()
	v, ok := m[key].(float64)
	require.True(t, ok, "%s should be a JSON number: %v", key, m)
	return int(v)
}

// sysPropertyValue re-reads a counter, so an assertion covers what was stored rather than an echo.
func sysPropertyValue(t *testing.T, client *Client, id string) int {
	t.Helper()
	status, body, err := client.GetListRaw(sysPropertiesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return intField(t, parseJSON(body), "value")
}

func setSysPropertyValue(t *testing.T, client *Client, id string, value int) map[string]any {
	t.Helper()
	status, body, err := client.Patch(sysPropertiesPath+"/"+id, map[string]any{"value": value}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// restoreSysPropertyOnCleanup puts a counter back to the value it holds now once the test ends.
// Counters are shared by every test on the account, so a test that moves one must not leave it moved.
func restoreSysPropertyOnCleanup(t *testing.T, client *Client, id string) {
	t.Helper()
	original := sysPropertyValue(t, client, id)
	t.Cleanup(func() {
		status, body, err := client.Patch(sysPropertiesPath+"/"+id, map[string]any{"value": original}, newIdempotencyKey())
		if assert.NoError(t, err) {
			assert.Equal(t, 200, status, "restoring counter %s to %d: %s", id, original, string(body))
		}
	})
}

func latestValuePath(code string) string {
	return sysPropertiesPath + "/" + code + "/latest-value"
}

// latestSysPropertyValue reads a counter's next number the way the dashboard's create forms do.
func latestSysPropertyValue(t *testing.T, client *Client, code string) int {
	t.Helper()
	status, body, err := client.GetListRaw(latestValuePath(code), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "sys_property_value", jsonField(got, "object"))
	n, err := strconv.Atoi(jsonField(got, "value"))
	require.NoError(t, err, "value is a whole number in a string: %s", string(body))
	return n
}

// numberInUse reports whether an existing record already carries number in the series a counter
// feeds. It reads the database so the answer does not lean on the code under test.
func numberInUse(t *testing.T, accountID, code string, number int) bool {
	t.Helper()
	n := strconv.Itoa(number)
	switch code {
	case "customer_number", "supplier_number":
		return countInAccount(t, `SELECT COUNT(*) FROM account_relation
			WHERE external_number = ? AND owner_account_id = ? AND account_relation_role_code = ?`,
			n, accountID, strings.TrimSuffix(code, "_number")) > 0
	case "purchase_order_number":
		return countInAccount(t, `SELECT COUNT(*) FROM sales_order
			WHERE number = ? AND owner_account_id = ? AND buyer_account_id = ? AND sales_order_type_code = 'purchase_order'`,
			n, accountID, accountID) > 0
	}
	t.Fatalf("no in-use check for counter %s", code)
	return false
}

// unusedSupplierNumbers picks a run of span supplier numbers far above any the account has handed
// out, so a test can occupy them and point the counter at them without meeting another test's suppliers.
func unusedSupplierNumbers(t *testing.T, span int) int {
	t.Helper()
	base := 1_500_000_000 + rand.IntN(500_000_000)
	for n := base; n < base+span; n++ {
		require.False(t, numberInUse(t, SeedAccountID, "supplier_number", n), "precondition: supplier number %d is free", n)
	}
	return base
}

func createSupplierNumbered(t *testing.T, number int) {
	t.Helper()
	createAndCleanup(t, suppliersPath, map[string]any{
		"name":   uniqueName("e2e-sysprop-supplier"),
		"number": strconv.Itoa(number),
	})
}

// --- List ---

// The settings page renders every counter by its type, so each must carry its type in full, and an
// account holds at most one counter per type.
func TestSysProperties_ListShowsEachCounterWithItsType(t *testing.T) {
	t.Parallel()

	props := listSysProperties(t, apiClient, nil)
	require.NotEmpty(t, props)

	byCode := map[string]string{}
	for _, p := range props {
		id := jsonField(p, "id")
		assertIDFormat(t, id, "sypp")
		assertObjectField(t, p, "sys_property")
		typ := jsonObject(p, "type")
		require.NotNil(t, typ, "a counter always carries its type: %v", p)
		assertIDFormat(t, jsonField(typ, "id"), "sypptp")
		assert.Equal(t, "sys_property_type", jsonField(typ, "object"))
		assert.NotEmpty(t, jsonField(typ, "name"), "the type name is what the settings page shows: %v", p)
		code := jsonField(typ, "code")
		require.NotEmpty(t, code, "the type code is what the dashboard keys counters by: %v", p)
		intField(t, p, "value")
		assertValidTimestamp(t, jsonField(p, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(p, "updated_at"), "updated_at")

		_, dup := byCode[code]
		assert.False(t, dup, "an account holds one %s counter, got a second (%s)", code, id)
		byCode[code] = id
	}

	assert.Equal(t, SeedSysPropertyID, byCode["transaction_number"])
	for _, code := range dashboardNumberCounters {
		assert.Contains(t, byCode, code, "the dashboard prefills from the %s counter", code)
	}
}

// SysPropertyType.code is published as an enum, so the list must not answer with a code outside it.
// production_schedule_version, the counter schedule generation keeps, is internal and not listed. The
// documented codes are read from the spec so the test follows the enum.
func TestSysProperties_ListedTypeCodesAreDocumented(t *testing.T) {
	t.Parallel()

	spec, err := LoadFullSpec()
	require.NoError(t, err)
	require.NotNil(t, spec.Components)
	documented := map[string]bool{}
	for _, v := range spec.Components.Schemas["SysPropertyType"].Properties["code"].Enum {
		documented[fmt.Sprint(v)] = true
	}
	require.NotEmpty(t, documented, "the spec enumerates SysPropertyType.code")

	props := listSysProperties(t, apiClient, nil)
	hasScheduleCounter := false
	for _, p := range props {
		hasScheduleCounter = hasScheduleCounter || sysPropertyCode(p) == "production_schedule_version"
	}
	if !hasScheduleCounter {
		// The schedule counter exists only once the account has generated a schedule. Generate one so
		// the check cannot pass merely because this test ran before the schedule tests.
		created := generateSchedule(t, map[string]any{"name": uniqueName("e2e-sysprop-schedule"), "horizon_weeks": 4})
		cleanupSchedule(t, jsonField(created, "id"))
		props = listSysProperties(t, apiClient, nil)
	}

	for _, p := range props {
		code := sysPropertyCode(p)
		assert.True(t, documented[code], "counter %s has type code %q, which SysPropertyType.code does not document", jsonField(p, "id"), code)
	}
}

// The settings page searches counters by what it shows, the type name, ignoring case.
func TestSysProperties_SearchMatchesTheTypeName(t *testing.T) {
	t.Parallel()

	customer := listSysProperties(t, apiClient, url.Values{"q": {"customer"}})
	require.Len(t, customer, 1, "only the customer number counter is named for customers: %v", customer)
	assert.Equal(t, "customer_number", sysPropertyCode(customer[0]))

	numbered := listSysProperties(t, apiClient, url.Values{"q": {"NUMBER"}})
	var codes []string
	for _, p := range numbered {
		codes = append(codes, sysPropertyCode(p))
		assert.Contains(t, strings.ToLower(jsonField(jsonObject(p, "type"), "name")), "number")
	}
	for _, code := range append([]string{"transaction_number"}, dashboardNumberCounters...) {
		assert.Contains(t, codes, code)
	}
	assert.NotContains(t, codes, "sscc_count", "Sscc Count is not named a number")
}

func TestSysProperties_SearchWithNoMatchIsEmpty(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(sysPropertiesPath, url.Values{"q": {uniqueName("e2e-no-such-counter")}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assertEmptyListData(t, list.Data)
	assert.False(t, list.PageInfo.HasNextPage)

	// LIKE wildcards match literally; otherwise "%" would list every counter.
	for _, q := range []string{"%", "_"} {
		list, _, err := apiClient.GetList(sysPropertiesPath, url.Values{"q": {q}})
		require.NoError(t, err)
		assertEmptyListData(t, list.Data, "q="+q+" matches no counter name")
	}
}

// The dashboard reads every counter by walking cursors; a page that repeats or drops one would show a
// counter twice or hide it.
func TestSysProperties_CursorPaginationVisitsEveryCounterOnce(t *testing.T) {
	t.Parallel()

	want := sysPropertyIDs(listSysProperties(t, apiClient, nil))
	require.GreaterOrEqual(t, len(want), 3, "pagination needs more counters than one page holds")

	page1, _, err := apiClient.GetList(sysPropertiesPath, url.Values{"limit": {"2"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 2)
	assert.False(t, page1.PageInfo.HasPrevPage, "the first page has nothing before it")
	require.True(t, page1.PageInfo.HasNextPage)

	var walked []string
	list := page1
	for page := 0; ; page++ {
		require.Less(t, page, len(want), "the walk must end")
		require.LessOrEqual(t, len(list.Data), 2, "limit=2 pages hold at most two counters")
		for _, raw := range list.Data {
			walked = append(walked, DataItemField(raw, "id"))
		}
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil {
			break
		}
		list, _, err = apiClient.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
	}
	assert.Equal(t, want, walked, "a two-at-a-time walk visits the same counters in the same order as one page")

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.True(t, page2.PageInfo.HasPrevPage)
	back, _, err := apiClient.GetListFromPageURL(page2.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	require.Len(t, back.Data, 2)
	assert.Equal(t,
		[]string{DataItemField(page1.Data[0], "id"), DataItemField(page1.Data[1], "id")},
		[]string{DataItemField(back.Data[0], "id"), DataItemField(back.Data[1], "id")},
		"stepping back from page two returns page one")
}

// --- Retrieve ---

func TestSysProperties_RetrieveReturnsTheCounter(t *testing.T) {
	t.Parallel()

	got := parseJSON(mustGet(t, sysPropertiesPath+"/"+SeedSysPropertyID))
	assert.Equal(t, SeedSysPropertyID, jsonField(got, "id"))
	assertObjectField(t, got, "sys_property")
	typ := jsonObject(got, "type")
	require.NotNil(t, typ)
	assertIDFormat(t, jsonField(typ, "id"), "sypptp")
	assert.Equal(t, "sys_property_type", jsonField(typ, "object"))
	assert.Equal(t, "transaction_number", jsonField(typ, "code"))
	assert.Equal(t, "Transaction Number", jsonField(typ, "name"))
	intField(t, got, "value")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

func TestSysProperties_RetrieveUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(sysPropertiesPath+"/"+mustGenID(t, "sypp"), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// --- Update ---

// A PATCH without a value is refused and leaves the counter where it is: zeroing it would reissue every
// number the counter had handed out. The seeded counter is the account's transaction numbers, which
// other tests draw from, so its value is never changed here.
func TestSysProperties_UpdateWithoutValueKeepsTheCounter(t *testing.T) {
	// Not parallel: the closing PATCH writes back the value read first, which would rewind the counter
	// past any transaction number a parallel test drew in between.
	path := sysPropertiesPath + "/" + SeedSysPropertyID

	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	before := parseJSON(body)["value"]

	status, body, err = apiClient.Patch(path, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	status, body, err = apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, before, parseJSON(body)["value"], "the refused PATCH left the counter alone")

	status, body, err = apiClient.Patch(path, map[string]any{"value": before}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, before, parseJSON(body)["value"])
}

// Realigning a counter is how an account carries on the numbering of a previous system. The new value
// must be what the response, a re-read and the list all show, and nothing else about the counter moves.
func TestSysProperties_UpdateMovesTheCounter(t *testing.T) {
	// Not parallel: moves the account's supplier counter.
	prop := sysPropertyByCode(t, apiClient, "supplier_number")
	id := jsonField(prop, "id")
	restoreSysPropertyOnCleanup(t, apiClient, id)
	target := unusedSupplierNumbers(t, 1)

	updated := setSysPropertyValue(t, apiClient, id, target)
	assert.Equal(t, id, jsonField(updated, "id"))
	assertObjectField(t, updated, "sys_property")
	assert.Equal(t, target, intField(t, updated, "value"))
	assert.Equal(t, prop["type"], updated["type"], "the counter's type does not change")
	assert.Equal(t, jsonField(prop, "created_at"), jsonField(updated, "created_at"))
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
	before, err := time.Parse(time.RFC3339Nano, jsonField(prop, "updated_at"))
	require.NoError(t, err)
	after, err := time.Parse(time.RFC3339Nano, jsonField(updated, "updated_at"))
	require.NoError(t, err)
	assert.False(t, after.Before(before), "updated_at moves forward: %s then %s", before, after)

	assert.Equal(t, target, sysPropertyValue(t, apiClient, id), "a re-read shows the new value")
	listed := listFindByField(t, sysPropertiesPath, nil, "id", id)
	require.NotNil(t, listed)
	assert.Equal(t, target, intField(t, parseJSON(listed), "value"), "the list shows the new value")
}

// A save that timed out gets retried. Replaying the key must answer the first result rather than write
// again, and reusing it for a different value must be refused rather than silently ignored.
func TestSysProperties_UpdateIsIdempotent(t *testing.T) {
	// Not parallel: moves the account's supplier counter.
	id := jsonField(sysPropertyByCode(t, apiClient, "supplier_number"), "id")
	restoreSysPropertyOnCleanup(t, apiClient, id)
	value := unusedSupplierNumbers(t, 2)
	key := newIdempotencyKey()
	path := sysPropertiesPath + "/" + id

	status, first, err := apiClient.Patch(path, map[string]any{"value": value}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, first)

	status, replay, err := apiClient.Patch(path, map[string]any{"value": value}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, replay)
	assert.Equal(t, jsonField(parseJSON(first), "updated_at"), jsonField(parseJSON(replay), "updated_at"),
		"a replay answers the stored result instead of writing again")
	assert.Equal(t, value, intField(t, parseJSON(replay), "value"))

	status, body, err := apiClient.Patch(path, map[string]any{"value": value + 1}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")

	assert.Equal(t, value, sysPropertyValue(t, apiClient, id), "the reused key changed nothing")
}

func TestSysProperties_UpdateRejectsANullValue(t *testing.T) {
	t.Parallel()
	id := jsonField(sysPropertyByCode(t, apiClient, "supplier_number"), "id")

	status, body, err := apiClient.Patch(sysPropertiesPath+"/"+id, map[string]any{"value": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "value")
}

// A value that is not a whole number a counter can hold is refused naming the field: the message is the
// toast the settings page shows when someone types 1.5. The request field is a field.Optional[int32],
// whose type error is raised inside its own UnmarshalJSON, where the decoder attaches no field.
func TestSysProperties_UpdateRejectsANonIntegerValueNamingTheField(t *testing.T) {
	t.Parallel()
	id := jsonField(sysPropertyByCode(t, apiClient, "supplier_number"), "id")

	for name, value := range map[string]any{
		"word":           "abc",
		"numeric string": "1001",
		"fraction":       1.5,
		"above int32":    int64(2147483648),
		"below int32":    int64(-2147483649),
		"object":         map[string]any{"value": 1},
		"array":          []int{1},
	} {
		t.Run(name, func(t *testing.T) {
			status, body, err := apiClient.Patch(sysPropertiesPath+"/"+id, map[string]any{"value": value}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
			assertErrorParam(t, errObj, "value")
			assert.Contains(t, errObj["message"], "'value'", "the message names the field")
		})
	}
}

// --- Next number ---

// Opening a create form reads the next number to prefill it. The read must hand out a number no record
// carries, and must not take it: opening the form twice, or abandoning it, must not burn numbers.
func TestSysProperties_LatestValueIsTheNextNumberWithoutTakingIt(t *testing.T) {
	// Not parallel: reading the next number advances a counter whose number is already taken, and these
	// counters are shared with every test that creates customers or purchase orders.
	for _, code := range dashboardNumberCounters {
		t.Run(code, func(t *testing.T) {
			id := jsonField(sysPropertyByCode(t, apiClient, code), "id")
			restoreSysPropertyOnCleanup(t, apiClient, id)

			want := sysPropertyValue(t, apiClient, id)
			for numberInUse(t, SeedAccountID, code, want) {
				want++
			}

			first := latestSysPropertyValue(t, apiClient, code)
			assert.Equal(t, want, first, "the next number is the first free one from the counter on")
			assert.False(t, numberInUse(t, SeedAccountID, code, first), "%d is already taken", first)
			assert.Equal(t, first, latestSysPropertyValue(t, apiClient, code), "reading it again must not take it")
			assert.Equal(t, first, sysPropertyValue(t, apiClient, id), "the counter rests on the number it handed out")
		})
	}
}

// A counter pointing at a number nobody holds hands that number out as it is.
func TestSysProperties_LatestValueKeepsAFreeNumber(t *testing.T) {
	// Not parallel: moves the account's supplier counter.
	id := jsonField(sysPropertyByCode(t, apiClient, "supplier_number"), "id")
	restoreSysPropertyOnCleanup(t, apiClient, id)
	free := unusedSupplierNumbers(t, 1)
	setSysPropertyValue(t, apiClient, id, free)

	assert.Equal(t, free, latestSysPropertyValue(t, apiClient, "supplier_number"))
	assert.Equal(t, free, sysPropertyValue(t, apiClient, id), "a free number is not stepped past")
}

// A counter sitting on a number a record already holds steps past it and stays there, so the form is
// never prefilled with a number its create would reject.
func TestSysProperties_LatestValueStepsPastANumberInUse(t *testing.T) {
	// Not parallel: moves the account's supplier counter.
	id := jsonField(sysPropertyByCode(t, apiClient, "supplier_number"), "id")
	restoreSysPropertyOnCleanup(t, apiClient, id)
	taken := unusedSupplierNumbers(t, 2)
	createSupplierNumbered(t, taken)
	setSysPropertyValue(t, apiClient, id, taken)

	assert.Equal(t, taken+1, latestSysPropertyValue(t, apiClient, "supplier_number"))
	assert.Equal(t, taken+1, sysPropertyValue(t, apiClient, id), "the counter moves past the taken number")
	assert.Equal(t, taken+1, latestSysPropertyValue(t, apiClient, "supplier_number"), "and the next read stays on it")
}

// The endpoint promises "the next available counter value", so with N and N+1 both held it must return
// N+2: N+1 would prefill a create that fails as a duplicate. A counter realigned below numbers already
// issued, the PATCH's documented use, lands here.
func TestSysProperties_LatestValueSkipsEveryNumberInUse(t *testing.T) {
	// Not parallel: moves the account's supplier counter.
	id := jsonField(sysPropertyByCode(t, apiClient, "supplier_number"), "id")
	restoreSysPropertyOnCleanup(t, apiClient, id)
	taken := unusedSupplierNumbers(t, 3)
	createSupplierNumbered(t, taken)
	createSupplierNumbered(t, taken+1)
	setSysPropertyValue(t, apiClient, id, taken)

	got := latestSysPropertyValue(t, apiClient, "supplier_number")
	assert.False(t, numberInUse(t, SeedAccountID, "supplier_number", got), "handed out supplier number %d, which a supplier already has", got)
	assert.Equal(t, taken+2, got, "the first free number after %d and %d", taken, taken+1)
}

// Each account numbers its own records, so the next number comes from the caller's counter alone.
func TestSysProperties_LatestValueReadsTheCallersOwnCounter(t *testing.T) {
	// Not parallel: moves both tenants' supplier counters.
	tenantB := getTenantBClient()
	id := jsonField(sysPropertyByCode(t, apiClient, "supplier_number"), "id")
	restoreSysPropertyOnCleanup(t, apiClient, id)
	tenantBID := jsonField(sysPropertyByCode(t, tenantB, "supplier_number"), "id")
	restoreSysPropertyOnCleanup(t, tenantB, tenantBID)

	mine := unusedSupplierNumbers(t, 1)
	setSysPropertyValue(t, apiClient, id, mine)
	assert.Equal(t, mine, latestSysPropertyValue(t, apiClient, "supplier_number"))

	theirs := latestSysPropertyValue(t, tenantB, "supplier_number")
	assert.NotEqual(t, mine, theirs, "tenant B must not be handed this account's number")
	assert.Equal(t, sysPropertyValue(t, tenantB, tenantBID), theirs, "tenant B's number comes from its own counter")
	assert.Equal(t, mine, sysPropertyValue(t, apiClient, id), "tenant B's read left this account's counter alone")
}

// An unknown type would otherwise be written through as a counter of a type that does not exist; it
// must be refused as the caller's mistake, naming the field.
func TestSysProperties_LatestValueRejectsAnUnknownType(t *testing.T) {
	t.Parallel()

	for _, code := range []string{"invoice_number", "CUSTOMER_NUMBER", uniqueName("e2e-counter")} {
		t.Run(code, func(t *testing.T) {
			status, body, err := apiClient.GetListRaw(latestValuePath(code), nil)
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error"), "type_code")
		})
	}
}

// --- Tenancy ---

// Counters number one account's records. Another account must not see them nor move them: a moved
// counter reissues numbers already printed on invoices and orders.
func TestSysProperties_AnotherTenantCannotReadOrMoveACounter(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()
	prop := sysPropertyByCode(t, apiClient, "supplier_number")
	id := jsonField(prop, "id")

	status, body, err := tenantB.GetListRaw(sysPropertiesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	// The current value, so even a wrongly accepted write would leave the counter where it is.
	status, body, err = tenantB.Patch(sysPropertiesPath+"/"+id, map[string]any{"value": intField(t, prop, "value")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")

	theirs := sysPropertyIDs(listSysProperties(t, tenantB, nil))
	require.NotEmpty(t, theirs)
	for _, mine := range sysPropertyIDs(listSysProperties(t, apiClient, nil)) {
		assert.NotContains(t, theirs, mine, "tenant B's list must not include this account's counters")
	}

	status, body, err = apiClient.GetListRaw(sysPropertiesPath+"/"+theirs[0], nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// --- Authorization ---

// The dashboard reads counters as the signed-in user, not with an API key.
func TestSysProperties_SignedInUserCanReadCounters(t *testing.T) {
	t.Parallel()
	session := loginAsSeedUser(t)

	assert.Contains(t, sysPropertyIDs(listSysProperties(t, session, nil)), SeedSysPropertyID)

	status, body, err := session.GetListRaw(sysPropertiesPath+"/"+SeedSysPropertyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "transaction_number", sysPropertyCode(parseJSON(body)))
}

// Counters decide every number the account issues, so a role without the system properties permission
// can neither read nor move them, nor draw a next number.
func TestSysProperties_RequireTheSystemPropertiesPermission(t *testing.T) {
	t.Parallel()
	prop := sysPropertyByCode(t, apiClient, "supplier_number")
	id := jsonField(prop, "id")
	salesRep := roleScopedClient(t, SeedSalesRepRoleID)

	for _, path := range []string{sysPropertiesPath, sysPropertiesPath + "/" + id, latestValuePath("supplier_number")} {
		status, body, err := salesRep.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
		requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	}

	// The current value, so even a wrongly accepted write would leave the counter where it is.
	status, body, err := salesRep.Patch(sysPropertiesPath+"/"+id, map[string]any{"value": intField(t, prop, "value")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	status, body, err = apiClient.WithBearerToken("", SeedAccountID).GetListRaw(sysPropertiesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
}
