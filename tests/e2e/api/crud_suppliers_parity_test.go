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

const supplierAddressesPath = "/v1/sales/addresses"

func supplierMaterialsOf(supplierID string) string {
	return suppliersPath + "/" + supplierID + "/materials"
}

// searchToken is a word unique to one test, made of letters and digits so a word search keeps it whole.
func searchToken(prefix string) string {
	return strings.ReplaceAll(uniqueName(prefix), "-", "")
}

func paritySupplierAddressBody(name, street string) map[string]any {
	return map[string]any{
		"name":          name,
		"street_line_1": street,
		"locality":      "Los Angeles",
		"state":         "CA",
		"postal_code":   "90001",
		"country":       "US",
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

// parityMaterial creates a material and returns it with the id of its catalog item.
func parityMaterial(t *testing.T) (materialID, itemID string) {
	t.Helper()
	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-supmat")))
	materialID = jsonField(created, "id")
	require.NoError(t, authDB(t).QueryRow("SELECT item_id FROM material WHERE id = ?", materialID).Scan(&itemID))
	return materialID, itemID
}

func linkSupplierMaterial(t *testing.T, supplierID, materialID string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"material_id": materialID, "supplier_part_number": uniqueName("PN")}
	for k, v := range extra {
		body[k] = v
	}
	status, resp, err := apiClient.Post(supplierMaterialsOf(supplierID), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(supplierMaterialsOf(supplierID) + "/" + materialID) })
	return parseJSON(resp)
}

// softDeleteItem marks an item deleted the way the catalog does, and restores it when the test ends.
func softDeleteItem(t *testing.T, itemID string) {
	t.Helper()
	_, err := authDB(t).Exec("UPDATE item SET deleted_at = NOW(3) WHERE id = ?", itemID)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = authDB(t).Exec("UPDATE item SET deleted_at = NULL WHERE id = ?", itemID) })
}

func supplierMaterialRowID(t *testing.T, supplierID, materialID string) string {
	t.Helper()
	var id string
	require.NoError(t, authDB(t).QueryRow(
		"SELECT id FROM supplier_material WHERE supplier_account_id = ? AND material_id = ?", supplierID, materialID).Scan(&id))
	return id
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
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, number, jsonField(got, "number"))
	assertNilField(t, got, "note")
	assertNilField(t, got, "bill_to_address")
	assertNilField(t, got, "ship_to_address")
	assert.Equal(t, "0", jsonField(got, "material_count"), "a new supplier has no materials")
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
	status, body, err := apiClient.Post(suppliersPath+"?include=bill_to_address,ship_to_address", map[string]any{
		"name":            name,
		"number":          uniqueName("SUP"),
		"note":            "net 30",
		"bill_to_address": paritySupplierAddressBody(name+" Billing", "1 Bill St"),
		"ship_to_address": paritySupplierAddressBody(name+" Dock", "2 Dock Rd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	id := jsonField(got, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + id) })

	assert.Equal(t, "net 30", jsonField(got, "note"))
	billTo, shipTo := jsonObject(got, "bill_to_address"), jsonObject(got, "ship_to_address")
	require.NotNil(t, billTo, "bill_to_address is returned from create when included")
	require.NotNil(t, shipTo, "ship_to_address is returned from create when included")
	assertObjectField(t, billTo, "address")
	assert.Equal(t, name+" Billing", jsonField(billTo, "name"))
	assert.Equal(t, "1 Bill St", jsonField(jsonObject(billTo, "geolocation"), "street_line_1"))
	assert.Equal(t, name+" Dock", jsonField(shipTo, "name"))
	assert.NotEqual(t, jsonField(billTo, "id"), jsonField(shipTo, "id"), "two different addresses are two records")

	plain := getSupplier(t, id)
	assertNilField(t, plain, "bill_to_address")
	assertNilField(t, plain, "ship_to_address")
	included := getSupplier(t, id, "bill_to_address", "ship_to_address")
	assert.Equal(t, jsonField(billTo, "id"), jsonField(jsonObject(included, "bill_to_address"), "id"))
	assert.Equal(t, jsonField(shipTo, "id"), jsonField(jsonObject(included, "ship_to_address"), "id"))
}

// The dashboard omits ship_to when it equals bill_to; the bill-to address then becomes both defaults.
func TestSupplierParity_CreateWithOnlyBillToDefaultsShipTo(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-same")
	got := paritySupplier(t, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody(name, "3 Same St"),
	})

	read := getSupplier(t, jsonField(got, "id"), "bill_to_address", "ship_to_address")
	billID := jsonField(jsonObject(read, "bill_to_address"), "id")
	require.NotEmpty(t, billID)
	assert.Equal(t, billID, jsonField(jsonObject(read, "ship_to_address"), "id"))
}

func TestSupplierParity_CreateWithIdenticalAddressesSavesOne(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-twin")
	address := paritySupplierAddressBody(name, "4 Twin Ave")
	got := paritySupplier(t, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": address, "ship_to_address": address,
	})

	read := getSupplier(t, jsonField(got, "id"), "bill_to_address", "ship_to_address")
	billID := jsonField(jsonObject(read, "bill_to_address"), "id")
	require.NotEmpty(t, billID)
	assert.Equal(t, billID, jsonField(jsonObject(read, "ship_to_address"), "id"), "an identical ship-to reuses the bill-to")
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
		"missing name":   {"number": uniqueName("SUP")},
		"missing number": {"name": uniqueName("e2e-sup-val")},
		"blank note":     {"name": uniqueName("e2e-sup-val"), "number": uniqueName("SUP"), "note": ""},
	} {
		status, resp, err := apiClient.Post(suppliersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 400, status, "%s: %s", name, string(resp))
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

func TestSupplierParity_ListSearchMatchesEveryWordAcrossNameNumberAndNotes(t *testing.T) {
	t.Parallel()
	shared, numberTok, noteTok := searchToken("shr"), searchToken("num"), searchToken("note")
	alpha := jsonField(paritySupplier(t, map[string]any{
		"name": "Alpha " + shared + " Mills", "number": "N-" + numberTok, "note": "dyes " + noteTok,
	}), "id")
	beta := jsonField(paritySupplier(t, map[string]any{"name": "Beta " + shared, "number": uniqueName("SUP")}), "id")

	search := func(q string) []string { return listedSupplierIDs(t, apiClient, url.Values{"q": {q}}) }
	assert.ElementsMatch(t, []string{alpha, beta}, search(shared))
	assert.Equal(t, []string{alpha}, search("mills "+shared), "words match in any order and case")
	assert.Equal(t, []string{alpha}, search(numberTok), "the number is searched")
	assert.Equal(t, []string{alpha}, search("N-"+numberTok), "a number keeps its punctuation")
	assert.Equal(t, []string{alpha}, search(noteTok), "the notes are searched")
	assert.Equal(t, []string{alpha}, search(shared+" "+noteTok), "words may match different fields")
	assert.Empty(t, search(shared+" Beta Mills"), "no supplier has every word")
	assert.Empty(t, search(shared+" zzznomatchzzz"))
}

func TestSupplierParity_ListFiltersByItemAndDateRange(t *testing.T) {
	t.Parallel()
	tok := searchToken("flt")
	linked := paritySupplier(t, map[string]any{"name": "Linked " + tok, "number": uniqueName("SUP")})
	other := jsonField(paritySupplier(t, map[string]any{"name": "Other " + tok, "number": uniqueName("SUP")}), "id")
	linkedID := jsonField(linked, "id")
	materialID, itemID := parityMaterial(t)
	linkSupplierMaterial(t, linkedID, materialID, nil)

	assert.Equal(t, []string{linkedID}, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "item_ids": {itemID}}))
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "item_ids": {mustGenID(t, "it")}}))

	createdAt, err := time.Parse(time.RFC3339Nano, jsonField(linked, "created_at"))
	require.NoError(t, err)
	window := url.Values{
		"q":         {tok},
		"starts_at": {createdAt.Add(-time.Minute).UTC().Format(time.RFC3339)},
		"ends_at":   {createdAt.Add(time.Minute).UTC().Format(time.RFC3339)},
	}
	assert.ElementsMatch(t, []string{linkedID, other}, listedSupplierIDs(t, apiClient, window))
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "ends_at": {createdAt.Add(-time.Hour).UTC().Format(time.RFC3339)}}))
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "starts_at": {createdAt.Add(time.Hour).UTC().Format(time.RFC3339)}}))
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
		if !list.PageInfo.HasPrevPage || list.PageInfo.PreviousPageURL == nil {
			break
		}
		list, _, err = apiClient.GetListFromPageURL(list.PageInfo.PreviousPageURL)
		require.NoError(t, err)
	}
	assert.ElementsMatch(t, ids, backward, "walking back from the last page visits each row once")
}

// material_count is what the supplier's materials list returns: a link to a deleted item is not counted.
func TestSupplierParity_MaterialCountMatchesTheMaterialsList(t *testing.T) {
	t.Parallel()
	tok := searchToken("cnt")
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": "Counted " + tok, "number": uniqueName("SUP")}), "id")
	firstMaterial, _ := parityMaterial(t)
	secondMaterial, secondItem := parityMaterial(t)
	linkSupplierMaterial(t, supplierID, firstMaterial, nil)
	linkSupplierMaterial(t, supplierID, secondMaterial, nil)

	listedCount := func() string {
		row := listFindByField(t, suppliersPath, url.Values{"q": {tok}}, "id", supplierID)
		require.NotNil(t, row)
		return jsonField(parseJSON(row), "material_count")
	}
	assert.Equal(t, "2", jsonField(getSupplier(t, supplierID), "material_count"))
	assert.Equal(t, "2", listedCount())

	softDeleteItem(t, secondItem)

	assert.Equal(t, "1", jsonField(getSupplier(t, supplierID), "material_count"))
	assert.Equal(t, "1", listedCount())
	list, _, err := apiClient.GetList(supplierMaterialsOf(supplierID), nil)
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
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
	_, billChanged := changeForField(changes, "bill_to_address")
	assert.False(t, billChanged, "an address the update did not touch is not audited as changed: %v", changes)
}

func TestSupplierParity_UpdateNumberAndNote(t *testing.T) {
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

	status, got, body = patchSupplier(t, path, map[string]any{"note": "call first", "update_note": true})
	requireStatus(t, 200, status, body)
	assert.Equal(t, "call first", jsonField(got, "note"))

	status, got, body = patchSupplier(t, path, map[string]any{"note": "ignored"})
	requireStatus(t, 200, status, body)
	assert.Equal(t, "call first", jsonField(got, "note"), "a note without update_note is not applied")

	status, got, body = patchSupplier(t, path, map[string]any{"update_note": true})
	requireStatus(t, 200, status, body)
	assertNilField(t, got, "note")
	assert.Equal(t, newNumber, jsonField(got, "number"), "fields the update omitted are kept")
}

func TestSupplierParity_UpdateDefaultAddresses(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-defaults")
	id := jsonField(paritySupplier(t, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody(name, "6 First St"),
	}), "id")
	path := suppliersPath + "/" + id + "?include=bill_to_address,ship_to_address"
	original := jsonField(jsonObject(getSupplier(t, id, "ship_to_address"), "ship_to_address"), "id")
	saved := supplierAccountAddress(t, id, name+" Annex")

	status, got, body := patchSupplier(t, path, map[string]any{"bill_to_address_id": saved})
	requireStatus(t, 200, status, body)
	assert.Equal(t, saved, jsonField(jsonObject(got, "bill_to_address"), "id"), "the update returns its addresses when included")
	assert.Equal(t, original, jsonField(jsonObject(got, "ship_to_address"), "id"), "the other default is kept")

	status, got, body = patchSupplier(t, path, map[string]any{"ship_to_address_id": saved})
	requireStatus(t, 200, status, body)
	assert.Equal(t, saved, jsonField(jsonObject(got, "ship_to_address"), "id"))
	read := getSupplier(t, id, "bill_to_address", "ship_to_address")
	assert.Equal(t, saved, jsonField(jsonObject(read, "bill_to_address"), "id"))
	assert.Equal(t, saved, jsonField(jsonObject(read, "ship_to_address"), "id"))

	plainStatus, plain, plainBody := patchSupplier(t, suppliersPath+"/"+id, map[string]any{"note": "x", "update_note": true})
	requireStatus(t, 200, plainStatus, plainBody)
	assertNilField(t, plain, "bill_to_address")
	assertNilField(t, plain, "ship_to_address")
}

// A default address must be one of the supplier's own; another account's would be read back through the supplier.
func TestSupplierParity_UpdateRefusesAnAddressThatIsNotTheSuppliers(t *testing.T) {
	t.Parallel()
	id := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-sup-foreign"), "number": uniqueName("SUP")}), "id")
	path := suppliersPath + "/" + id

	ownersAddress := createE2EAddress(t, uniqueName("e2e-owner-addr"))
	status, _, body := patchSupplier(t, path, map[string]any{"bill_to_address_id": ownersAddress})
	requireErrorParam(t, 404, status, body, "bill_to_address_id")

	bStatus, bBody, err := getTenantBClient().Post(supplierAddressesPath, paritySupplierAddressBody(uniqueName("e2e-tenantb-addr"), "7 Other Co"), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, bStatus, bBody)
	tenantBAddress := jsonField(parseJSON(bBody), "id")
	t.Cleanup(func() { _, _, _ = getTenantBClient().Delete(supplierAddressesPath + "/" + tenantBAddress) })
	status, _, body = patchSupplier(t, path, map[string]any{"ship_to_address_id": tenantBAddress})
	requireErrorParam(t, 404, status, body, "ship_to_address_id")

	status, _, body = patchSupplier(t, path, map[string]any{"bill_to_address_id": mustGenID(t, "ad")})
	requireErrorParam(t, 404, status, body, "bill_to_address_id")

	read := getSupplier(t, id, "bill_to_address", "ship_to_address")
	assertNilField(t, read, "bill_to_address")
	assertNilField(t, read, "ship_to_address")
}

// --- Delete ---

func TestSupplierParity_DeleteReturnsTheSupplierThenIsGone(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sup-del")
	status, body, err := apiClient.Post(suppliersPath, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody(name, "8 Gone St"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	status, body, err = apiClient.Delete(suppliersPath + "/" + id + "?include=bill_to_address")
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	deleted := parseJSON(body)
	assert.Equal(t, id, jsonField(deleted, "id"))
	assert.Equal(t, name, jsonField(deleted, "name"))
	assert.NotEmpty(t, jsonField(jsonObject(deleted, "bill_to_address"), "id"), "the delete returns the addresses when included")
	assertNilField(t, deleted, "ship_to_address")

	getStatus, getBody, err := apiClient.GetListRaw(suppliersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus, string(getBody))

	againStatus, againBody, err := apiClient.Delete(suppliersPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, againStatus, "a second delete is gone, not a fresh success: %s", string(againBody))

	patchStatus, _, patchBody := patchSupplier(t, suppliersPath+"/"+id, map[string]any{"note": "x", "update_note": true})
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
	materialID, _ := parityMaterial(t)
	linkSupplierMaterial(t, id, materialID, nil)
	link := supplierMaterialsOf(id) + "/" + materialID

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

	status, body, err = clientB.GetListRaw(supplierMaterialsOf(id), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Empty(t, jsonArray(parseJSON(body), "data"), "tenant B sees none of tenant A's links")
	status, body, err = clientB.GetListRaw(link, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Patch(link, map[string]any{"supplier_part_number": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Delete(link)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = clientB.Post(supplierMaterialsOf(id), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "supplier_id")

	status, body, err = apiClient.GetListRaw(link, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestSupplierParity_PermissionRefusals(t *testing.T) {
	t.Parallel()
	id := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-sup-perm"), "number": uniqueName("SUP")}), "id")
	materialID, _ := parityMaterial(t)
	linkSupplierMaterial(t, id, materialID, nil)
	role := createAndCleanup(t, rolesPath, map[string]any{"name": uniqueName("e2e-sup-reader"), "permissions": []string{"suppliers:read"}})
	reader := roleScopedClient(t, jsonField(role, "id"))

	status, body, err := reader.GetListRaw(suppliersPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = reader.GetListRaw(suppliersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = reader.GetListRaw(supplierMaterialsOf(id)+"/"+materialID, nil)
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
		"link material": func() (int, []byte, error) {
			return reader.Post(supplierMaterialsOf(id), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
		},
		"update link": func() (int, []byte, error) {
			return reader.Patch(supplierMaterialsOf(id)+"/"+materialID, map[string]any{"supplier_part_number": "x"}, newIdempotencyKey())
		},
		"delete link": func() (int, []byte, error) { return reader.Delete(supplierMaterialsOf(id) + "/" + materialID) },
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

// --- Supplier materials ---

func TestSupplierMaterialParity_CRUD(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-crud"), "number": uniqueName("SUP")}), "id")
	materialID, _ := parityMaterial(t)
	partNumber := uniqueName("PN")
	path := supplierMaterialsOf(supplierID) + "/" + materialID

	created := linkSupplierMaterial(t, supplierID, materialID, map[string]any{
		"supplier_part_number": partNumber, "supplier_description": "Cone-wound", "is_active": false,
	})
	assert.Equal(t, materialID, jsonField(created, "id"), "a link is identified by its material")
	assertObjectField(t, created, "supplier_material")
	assert.Equal(t, partNumber, jsonField(created, "supplier_part_number"))
	assert.Equal(t, "Cone-wound", jsonField(created, "supplier_description"))
	assert.Equal(t, "inactive", jsonField(created, "status"))
	assertNilField(t, created, "material")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	got := parseJSON(mustGet(t, path))
	assert.Equal(t, partNumber, jsonField(got, "supplier_part_number"))

	status, updated, body := patchSupplier(t, path, map[string]any{"supplier_part_number": partNumber + "-B", "is_active": true})
	requireStatus(t, 200, status, body)
	assert.Equal(t, partNumber+"-B", jsonField(updated, "supplier_part_number"))
	assert.Equal(t, "active", jsonField(updated, "status"))
	assert.Equal(t, "Cone-wound", jsonField(updated, "supplier_description"), "an omitted description is kept")

	status, updated, body = patchSupplier(t, path, map[string]any{"supplier_description": nil})
	requireStatus(t, 200, status, body)
	assertNilField(t, updated, "supplier_description")
	assertNilField(t, parseJSON(mustGet(t, path)), "supplier_description")

	status, body, err := apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, materialID, jsonField(parseJSON(body), "id"))
	status, body, err = apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
}

func TestSupplierMaterialParity_DuplicateLinkConflicts(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-dup"), "number": uniqueName("SUP")}), "id")
	materialID, _ := parityMaterial(t)
	linkSupplierMaterial(t, supplierID, materialID, nil)

	status, body, err := apiClient.Post(supplierMaterialsOf(supplierID), map[string]any{"material_id": materialID, "supplier_part_number": "again"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 409, status, body, "material_id")
}

func TestSupplierMaterialParity_ListAndRetrieveIncludes(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-inc"), "number": uniqueName("SUP")}), "id")
	materialID, itemID := parityMaterial(t)
	linkSupplierMaterial(t, supplierID, materialID, nil)

	list, status, err := apiClient.GetList(supplierMaterialsOf(supplierID), url.Values{"include": {"material"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)
	row := parseJSON(list.Data[0])
	material := jsonObject(row, "material")
	require.NotNil(t, material)
	assert.Equal(t, materialID, jsonField(material, "id"))
	assertNilField(t, material, "item")

	got := parseJSON(mustGet(t, supplierMaterialsOf(supplierID)+"/"+materialID+"?include=material.item"))
	assert.Equal(t, itemID, jsonField(jsonObject(jsonObject(got, "material"), "item"), "id"))
}

// The column is TEXT, so a description runs to 65,535 bytes; the request counts characters, the service bytes.
func TestSupplierMaterialParity_LongDescriptions(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-long"), "number": uniqueName("SUP")}), "id")
	materialID, _ := parityMaterial(t)
	long := strings.Repeat("spec ", 1000)
	created := linkSupplierMaterial(t, supplierID, materialID, map[string]any{"supplier_description": long})
	assert.Equal(t, long, jsonField(created, "supplier_description"))
	path := supplierMaterialsOf(supplierID) + "/" + materialID

	full := strings.Repeat("a", 65535)
	status, got, body := patchSupplier(t, path, map[string]any{"supplier_description": full})
	requireStatus(t, 200, status, body)
	assert.Len(t, jsonField(got, "supplier_description"), 65535)

	status, _, body = patchSupplier(t, path, map[string]any{"supplier_description": full + "a"})
	requireErrorParam(t, 400, status, body, "supplier_description")

	status, _, body = patchSupplier(t, path, map[string]any{"supplier_description": strings.Repeat("🧶", 17000)})
	requireErrorParam(t, 400, status, body, "supplier_description")
	assert.Len(t, jsonField(parseJSON(mustGet(t, path)), "supplier_description"), 65535, "a refused update changes nothing")
}

func TestSupplierMaterialParity_RefusesEndsThatAreNotTheOwners(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-own"), "number": uniqueName("SUP")}), "id")
	materialID, _ := parityMaterial(t)

	status, body, err := apiClient.Post(supplierMaterialsOf(mustGenID(t, "ac")), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "supplier_id")

	status, body, err = apiClient.Post(supplierMaterialsOf(supplierID), map[string]any{"material_id": mustGenID(t, "ml"), "supplier_part_number": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "material_id")

	// Tenant B links tenant A's material to its own supplier.
	clientB := getTenantBClient()
	bStatus, bBody, err := clientB.Post(suppliersPath, map[string]any{"name": uniqueName("e2e-tenantb-sup"), "number": uniqueName("SUP")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, bStatus, bBody)
	tenantBSupplier := jsonField(parseJSON(bBody), "id")
	t.Cleanup(func() { _, _, _ = clientB.Delete(suppliersPath + "/" + tenantBSupplier) })
	status, body, err = clientB.Post(supplierMaterialsOf(tenantBSupplier), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "material_id")

	// Tenant A names tenant B's supplier.
	status, body, err = apiClient.Post(supplierMaterialsOf(tenantBSupplier), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "supplier_id")
}

func TestSupplierMaterialParity_DeletedItemLeavesTheLinkUnreachable(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-gone"), "number": uniqueName("SUP")}), "id")
	otherSupplier := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-gone2"), "number": uniqueName("SUP")}), "id")
	materialID, itemID := parityMaterial(t)
	linkSupplierMaterial(t, supplierID, materialID, nil)
	path := supplierMaterialsOf(supplierID) + "/" + materialID

	softDeleteItem(t, itemID)

	list, _, err := apiClient.GetList(supplierMaterialsOf(supplierID), nil)
	require.NoError(t, err)
	assert.Empty(t, list.Data, "a link to a deleted item is not listed")
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, string(body))
	status, _, body = patchSupplier(t, path, map[string]any{"supplier_part_number": "x"})
	assert.Equal(t, 404, status, string(body))

	status, body, err = apiClient.Post(supplierMaterialsOf(otherSupplier), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "material_id")
}

func TestSupplierMaterialParity_IsAudited(t *testing.T) {
	t.Parallel()
	supplierID := jsonField(paritySupplier(t, map[string]any{"name": uniqueName("e2e-supmat-audit"), "number": uniqueName("SUP")}), "id")
	materialID, _ := parityMaterial(t)
	partNumber := uniqueName("PN")
	linkSupplierMaterial(t, supplierID, materialID, map[string]any{"supplier_part_number": partNumber})
	rowID := supplierMaterialRowID(t, supplierID, materialID)
	path := supplierMaterialsOf(supplierID) + "/" + materialID

	created, ok := changeForField(jsonListData(expectAuditEventWithChanges(t, rowID, "supplier_material", "create"), "changes"), "supplier_part_number")
	require.True(t, ok)
	assert.Equal(t, partNumber, jsonField(created, "new_value"))

	status, _, body := patchSupplier(t, path, map[string]any{"supplier_part_number": partNumber + "-B"})
	requireStatus(t, 200, status, body)
	updated, ok := changeForField(jsonListData(expectAuditEventWithChanges(t, rowID, "supplier_material", "update"), "changes"), "supplier_part_number")
	require.True(t, ok)
	assert.Equal(t, partNumber+"-B", jsonField(updated, "new_value"))

	status, body, err := apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	expectAuditEventWithChanges(t, rowID, "supplier_material", "delete")
}
