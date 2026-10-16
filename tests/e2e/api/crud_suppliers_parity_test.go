//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const supplierAddressesPath = "/v1/core/addresses"

// searchToken is a word unique to one test, made of letters and digits so a word search keeps it whole.
func searchToken(prefix string) string {
	return strings.ReplaceAll(uniqueName(prefix), "-", "")
}

func paritySupplierAddressBody(name, street string) map[string]any {
	return map[string]any{
		"name":        name,
		"line_1":      street,
		"city":        "Los Angeles",
		"state":       "CA",
		"postal_code": "90001",
		"country":     "US",
	}
}

// supplierAccountAddress saves an address on the supplier's own account, as the dashboard's supplier page does.
func supplierAccountAddress(t *testing.T, supplierID, name string) string {
	t.Helper()
	status, body, err := apiClient.WithAccountID(supplierID).Post(supplierAddressesPath,
		paritySupplierAddressBody(name, "9 Spindle Way"), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	return id
}

// linkSupplierItem records that the supplier provides the item and deletes the supplier item after the test.
func linkSupplierItem(t *testing.T, supplierID, itemID string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"item_id": itemID, "supplier_sku": uniqueName("PN")}
	for k, v := range extra {
		body[k] = v
	}
	status, resp, err := apiClient.Post(supplierItemsPath(supplierID), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	created := parseJSON(resp)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(supplierItemsPath(supplierID) + "/" + jsonField(created, "id")) })
	return created
}

func listedSupplierIDs(t *testing.T, client *Client, params url.Values) []string {
	t.Helper()
	merged := url.Values{"limit": {"100"}}
	for k, v := range params {
		merged[k] = v
	}
	status, body, err := client.GetListRaw(suppliersPath, merged)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	ids := []string{}
	for _, row := range jsonArray(parseJSON(body), "data") {
		ids = append(ids, jsonField(row.(map[string]any), "id"))
	}
	return ids
}

func getSupplier(t *testing.T, id string, include ...string) map[string]any {
	t.Helper()
	var params url.Values
	if len(include) > 0 {
		params = url.Values{"include": {strings.Join(include, ",")}}
	}
	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+id, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// supplierDefault is the supplier's defaults.<key>.
func supplierDefault(supplier map[string]any, key string) map[string]any {
	return jsonObject(jsonObject(supplier, "defaults"), key)
}

func patchSupplier(t *testing.T, path string, body map[string]any) (int, map[string]any, []byte) {
	t.Helper()
	status, resp, err := apiClient.Patch(path, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(resp))
	return status, parseJSON(resp), resp
}

func requireErrorParam(t *testing.T, wantStatus, status int, body []byte, param string) {
	t.Helper()
	requireStatus(t, wantStatus, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), param)
}

// --- Create ---

func TestSupplierParity_CreateMinimalAssertsEveryField(t *testing.T) {
	t.Parallel()
	name, number := uniqueName("e2e-sup-min"), uniqueName("SUP")

	got := paritySupplier(t, map[string]any{"name": name, "number": number})

	assertIDFormat(t, jsonField(got, "id"), "ac")
	assertObjectField(t, got, "supplier")
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, number, jsonField(got, "number"))
	assertNilField(t, got, "instructions")
	for _, key := range []string{"bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term"} {
		assertNilField(t, jsonObject(got, "defaults"), key)
	}
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

func paritySupplier(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	return createAndCleanup(t, suppliersPath, body)
}

func TestSupplierParity_CreateReturnsAddressesWhenIncluded(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-addr")
	status, body, err := apiClient.Post(suppliersPath+"?include=defaults.bill_to_address,defaults.ship_to_address", map[string]any{
		"name":            name,
		"number":          uniqueName("SUP"),
		"instructions":    "net 30",
		"bill_to_address": paritySupplierAddressBody(name+" Billing", "1 Bill St"),
		"ship_to_address": paritySupplierAddressBody(name+" Dock", "2 Dock Rd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	id := jsonField(got, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + id) })

	assert.Equal(t, "net 30", jsonField(got, "instructions"))
	billTo, shipTo := supplierDefault(got, "bill_to_address"), supplierDefault(got, "ship_to_address")
	require.NotNil(t, billTo, "defaults.bill_to_address is returned from create when included")
	require.NotNil(t, shipTo, "defaults.ship_to_address is returned from create when included")
	assertObjectField(t, billTo, "address")
	assert.Equal(t, name+" Billing", jsonField(billTo, "name"))
	assert.Equal(t, "1 Bill St", jsonField(billTo, "line_1"))
	assert.Equal(t, name+" Dock", jsonField(shipTo, "name"))
	assert.NotEqual(t, jsonField(billTo, "id"), jsonField(shipTo, "id"), "two different addresses are two records")

	plain := getSupplier(t, id)
	assertNilField(t, jsonObject(plain, "defaults"), "bill_to_address")
	assertNilField(t, jsonObject(plain, "defaults"), "ship_to_address")
	included := getSupplier(t, id, "defaults.bill_to_address", "defaults.ship_to_address")
	assert.Equal(t, jsonField(billTo, "id"), jsonField(supplierDefault(included, "bill_to_address"), "id"))
	assert.Equal(t, jsonField(shipTo, "id"), jsonField(supplierDefault(included, "ship_to_address"), "id"))
}

// The dashboard omits ship_to when it equals bill_to; the bill-to address then becomes both defaults.
func TestSupplierParity_CreateWithOnlyBillToDefaultsShipTo(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-same")
	got := paritySupplier(t, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody(name, "3 Same St"),
	})

	read := getSupplier(t, jsonField(got, "id"), "defaults.bill_to_address", "defaults.ship_to_address")
	billID := jsonField(supplierDefault(read, "bill_to_address"), "id")
	require.NotEmpty(t, billID)
	assert.Equal(t, billID, jsonField(supplierDefault(read, "ship_to_address"), "id"))
}

func TestSupplierParity_CreateWithIdenticalAddressesSavesOne(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-twin")
	address := paritySupplierAddressBody(name, "4 Twin Ave")
	got := paritySupplier(t, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": address, "ship_to_address": address,
	})

	read := getSupplier(t, jsonField(got, "id"), "defaults.bill_to_address", "defaults.ship_to_address")
	billID := jsonField(supplierDefault(read, "bill_to_address"), "id")
	require.NotEmpty(t, billID)
	assert.Equal(t, billID, jsonField(supplierDefault(read, "ship_to_address"), "id"), "an identical ship-to reuses the bill-to")
}

func TestSupplierParity_CreateDuplicateNumberConflicts(t *testing.T) {
	t.Parallel()
	number := uniqueName("SUP")
	paritySupplier(t, map[string]any{"name": uniqueName("e2e-sup-dup"), "number": number})

	status, body, err := apiClient.Post(suppliersPath, map[string]any{"name": uniqueName("e2e-sup-dup2"), "number": number}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 409, status, body, "number")
}

func TestSupplierParity_CreateValidation(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]map[string]any{
		"missing name":       {"number": uniqueName("SUP")},
		"missing number":     {"name": uniqueName("e2e-sup-val")},
		"blank instructions": {"name": uniqueName("e2e-sup-val"), "number": uniqueName("SUP"), "instructions": ""},
	} {
		status, resp, err := apiClient.Post(suppliersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 422, status, "%s: %s", name, string(resp))
	}
}

func TestSupplierParity_CreateIsIdempotent(t *testing.T) {
	t.Parallel()
	key := newIdempotencyKey()
	body := map[string]any{"name": uniqueName("e2e-sup-idem"), "number": uniqueName("SUP")}

	status1, resp1, err := apiClient.Post(suppliersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status1, resp1)
	id := jsonField(parseJSON(resp1), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + id) })

	status2, resp2, err := apiClient.Post(suppliersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status2, resp2)
	assert.Equal(t, id, jsonField(parseJSON(resp2), "id"), "a replay returns the supplier the first request created")
}

// --- List ---

func TestSupplierParity_ListSearchMatchesEveryWordAcrossNameAndNumber(t *testing.T) {
	t.Parallel()
	shared, numberTok, instructionsTok := searchToken("shr"), searchToken("num"), searchToken("ins")
	alpha := jsonField(paritySupplier(t, map[string]any{
		"name": "Alpha " + shared + " Mills", "number": "N-" + numberTok, "instructions": "dyes " + instructionsTok,
	}), "id")
	beta := jsonField(paritySupplier(t, map[string]any{"name": "Beta " + shared, "number": uniqueName("SUP")}), "id")

	search := func(q string) []string { return listedSupplierIDs(t, apiClient, url.Values{"q": {q}}) }
	assert.ElementsMatch(t, []string{alpha, beta}, search(shared))
	assert.Equal(t, []string{alpha}, search("mills "+shared), "words match in any order and case")
	assert.Equal(t, []string{alpha}, search(numberTok), "the number is searched")
	assert.Equal(t, []string{alpha}, search("N-"+numberTok), "a number keeps its punctuation")
	assert.Equal(t, []string{alpha}, search(shared+" "+numberTok), "words may match different fields")
	assert.Empty(t, search(instructionsTok), "instructions are not searched")
	assert.Empty(t, search(shared+" Beta Mills"), "no supplier has every word")
	assert.Empty(t, search(shared+" zzznomatchzzz"))
}

func TestSupplierParity_ListFiltersByItemAndDateRange(t *testing.T) {
	t.Parallel()
	tok := searchToken("flt")
	linked := paritySupplier(t, map[string]any{"name": "Linked " + tok, "number": uniqueName("SUP")})
	other := jsonField(paritySupplier(t, map[string]any{"name": "Other " + tok, "number": uniqueName("SUP")}), "id")
	linkedID := jsonField(linked, "id")
	itemID := forgeMaterialItem(t)
	linkSupplierItem(t, linkedID, itemID, nil)

	assert.Equal(t, []string{linkedID}, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "item_ids": {itemID}}))
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "item_ids": {mustGenID(t, "it")}}))

	createdAt, err := time.Parse(time.RFC3339Nano, jsonField(linked, "created_at"))
	require.NoError(t, err)
	window := url.Values{
		"q":              {tok},
		"created_after":  {createdAt.Add(-time.Minute).UTC().Format(time.RFC3339)},
		"created_before": {createdAt.Add(time.Minute).UTC().Format(time.RFC3339)},
	}
	assert.ElementsMatch(t, []string{linkedID, other}, listedSupplierIDs(t, apiClient, window))
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "created_before": {createdAt.Add(-time.Hour).UTC().Format(time.RFC3339)}}))
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "created_after": {createdAt.Add(time.Hour).UTC().Format(time.RFC3339)}}))
}

func TestSupplierParity_ListPagesEveryRowOnceBothWays(t *testing.T) {
	t.Parallel()
	tok := searchToken("page")
	var ids []string
	for i := 0; i < 3; i++ {
		ids = append(ids, jsonField(paritySupplier(t, map[string]any{"name": "Pager " + tok, "number": uniqueName("SUP")}), "id"))
	}
	params := url.Values{"q": {tok}}
	assertScopedCursorPagination(t, suppliersPath, params, ids)

	list, _, err := apiClient.GetList(suppliersPath, url.Values{"q": {tok}, "limit": {"1"}})
	require.NoError(t, err)
	for i := 0; i < len(ids) && list.PageInfo.NextPageURL != nil; i++ {
		list, _, err = apiClient.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
	}
	var backward []string
	for page := 0; page <= len(ids); page++ {
		for _, row := range list.Data {
			backward = append(backward, DataItemField(row, "id"))
		}
		if !list.PageInfo.HasPrevPage() || list.PageInfo.PreviousPageURL == nil {
			break
		}
		list, _, err = apiClient.GetListFromPageURL(list.PageInfo.PreviousPageURL)
		require.NoError(t, err)
	}
	assert.ElementsMatch(t, ids, backward, "walking back from the last page visits each row once")
}

// preview.9's material_count is how many supplier items the supplier has, on the list and the supplier alike.
func TestSupplierParity_Preview9MaterialCountCountsTheSupplierItems(t *testing.T) {
	t.Parallel()
	tok := searchToken("cnt")
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": "Counted " + tok, "number": uniqueName("SUP")}), "id")
	linkSupplierItem(t, supplierID, forgeMaterialItem(t), nil)
	linkSupplierItem(t, supplierID, forgeMaterialItem(t), nil)

	p9 := apiClient.WithAPIVersion(preview9Version)
	status, body, err := p9.GetListRaw(suppliersPath+"/"+supplierID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "2", jsonField(parseJSON(body), "material_count"))

	status, body, err = p9.GetListRaw(suppliersPath, url.Values{"q": {tok}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	rows := jsonArray(parseJSON(body), "data")
	require.Len(t, rows, 1)
	assert.Equal(t, "2", jsonField(rows[0].(map[string]any), "material_count"))

	assertNilField(t, getSupplier(t, supplierID), "material_count")
}

// --- Update ---

// A rename is the owner's name for the supplier: it lands on the relation, not on the supplier's own account.
func TestSupplierParity_RenameWritesTheAliasEverywhereItIsRead(t *testing.T) {
	t.Parallel()
	oldTok, newTok := searchToken("old"), searchToken("new")
	created := paritySupplier(t, map[string]any{
		"name": "Before " + oldTok, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody("Rename Billing", "5 Rename Rd"),
	})
	id := jsonField(created, "id")
	newName := "After " + newTok

	status, got, body := patchSupplier(t, suppliersPath+"/"+id, map[string]any{"name": newName})
	requireStatus(t, 200, status, body)
	assert.Equal(t, newName, jsonField(got, "name"))
	assert.Equal(t, newName, jsonField(getSupplier(t, id), "name"))
	assert.Equal(t, []string{id}, listedSupplierIDs(t, apiClient, url.Values{"q": {newTok}}))
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {oldTok}}))

	var accountName, alias string
	require.NoError(t, authDB(t).QueryRow("SELECT name FROM account WHERE id = ?", id).Scan(&accountName))
	require.NoError(t, authDB(t).QueryRow(
		"SELECT alias FROM account_relation WHERE counterparty_account_id = ? AND account_relation_role_code = 'supplier'", id).Scan(&alias))
	assert.Equal(t, "Before "+oldTok, accountName, "the supplier's own account keeps its name")
	assert.Equal(t, newName, alias)

	event := expectAuditEventWithChanges(t, id, "supplier", "update")
	changes := jsonListData(event, "changes")
	nameChange, ok := changeForField(changes, "name")
	require.True(t, ok, "the rename is audited: %v", changes)
	assert.Equal(t, "Before "+oldTok, jsonField(nameChange, "old_value"))
	assert.Equal(t, newName, jsonField(nameChange, "new_value"))
	_, billChanged := changeForField(changes, "bill_to_address_id")
	assert.False(t, billChanged, "an address the update did not touch is not audited as changed: %v", changes)
}

func TestSupplierParity_UpdateNumberAndInstructions(t *testing.T) {
	t.Parallel()
	takenNumber := uniqueName("SUP")
	paritySupplier(t, map[string]any{"name": uniqueName("e2e-sup-taken"), "number": takenNumber})
	id := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-sup-upd"), "number": uniqueName("SUP")}), "id")
	path := suppliersPath + "/" + id

	newNumber := uniqueName("SUP")
	status, got, body := patchSupplier(t, path, map[string]any{"number": newNumber})
	requireStatus(t, 200, status, body)
	assert.Equal(t, newNumber, jsonField(got, "number"))

	status, _, body = patchSupplier(t, path, map[string]any{"number": takenNumber})
	requireErrorParam(t, 409, status, body, "number")

	status, got, body = patchSupplier(t, path, map[string]any{"instructions": "call first"})
	requireStatus(t, 200, status, body)
	assert.Equal(t, "call first", jsonField(got, "instructions"))

	status, got, body = patchSupplier(t, path, map[string]any{"name": jsonField(got, "name")})
	requireStatus(t, 200, status, body)
	assert.Equal(t, "call first", jsonField(got, "instructions"), "instructions the update omitted are kept")

	status, got, body = patchSupplier(t, path, map[string]any{"instructions": nil})
	requireStatus(t, 200, status, body)
	assertNilField(t, got, "instructions")
	assert.Equal(t, newNumber, jsonField(got, "number"), "fields the update omitted are kept")
}

func TestSupplierParity_UpdateDefaultAddresses(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-defaults")
	id := jsonField(paritySupplier(t, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody(name, "6 First St"),
	}), "id")
	path := suppliersPath + "/" + id + "?include=defaults.bill_to_address,defaults.ship_to_address"
	original := jsonField(supplierDefault(getSupplier(t, id, "defaults.ship_to_address"), "ship_to_address"), "id")
	saved := supplierAccountAddress(t, id, name+" Annex")

	status, got, body := patchSupplier(t, path, map[string]any{"defaults": map[string]any{"bill_to_address_id": saved}})
	requireStatus(t, 200, status, body)
	assert.Equal(t, saved, jsonField(supplierDefault(got, "bill_to_address"), "id"), "the update returns its addresses when included")
	assert.Equal(t, original, jsonField(supplierDefault(got, "ship_to_address"), "id"), "the other default is kept")

	status, got, body = patchSupplier(t, path, map[string]any{"defaults": map[string]any{"ship_to_address_id": saved}})
	requireStatus(t, 200, status, body)
	assert.Equal(t, saved, jsonField(supplierDefault(got, "ship_to_address"), "id"))
	read := getSupplier(t, id, "defaults.bill_to_address", "defaults.ship_to_address")
	assert.Equal(t, saved, jsonField(supplierDefault(read, "bill_to_address"), "id"))
	assert.Equal(t, saved, jsonField(supplierDefault(read, "ship_to_address"), "id"))

	plainStatus, plain, plainBody := patchSupplier(t, suppliersPath+"/"+id, map[string]any{"instructions": "x"})
	requireStatus(t, 200, plainStatus, plainBody)
	assertNilField(t, jsonObject(plain, "defaults"), "bill_to_address")
	assertNilField(t, jsonObject(plain, "defaults"), "ship_to_address")
}

// A default address must be one of the supplier's own; another account's would be read back through the supplier.
func TestSupplierParity_UpdateRefusesAnAddressThatIsNotTheSuppliers(t *testing.T) {
	t.Parallel()
	id := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-sup-foreign"), "number": uniqueName("SUP")}), "id")
	path := suppliersPath + "/" + id

	ownersAddress := createE2EAddress(t, uniqueName("e2e-owner-addr"))
	status, _, body := patchSupplier(t, path, map[string]any{"defaults": map[string]any{"bill_to_address_id": ownersAddress}})
	requireErrorParam(t, 404, status, body, "defaults.bill_to_address_id")

	bStatus, bBody, err := getTenantBClient().Post(supplierAddressesPath, paritySupplierAddressBody(uniqueName("e2e-tenantb-addr"), "7 Other Co"), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, bStatus, bBody)
	tenantBAddress := jsonField(parseJSON(bBody), "id")
	t.Cleanup(func() { _, _, _ = getTenantBClient().Delete(supplierAddressesPath + "/" + tenantBAddress) })
	status, _, body = patchSupplier(t, path, map[string]any{"defaults": map[string]any{"ship_to_address_id": tenantBAddress}})
	requireErrorParam(t, 404, status, body, "defaults.ship_to_address_id")

	status, _, body = patchSupplier(t, path, map[string]any{"defaults": map[string]any{"bill_to_address_id": mustGenID(t, "ad")}})
	requireErrorParam(t, 404, status, body, "defaults.bill_to_address_id")

	read := getSupplier(t, id, "defaults.bill_to_address", "defaults.ship_to_address")
	assertNilField(t, jsonObject(read, "defaults"), "bill_to_address")
	assertNilField(t, jsonObject(read, "defaults"), "ship_to_address")
}

// --- Delete ---

func TestSupplierParity_DeleteReturnsAStubThenIsGone(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-del")
	status, body, err := apiClient.Post(suppliersPath, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody(name, "8 Gone St"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	status, body, err = apiClient.Delete(suppliersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertDeletedStub(t, body, id, "supplier")

	getStatus, getBody, err := apiClient.GetListRaw(suppliersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus, string(getBody))

	againStatus, againBody, err := apiClient.Delete(suppliersPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, againStatus, "a second delete is gone, not a fresh success: %s", string(againBody))

	patchStatus, _, patchBody := patchSupplier(t, suppliersPath+"/"+id, map[string]any{"instructions": "x"})
	assert.Equal(t, 404, patchStatus, string(patchBody))

	expectAuditEventWithChanges(t, id, "supplier", "delete")
}

func TestSupplierParity_CreateIsAudited(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-audit")
	id := jsonField(paritySupplier(t, map[string]any{"name": name, "number": uniqueName("SUP")}), "id")

	nameChange, ok := changeForField(jsonListData(expectAuditEventWithChanges(t, id, "supplier", "create"), "changes"), "name")
	require.True(t, ok)
	assert.Equal(t, name, jsonField(nameChange, "new_value"))
}

// --- Tenant isolation and permissions ---

func TestSupplierParity_TenantIsolationOnEveryRoute(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()
	tok := searchToken("iso")
	id := jsonField(paritySupplier(t, map[string]any{"name": "Isolated " + tok, "number": uniqueName("SUP")}), "id")
	itemID := forgeMaterialItem(t)
	link := supplierItemsPath(id) + "/" + jsonField(linkSupplierItem(t, id, itemID, nil), "id")

	status, body, err := clientB.GetListRaw(suppliersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Patch(suppliersPath+"/"+id, map[string]any{"name": "taken over"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Delete(suppliersPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	assert.Equal(t, []string{id}, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}}), "the supplier survives tenant B")
	assert.Empty(t, listedSupplierIDs(t, clientB, url.Values{"q": {tok}}))

	status, body, err = clientB.GetListRaw(supplierItemsPath(id), nil)
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "supplier_id")
	status, body, err = clientB.GetListRaw(link, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Patch(link, map[string]any{"supplier_sku": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Delete(link)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Post(supplierItemsPath(id), map[string]any{"item_id": itemID, "supplier_sku": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "supplier_id")

	status, body, err = apiClient.GetListRaw(link, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestSupplierParity_PermissionRefusals(t *testing.T) {
	t.Parallel()
	id := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-sup-perm"), "number": uniqueName("SUP")}), "id")
	itemID := forgeMaterialItem(t)
	link := supplierItemsPath(id) + "/" + jsonField(linkSupplierItem(t, id, itemID, nil), "id")
	role := createAndCleanup(t, rolesPath, map[string]any{"name": uniqueName("e2e-sup-reader"), "permissions": []string{"suppliers:read"}})
	reader := roleScopedClient(t, jsonField(role, "id"))

	status, body, err := reader.GetListRaw(suppliersPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = reader.GetListRaw(suppliersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = reader.GetListRaw(link, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for what, call := range map[string]func() (int, []byte, error){
		"create supplier": func() (int, []byte, error) {
			return reader.Post(suppliersPath, map[string]any{"name": uniqueName("x"), "number": uniqueName("SUP")}, newIdempotencyKey())
		},
		"update supplier": func() (int, []byte, error) {
			return reader.Patch(suppliersPath+"/"+id, map[string]any{"name": "x"}, newIdempotencyKey())
		},
		"delete supplier": func() (int, []byte, error) { return reader.Delete(suppliersPath + "/" + id) },
		"create supplier item": func() (int, []byte, error) {
			return reader.Post(supplierItemsPath(id), map[string]any{"item_id": itemID, "supplier_sku": "x"}, newIdempotencyKey())
		},
		"update supplier item": func() (int, []byte, error) {
			return reader.Patch(link, map[string]any{"supplier_sku": "x"}, newIdempotencyKey())
		},
		"delete supplier item": func() (int, []byte, error) { return reader.Delete(link) },
	} {
		status, body, err := call()
		require.NoError(t, err)
		assert.Equal(t, 403, status, "%s needs more than suppliers:read: %s", what, string(body))
	}

	scanner := roleScopedClient(t, SeedScannerRoleID)
	status, body, err = scanner.GetListRaw(suppliersPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "a role without suppliers cannot list them: %s", string(body))
}

// --- Supplier items ---

// The column is TEXT, so a description runs to 65,535 bytes; the request counts characters, the service bytes.
func TestSupplierItemParity_LongDescriptions(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supitem-long"), "number": uniqueName("SUP")}), "id")
	long := strings.Repeat("spec ", 1000)
	created := linkSupplierItem(t, supplierID, forgeMaterialItem(t), map[string]any{"supplier_description": long})
	assert.Equal(t, long, jsonField(created, "supplier_description"))
	path := supplierItemsPath(supplierID) + "/" + jsonField(created, "id")

	full := strings.Repeat("a", 65535)
	status, got, body := patchSupplier(t, path, map[string]any{"supplier_description": full})
	requireStatus(t, 200, status, body)
	assert.Len(t, jsonField(got, "supplier_description"), 65535)

	status, _, body = patchSupplier(t, path, map[string]any{"supplier_description": full + "a"})
	requireErrorParam(t, 422, status, body, "supplier_description")

	status, _, body = patchSupplier(t, path, map[string]any{"supplier_description": strings.Repeat("🧶", 17000)})
	requireErrorParam(t, 422, status, body, "supplier_description")
	assert.Len(t, jsonField(parseJSON(mustGet(t, path)), "supplier_description"), 65535, "a refused update changes nothing")
}

func TestSupplierItemParity_RefusesAnotherTenantsItemOrSupplier(t *testing.T) {
	t.Parallel()
	itemID := forgeMaterialItem(t)

	// Tenant B names tenant A's item on its own supplier.
	clientB := getTenantBClient()
	bStatus, bBody, err := clientB.Post(suppliersPath, map[string]any{"name": uniqueName("e2e-tenantb-sup"), "number": uniqueName("SUP")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, bStatus, bBody)
	tenantBSupplier := jsonField(parseJSON(bBody), "id")
	t.Cleanup(func() { _, _, _ = clientB.Delete(suppliersPath + "/" + tenantBSupplier) })
	status, body, err := clientB.Post(supplierItemsPath(tenantBSupplier), map[string]any{"item_id": itemID, "supplier_sku": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "item_id")

	// Tenant A names tenant B's supplier.
	status, body, err = apiClient.Post(supplierItemsPath(tenantBSupplier), map[string]any{"item_id": itemID, "supplier_sku": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "supplier_id")
}

func TestSupplierItemParity_IsAudited(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supitem-audit"), "number": uniqueName("SUP")}), "id")
	otherSupplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supitem-audit-b"), "number": uniqueName("SUP")}), "id")
	itemID := forgeMaterialItem(t)
	sku := uniqueName("PN")
	link := linkSupplierItem(t, supplierID, itemID, map[string]any{"supplier_sku": sku})
	linkID := jsonField(link, "id")
	path := supplierItemsPath(supplierID) + "/" + linkID

	created, ok := changeForField(jsonListData(expectAuditEventWithChanges(t, linkID, "supplier_item", "create"), "changes"), "supplier_sku")
	require.True(t, ok)
	assert.Equal(t, sku, jsonField(created, "new_value"))

	otherLinkID := jsonField(linkSupplierItem(t, otherSupplierID, itemID, nil), "id")
	expectSupplierItemRootedEvents(t, otherSupplierID, otherLinkID, 1)
	expectSupplierItemRootedEvents(t, supplierID, linkID, 1)

	status, _, body := patchSupplier(t, path, map[string]any{"supplier_sku": sku + "-B"})
	requireStatus(t, 200, status, body)
	updated, ok := changeForField(jsonListData(expectAuditEventWithChanges(t, linkID, "supplier_item", "update"), "changes"), "supplier_sku")
	require.True(t, ok)
	assert.Equal(t, sku+"-B", jsonField(updated, "new_value"))
	expectSupplierItemRootedEvents(t, supplierID, linkID, 2)

	status, body, err := apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	expectAuditEventWithChanges(t, linkID, "supplier_item", "delete")
	expectSupplierItemRootedEvents(t, supplierID, linkID, 3)
}

// expectSupplierItemRootedEvents polls until the supplier's history holds want events for the supplier item.
func expectSupplierItemRootedEvents(t *testing.T, supplierID, supplierItemID string, want int) {
	t.Helper()
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
			"root_resource_types": {"supplier"},
			"root_resource_ids":   {supplierID},
			"resource_types":      {"supplier_item"},
			"resource_ids":        {supplierItemID},
			"limit":               {"100"},
		})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("audit events: %d %s", status, body)
		}
		if got := len(jsonArray(parseJSON(body), "data")); got != want {
			return fmt.Errorf("supplier %s has %d supplier_item events for %s, want %d", supplierID, got, supplierItemID, want)
		}
		return nil
	})
}
