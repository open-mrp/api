//go:build e2e

package api_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Suppliers, supplier materials, the team and counterparty contacts, account settings and user profiles,
// as the dashboard calls them: the portal's and other tenants' reach, state transitions, replays that must
// apply once, and what must stay unique under concurrent requests. Field-by-field CRUD lives in the
// crud_* files for each resource.

// dashSuppliersCall is one request a test expects a fixed status from.
type dashSuppliersCall struct {
	name string
	do   func() (int, []byte, error)
}

func dashSuppliersExpectAll(t *testing.T, want int, calls []dashSuppliersCall) {
	t.Helper()
	for _, c := range calls {
		status, body, err := c.do()
		require.NoError(t, err, c.name)
		assert.Equal(t, want, status, "%s: %s", c.name, body)
	}
}

// dashSuppliersOnce sends each request once, so a 5xx is reported instead of retried until it passes.
func dashSuppliersOnce(c *Client) *Client {
	clone := *c
	clone.retries = 0
	return &clone
}

// dashSuppliersRace sends n requests at once and returns each one's status and body.
func dashSuppliersRace(n int, send func() (int, []byte, error)) ([]int, [][]byte) {
	statuses, bodies := make([]int, n), make([][]byte, n)
	var start, done sync.WaitGroup
	start.Add(1)
	for i := 0; i < n; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			statuses[i], bodies[i], _ = send()
		}(i)
	}
	start.Done()
	done.Wait()
	return statuses, bodies
}

func dashSuppliersNew(t *testing.T, name string) string {
	t.Helper()
	return jsonField(createAndCleanup(t, suppliersPath, map[string]any{"name": name, "number": uniqueName("SUP")}), "id")
}

// dashSuppliersTeamUser adds a user to the seed seller under roleID and removes them at cleanup.
func dashSuppliersTeamUser(t *testing.T, roleID string) (accountUserID, name, email string) {
	t.Helper()
	name = uniqueName("e2e-dash-team")
	email = name + "@e2e-test.openmrp.ai"
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{"name": name, "email": email, "role_id": roleID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID = jsonField(parseJSON(body), "id")
	t.Cleanup(func() { removeAccountUser(accountUserID) })
	return accountUserID, name, email
}

func dashSuppliersMemberStatus(t *testing.T, c *Client, accountUserID string) string {
	t.Helper()
	status, body, err := c.GetListRaw(accountUsersPath+"/"+accountUserID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonField(parseJSON(body), "status")
}

func dashSuppliersMemberAction(c *Client, accountUserID, action string) func() (int, []byte, error) {
	return func() (int, []byte, error) {
		return c.Put(accountUsersPath+"/"+accountUserID+"/actions/"+action, nil)
	}
}

// dashSuppliersGetAnonymously sends a GET with no credentials and no account, as a signed-out browser does.
func dashSuppliersGetAnonymously(t *testing.T, path string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, apiClient.baseURL+path, nil)
	require.NoError(t, err)
	req.Header.Set("OpenMRP-Version", apiClient.apiVersion)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body
}

// --- Suppliers ---

// The customer portal acts in the seller's account but is not its staff, so the seller's suppliers and their
// material links are out of its reach, read or write.
func TestDashSuppliers_CustomerPortalCannotReachSuppliers(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-dash-sup-portal")
	id := dashSuppliersNew(t, name)
	materialID, _ := parityMaterial(t)
	partNumber := uniqueName("PN")
	linkSupplierMaterial(t, id, materialID, map[string]any{"supplier_part_number": partNumber})
	link := supplierMaterialsOf(id) + "/" + materialID
	portal := getCustomerPortalClient()

	dashSuppliersExpectAll(t, 403, []dashSuppliersCall{
		{"list suppliers", func() (int, []byte, error) { return portal.GetListRaw(suppliersPath, nil) }},
		{"retrieve supplier", func() (int, []byte, error) { return portal.GetListRaw(suppliersPath+"/"+id, nil) }},
		{"create supplier", func() (int, []byte, error) {
			return portal.Post(suppliersPath, map[string]any{"name": uniqueName("e2e-dash-portal"), "number": uniqueName("SUP")}, newIdempotencyKey())
		}},
		{"update supplier", func() (int, []byte, error) {
			return portal.Patch(suppliersPath+"/"+id, map[string]any{"name": "portal rename"}, newIdempotencyKey())
		}},
		{"delete supplier", func() (int, []byte, error) { return portal.Delete(suppliersPath + "/" + id) }},
		{"list links", func() (int, []byte, error) { return portal.GetListRaw(supplierMaterialsOf(id), nil) }},
		{"retrieve link", func() (int, []byte, error) { return portal.GetListRaw(link, nil) }},
		{"create link", func() (int, []byte, error) {
			return portal.Post(supplierMaterialsOf(id), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
		}},
		{"update link", func() (int, []byte, error) {
			return portal.Patch(link, map[string]any{"supplier_part_number": "portal"}, newIdempotencyKey())
		}},
		{"delete link", func() (int, []byte, error) { return portal.Delete(link) }},
	})

	assert.Equal(t, name, jsonField(getSupplier(t, id), "name"), "the supplier is untouched")
	assert.Equal(t, partNumber, jsonField(parseJSON(mustGet(t, link)), "supplier_part_number"), "the link is untouched")
}

// A number is unique within the account as MySQL's case-insensitive collation compares it; another tenant may
// use the same number, and deleting a supplier frees its number.
func TestDashSuppliers_NumberIsUniqueWithinTheAccount(t *testing.T) {
	t.Parallel()
	number := uniqueName("SUP-UNQ")
	first := jsonField(createAndCleanup(t, suppliersPath, map[string]any{"name": uniqueName("e2e-dash-unq"), "number": number}), "id")
	other := dashSuppliersNew(t, uniqueName("e2e-dash-unq-other"))

	status, body, err := apiClient.Post(suppliersPath, map[string]any{"name": uniqueName("e2e-dash-unq-dup"), "number": strings.ToLower(number)}, newIdempotencyKey())
	require.NoError(t, err)
	requireErrorParam(t, 409, status, body, "number")

	status, _, body = patchSupplier(t, suppliersPath+"/"+other, map[string]any{"number": strings.ToLower(number)})
	requireErrorParam(t, 409, status, body, "number")

	status, _, body = patchSupplier(t, suppliersPath+"/"+first, map[string]any{"number": number})
	requireStatus(t, 200, status, body)

	tenantB := getTenantBClient()
	status, body, err = tenantB.Post(suppliersPath, map[string]any{"name": uniqueName("e2e-dash-unq-b"), "number": number}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	tenantBSupplier := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = tenantB.Delete(suppliersPath + "/" + tenantBSupplier) })

	status, body, err = apiClient.Delete(suppliersPath + "/" + first)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	reused := createAndCleanup(t, suppliersPath, map[string]any{"name": uniqueName("e2e-dash-unq-reuse"), "number": number})
	assert.Equal(t, number, jsonField(reused, "number"), "a deleted supplier's number is free again")
}

// A double-submitted form or two people saving at once arrive as separate requests; one supplier takes the
// number and the rest conflict.
func TestDashSuppliers_ConcurrentCreatesKeepTheNumberUnique(t *testing.T) {
	t.Parallel()
	tok := searchToken("race")
	number := uniqueName("SUP-RACE")

	statuses, bodies := dashSuppliersRace(6, func() (int, []byte, error) {
		return dashSuppliersOnce(apiClient).Post(suppliersPath, map[string]any{"name": "Racer " + tok, "number": number}, newIdempotencyKey())
	})
	created := 0
	for i, status := range statuses {
		if status == 201 {
			created++
			id := jsonField(parseJSON(bodies[i]), "id")
			t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + id) })
			continue
		}
		assert.Equal(t, 409, status, "a create that loses the number conflicts: %s", bodies[i])
	}
	assert.Equal(t, 1, created, "exactly one create takes the number: %v", statuses)
	assert.Len(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}}), 1, "one supplier holds the number")
}

// A retried create returns the first response and creates nothing more: one supplier, one address each for
// bill-to and ship-to.
func TestDashSuppliers_CreateReplayMakesOneSupplierAndItsAddressesOnce(t *testing.T) {
	t.Parallel()
	tok := searchToken("replay")
	key := newIdempotencyKey()
	path := suppliersPath + "?include=bill_to_address,ship_to_address"
	body := map[string]any{
		"name": "Replayed " + tok, "number": uniqueName("SUP"),
		"bill_to_address": paritySupplierAddressBody("Replay Billing", "1 Replay St"),
		"ship_to_address": paritySupplierAddressBody("Replay Dock", "2 Replay Rd"),
	}

	first, err := apiClient.PostFull(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, first.StatusCode, first.Body)
	id := jsonField(parseJSON(first.Body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + id) })

	replay, err := apiClient.PostFull(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, replay.StatusCode, replay.Body)
	assert.Equal(t, parseJSON(first.Body), parseJSON(replay.Body), "the replay is the first response")
	assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))

	assert.Equal(t, []string{id}, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}}))
	var addresses int
	require.NoError(t, authDB(t).QueryRow("SELECT COUNT(*) FROM account_address WHERE account_id = ?", id).Scan(&addresses))
	assert.Equal(t, 2, addresses, "the bill-to and ship-to were each created once")

	body["name"] = "Changed " + tok
	status, resp, err := apiClient.Post(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, resp)
	requireErrorResponse(t, resp, "validation_failed", "idempotency_error")
}

// A replayed save answers from the first response without writing again, so it cannot undo a later edit.
func TestDashSuppliers_UpdateReplayDoesNotWriteAgain(t *testing.T) {
	t.Parallel()
	id := dashSuppliersNew(t, uniqueName("e2e-dash-sup-upd"))
	path := suppliersPath + "/" + id
	key := newIdempotencyKey()
	firstBody := map[string]any{"note": "first", "update_note": true}

	first, err := apiClient.PatchFull(path, firstBody, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	status, _, body := patchSupplier(t, path, map[string]any{"note": "second", "update_note": true})
	requireStatus(t, 200, status, body)

	replay, err := apiClient.PatchFull(path, firstBody, key)
	require.NoError(t, err)
	requireStatus(t, 200, replay.StatusCode, replay.Body)
	assert.Equal(t, parseJSON(first.Body), parseJSON(replay.Body), "the replay is the first response")
	assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))
	assert.Equal(t, "second", jsonField(getSupplier(t, id), "note"), "the replay did not write the first note again")

	status, resp, err := apiClient.Patch(path, map[string]any{"note": "third", "update_note": true}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, resp)
	requireErrorResponse(t, resp, "validation_failed", "idempotency_error")
	assert.Equal(t, "second", jsonField(getSupplier(t, id), "note"))
}

// Each invalid create is refused on the field at fault and creates nothing, inline addresses included.
func TestDashSuppliers_CreateRejectsInvalidFields(t *testing.T) {
	t.Parallel()
	tok := searchToken("inval")
	long := strings.Repeat("a", 256)
	address := func(overrides map[string]any) map[string]any {
		a := paritySupplierAddressBody("Invalid "+tok, "1 Invalid St")
		for k, v := range overrides {
			if v == nil {
				delete(a, k)
				continue
			}
			a[k] = v
		}
		return a
	}
	cases := []struct {
		name  string
		extra map[string]any
		param string
	}{
		{"name over 255", map[string]any{"name": long}, "name"},
		{"null name", map[string]any{"name": nil}, "name"},
		{"number over 255", map[string]any{"number": long}, "number"},
		{"blank number", map[string]any{"number": ""}, "number"},
		{"number not a string", map[string]any{"number": 42}, "number"},
		{"null note", map[string]any{"note": nil}, "note"},
		{"null bill-to", map[string]any{"bill_to_address": nil}, "bill_to_address"},
		{"bill-to without a country", map[string]any{"bill_to_address": address(map[string]any{"country": nil})}, "bill_to_address.country"},
		{"bill-to country over 2 letters", map[string]any{"bill_to_address": address(map[string]any{"country": "USA"})}, "bill_to_address.country"},
		{"bill-to without a name", map[string]any{"bill_to_address": address(map[string]any{"name": nil})}, "bill_to_address.name"},
		{"ship-to with a malformed email", map[string]any{"ship_to_address": address(map[string]any{"email": "not-an-email"})}, "ship_to_address.email"},
	}
	once := dashSuppliersOnce(apiClient)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"name": "Invalid " + tok, "number": uniqueName("SUP")}
			for k, v := range tc.extra {
				body[k] = v
			}
			status, resp, err := once.Post(suppliersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				id := jsonField(parseJSON(resp), "id")
				t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + id) })
			}
			requireStatus(t, 400, status, resp)
			assertErrorParam(t, requireErrorResponse(t, resp, "", "invalid_request_error"), tc.param)
		})
	}
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}}), "no refused create left a supplier")
}

// Each invalid update is refused on the field at fault, and a refused update changes nothing.
func TestDashSuppliers_UpdateRejectsInvalidFields(t *testing.T) {
	t.Parallel()
	name, number := uniqueName("e2e-dash-sup-inval"), uniqueName("SUP")
	id := jsonField(createAndCleanup(t, suppliersPath, map[string]any{"name": name, "number": number, "note": "kept"}), "id")
	long := strings.Repeat("a", 256)

	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"nothing to update", map[string]any{}, "validation_failed", ""},
		{"null name", map[string]any{"name": nil}, "invalid_format", "name"},
		{"blank name", map[string]any{"name": ""}, "invalid_format", "name"},
		{"name over 255", map[string]any{"name": long}, "", "name"},
		{"null number", map[string]any{"number": nil}, "invalid_format", "number"},
		{"blank number", map[string]any{"number": ""}, "invalid_format", "number"},
		{"number over 255", map[string]any{"number": long}, "", "number"},
		{"blank note", map[string]any{"note": "", "update_note": true}, "invalid_format", "note"},
		{"update_note not a boolean", map[string]any{"update_note": "yes"}, "invalid_format", "update_note"},
		{"null bill-to id", map[string]any{"bill_to_address_id": nil}, "invalid_format", "bill_to_address_id"},
		{"blank ship-to id", map[string]any{"ship_to_address_id": ""}, "invalid_format", "ship_to_address_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, resp, err := apiClient.Patch(suppliersPath+"/"+id, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, resp)
			errObj := requireErrorResponse(t, resp, tc.code, "invalid_request_error")
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
		})
	}
	got := getSupplier(t, id)
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, number, jsonField(got, "number"))
	assert.Equal(t, "kept", jsonField(got, "note"))
}

func TestDashSuppliers_ListRejectsMalformedDates(t *testing.T) {
	t.Parallel()
	for param, value := range map[string]string{"starts_at": "yesterday", "ends_at": "2026-13-45"} {
		status, body, err := apiClient.GetListRaw(suppliersPath, url.Values{param: {value}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error"), param)
	}
}

// Another tenant gets the same 404 for a supplier this account deleted as for one that never existed; a
// different answer would tell it the ID was once a supplier somewhere.
func TestDashSuppliers_AnotherTenantCannotTellADeletedSupplierExisted(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(suppliersPath, map[string]any{"name": uniqueName("e2e-dash-sup-gone"), "number": uniqueName("SUP")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	status, body, err = apiClient.Delete(suppliersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	tenantB := getTenantBClient()
	dashSuppliersExpectAll(t, 404, []dashSuppliersCall{
		{"retrieve", func() (int, []byte, error) { return tenantB.GetListRaw(suppliersPath+"/"+id, nil) }},
		{"update", func() (int, []byte, error) {
			return tenantB.Patch(suppliersPath+"/"+id, map[string]any{"name": "x"}, newIdempotencyKey())
		}},
		{"delete", func() (int, []byte, error) { return tenantB.Delete(suppliersPath + "/" + id) }},
	})
}

// --- Supplier materials ---

// A retried link returns the first response rather than conflicting with the link it made.
func TestDashSuppliers_MaterialLinkReplayLinksOnce(t *testing.T) {
	t.Parallel()
	supplierID := dashSuppliersNew(t, uniqueName("e2e-dash-link-replay"))
	materialID, _ := parityMaterial(t)
	key := newIdempotencyKey()
	body := map[string]any{"material_id": materialID, "supplier_part_number": uniqueName("PN")}

	first, err := apiClient.PostFull(supplierMaterialsOf(supplierID), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, first.StatusCode, first.Body)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(supplierMaterialsOf(supplierID) + "/" + materialID) })

	replay, err := apiClient.PostFull(supplierMaterialsOf(supplierID), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, replay.StatusCode, replay.Body)
	assert.Equal(t, parseJSON(first.Body), parseJSON(replay.Body), "the replay is the first response")
	assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))

	list, _, err := apiClient.GetList(supplierMaterialsOf(supplierID), nil)
	require.NoError(t, err)
	assert.Len(t, list.Data, 1, "one link")

	body["supplier_part_number"] = uniqueName("PN")
	status, resp, err := apiClient.Post(supplierMaterialsOf(supplierID), body, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, resp)
	requireErrorResponse(t, resp, "validation_failed", "idempotency_error")
}

func TestDashSuppliers_MaterialUpdateReplayDoesNotWriteAgain(t *testing.T) {
	t.Parallel()
	supplierID := dashSuppliersNew(t, uniqueName("e2e-dash-link-upd"))
	materialID, _ := parityMaterial(t)
	linkSupplierMaterial(t, supplierID, materialID, nil)
	path := supplierMaterialsOf(supplierID) + "/" + materialID
	key := newIdempotencyKey()
	firstBody := map[string]any{"supplier_part_number": "PN-FIRST"}

	first, err := apiClient.PatchFull(path, firstBody, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	status, _, body := patchSupplier(t, path, map[string]any{"supplier_part_number": "PN-SECOND"})
	requireStatus(t, 200, status, body)

	replay, err := apiClient.PatchFull(path, firstBody, key)
	require.NoError(t, err)
	requireStatus(t, 200, replay.StatusCode, replay.Body)
	assert.Equal(t, parseJSON(first.Body), parseJSON(replay.Body), "the replay is the first response")
	assert.Equal(t, "PN-SECOND", jsonField(parseJSON(mustGet(t, path)), "supplier_part_number"), "the replay did not write again")
}

func TestDashSuppliers_MaterialRejectsInvalidFields(t *testing.T) {
	t.Parallel()
	supplierID := dashSuppliersNew(t, uniqueName("e2e-dash-link-inval"))
	linkedMaterial, _ := parityMaterial(t)
	partNumber := uniqueName("PN")
	linkSupplierMaterial(t, supplierID, linkedMaterial, map[string]any{"supplier_part_number": partNumber, "supplier_description": "kept"})
	freeMaterial, _ := parityMaterial(t)
	long := strings.Repeat("a", 256)

	creates := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"missing material", map[string]any{"supplier_part_number": "x"}, "material_id"},
		{"missing part number", map[string]any{"material_id": freeMaterial}, "supplier_part_number"},
		{"blank part number", map[string]any{"material_id": freeMaterial, "supplier_part_number": ""}, "supplier_part_number"},
		{"part number over 255", map[string]any{"material_id": freeMaterial, "supplier_part_number": long}, "supplier_part_number"},
		{"null description", map[string]any{"material_id": freeMaterial, "supplier_part_number": "x", "supplier_description": nil}, "supplier_description"},
		{"null is_active", map[string]any{"material_id": freeMaterial, "supplier_part_number": "x", "is_active": nil}, "is_active"},
		{"is_active not a boolean", map[string]any{"material_id": freeMaterial, "supplier_part_number": "x", "is_active": "yes"}, "is_active"},
	}
	for _, tc := range creates {
		t.Run("create "+tc.name, func(t *testing.T) {
			status, resp, err := apiClient.Post(supplierMaterialsOf(supplierID), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, resp)
			assertErrorParam(t, requireErrorResponse(t, resp, "", "invalid_request_error"), tc.param)
		})
	}

	path := supplierMaterialsOf(supplierID) + "/" + linkedMaterial
	updates := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"nothing to update", map[string]any{}, ""},
		{"null part number", map[string]any{"supplier_part_number": nil}, "supplier_part_number"},
		{"blank part number", map[string]any{"supplier_part_number": ""}, "supplier_part_number"},
		{"part number over 255", map[string]any{"supplier_part_number": long}, "supplier_part_number"},
		{"null is_active", map[string]any{"is_active": nil}, "is_active"},
	}
	for _, tc := range updates {
		t.Run("update "+tc.name, func(t *testing.T) {
			status, resp, err := apiClient.Patch(path, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, resp)
			errObj := requireErrorResponse(t, resp, "", "invalid_request_error")
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
		})
	}

	list, _, err := apiClient.GetList(supplierMaterialsOf(supplierID), nil)
	require.NoError(t, err)
	assert.Len(t, list.Data, 1, "no refused create linked a material")
	got := parseJSON(mustGet(t, path))
	assert.Equal(t, partNumber, jsonField(got, "supplier_part_number"))
	assert.Equal(t, "kept", jsonField(got, "supplier_description"))
	assert.Equal(t, "active", jsonField(got, "status"))
}

// Linking one material twice at once must end in one link and a conflict, not a server error.
func TestDashSuppliers_ConcurrentLinksOfOneMaterialMakeOneLink(t *testing.T) {
	t.Parallel()
	supplierID := dashSuppliersNew(t, uniqueName("e2e-dash-link-race"))
	materialID, _ := parityMaterial(t)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(supplierMaterialsOf(supplierID) + "/" + materialID) })

	statuses, bodies := dashSuppliersRace(6, func() (int, []byte, error) {
		return dashSuppliersOnce(apiClient).Post(supplierMaterialsOf(supplierID), map[string]any{"material_id": materialID, "supplier_part_number": uniqueName("PN")}, newIdempotencyKey())
	})
	created := 0
	for i, status := range statuses {
		if status == 201 {
			created++
			continue
		}
		assert.Equal(t, 409, status, "a link that loses conflicts: %s", bodies[i])
	}
	assert.Equal(t, 1, created, "exactly one link is made: %v", statuses)
	list, _, err := apiClient.GetList(supplierMaterialsOf(supplierID), nil)
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

// Deleting a supplier takes its material links with it, and a deleted link can be made again.
func TestDashSuppliers_LinksFollowTheirSupplierAndCanBeRemade(t *testing.T) {
	t.Parallel()
	supplierID := dashSuppliersNew(t, uniqueName("e2e-dash-link-life"))
	materialID, _ := parityMaterial(t)
	linkSupplierMaterial(t, supplierID, materialID, nil)
	path := supplierMaterialsOf(supplierID) + "/" + materialID

	status, body, err := apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	remade := linkSupplierMaterial(t, supplierID, materialID, map[string]any{"supplier_part_number": "PN-AGAIN"})
	assert.Equal(t, "PN-AGAIN", jsonField(remade, "supplier_part_number"))
	assert.Equal(t, "active", jsonField(remade, "status"))

	status, body, err = apiClient.Delete(suppliersPath + "/" + supplierID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	dashSuppliersExpectAll(t, 404, []dashSuppliersCall{
		{"retrieve link", func() (int, []byte, error) { return apiClient.GetListRaw(path, nil) }},
		{"update link", func() (int, []byte, error) {
			return apiClient.Patch(path, map[string]any{"supplier_part_number": "x"}, newIdempotencyKey())
		}},
		{"delete link", func() (int, []byte, error) { return apiClient.Delete(path) }},
		{"create link", func() (int, []byte, error) {
			return apiClient.Post(supplierMaterialsOf(supplierID), map[string]any{"material_id": materialID, "supplier_part_number": "x"}, newIdempotencyKey())
		}},
	})
	list, _, err := apiClient.GetList(supplierMaterialsOf(supplierID), nil)
	require.NoError(t, err)
	assert.Empty(t, list.Data, "a deleted supplier lists no links")
}

// --- Team and contacts ---

// The customer portal acts in the seller's account but is not on its team, so it can neither see the team nor
// add, change or change the status of anyone on it.
func TestDashAccounts_CustomerPortalCannotReachTheSellersTeam(t *testing.T) {
	t.Parallel()
	id, name, _ := dashSuppliersTeamUser(t, SeedSalesRepRoleID)
	portal := getCustomerPortalClient()

	dashSuppliersExpectAll(t, 403, []dashSuppliersCall{
		{"list", func() (int, []byte, error) { return portal.GetListRaw(accountUsersPath, nil) }},
		{"retrieve", func() (int, []byte, error) { return portal.GetListRaw(accountUsersPath+"/"+id, nil) }},
		{"create", func() (int, []byte, error) {
			return portal.Post(accountUsersPath, map[string]any{"email": uniqueName("e2e-dash-portal") + "@e2e-test.openmrp.ai"}, newIdempotencyKey())
		}},
		{"update", func() (int, []byte, error) {
			return portal.Patch(accountUsersPath+"/"+id, map[string]any{"name": "portal rename"}, newIdempotencyKey())
		}},
		{"disable", dashSuppliersMemberAction(portal, id, "disable")},
		{"activate", dashSuppliersMemberAction(portal, id, "activate")},
		{"remove", dashSuppliersMemberAction(portal, id, "remove")},
	})

	got := getContact(t, apiClient, id)
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.Equal(t, name, jsonField(jsonObject(got, "user"), "name"))
}

// Each action takes a member to its state from any other, repeating one is a no-op, and a removed member must be
// restored before it can be locked.
func TestDashAccounts_TeamMemberStatusTransitions(t *testing.T) {
	t.Parallel()
	id, name, _ := dashSuppliersTeamUser(t, SeedSalesRepRoleID)

	for i, step := range []struct {
		action string
		want   int
		status string
	}{
		{"disable", 200, "disabled"},
		{"disable", 200, "disabled"},
		{"remove", 200, "removed"},
		{"remove", 200, "removed"},
		{"disable", 400, "removed"},
		{"activate", 200, "active"},
		{"activate", 200, "active"},
		{"disable", 200, "disabled"},
		{"activate", 200, "active"},
		{"remove", 200, "removed"},
	} {
		status, body, err := dashSuppliersMemberAction(apiClient, id, step.action)()
		require.NoError(t, err)
		require.Equal(t, step.want, status, "step %d (%s): %s", i, step.action, body)
		require.Equal(t, step.status, dashSuppliersMemberStatus(t, apiClient, id), "after step %d (%s)", i, step.action)
	}

	assert.Nil(t, findContact(listContacts(t, apiClient, url.Values{"q": {name}}), id), "a removed member is not listed")
	removed := findContact(listContacts(t, apiClient, url.Values{"q": {name}, "removed_scope": {"included"}}), id)
	require.NotNil(t, removed, "a removed member is listed when asked for")
	assert.Equal(t, "removed", jsonField(removed, "status"))
}

// Another tenant gets 404 for every route on this account's member, before and after removal, and its list
// does not show them.
func TestDashAccounts_AnotherTenantGets404ForTheTeam(t *testing.T) {
	t.Parallel()
	id, name, _ := dashSuppliersTeamUser(t, SeedSalesRepRoleID)
	tenantB := getTenantBClient()
	calls := []dashSuppliersCall{
		{"retrieve", func() (int, []byte, error) { return tenantB.GetListRaw(accountUsersPath+"/"+id, nil) }},
		{"update", func() (int, []byte, error) {
			return tenantB.Patch(accountUsersPath+"/"+id, map[string]any{"name": "taken over"}, newIdempotencyKey())
		}},
		{"disable", dashSuppliersMemberAction(tenantB, id, "disable")},
		{"activate", dashSuppliersMemberAction(tenantB, id, "activate")},
		{"remove", dashSuppliersMemberAction(tenantB, id, "remove")},
	}

	dashSuppliersExpectAll(t, 404, calls)
	assert.Empty(t, listContacts(t, tenantB, url.Values{"q": {name}, "removed_scope": {"included"}}))
	assert.Equal(t, "active", dashSuppliersMemberStatus(t, apiClient, id))

	removeAccountUser(id)
	require.Equal(t, "removed", dashSuppliersMemberStatus(t, apiClient, id))
	dashSuppliersExpectAll(t, 404, calls)
	assert.Equal(t, "removed", dashSuppliersMemberStatus(t, apiClient, id), "tenant B restored nothing")
}

// A retried invite returns the first response and sends one welcome email.
func TestDashAccounts_CreateReplaySendsOneWelcomeEmail(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	name := uniqueName("e2e-dash-invite")
	email := name + "@e2e-test.openmrp.ai"
	key := newIdempotencyKey()
	body := map[string]any{"name": name, "email": email, "role_id": SeedSalesRepRoleID}

	first, err := apiClient.PostFull(accountUsersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, first.StatusCode, first.Body)
	id := jsonField(parseJSON(first.Body), "id")
	t.Cleanup(func() { removeAccountUser(id) })

	replay, err := apiClient.PostFull(accountUsersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, replay.StatusCode, replay.Body)
	assert.Equal(t, parseJSON(first.Body), parseJSON(replay.Body), "the replay is the first response")
	assert.Equal(t, "true", replay.Header.Get("Idempotent-Replayed"))
	assert.Equal(t, "new_user_welcome", queuedSendEmail(t, email)["template_id"], "exactly one welcome email")

	body["name"] = uniqueName("e2e-dash-invite-changed")
	status, resp, err := apiClient.Post(accountUsersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, resp)
	requireErrorResponse(t, resp, "validation_failed", "idempotency_error")
}

// An email or username belongs to one user, so an update may not take one another user holds; keeping one's own
// is not a conflict.
func TestDashAccounts_UpdateRefusesAnEmailOrUsernameInUse(t *testing.T) {
	t.Parallel()
	_, _, takenEmail := dashSuppliersTeamUser(t, SeedSalesRepRoleID)
	id, _, ownEmail := dashSuppliersTeamUser(t, SeedSalesRepRoleID)
	takenUsername := uniqueName("e2e-dash-scn")
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{"username": takenUsername, "password": "ScannerPass123!"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	scannerID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { removeAccountUser(scannerID) })

	for param, value := range map[string]string{"email": takenEmail, "username": takenUsername} {
		status, body, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{param: value}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "resource_conflict", "invalid_request_error"), param)
	}
	updated := patchContact(t, apiClient, id, map[string]any{"email": ownEmail})
	assert.Equal(t, ownEmail, jsonField(jsonObject(updated, "user"), "email"))
	assertNilField(t, jsonObject(updated, "user"), "username")
}

func TestDashAccounts_CreateAndUpdateRejectInvalidFields(t *testing.T) {
	t.Parallel()
	id, name, _ := dashSuppliersTeamUser(t, SeedSalesRepRoleID)

	creates := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"neither email nor username", map[string]any{"name": uniqueName("e2e-dash-noid")}, "validation_failed", ""},
		{"null email", map[string]any{"email": nil}, "invalid_format", "email"},
		{"null name", map[string]any{"name": nil, "email": uniqueName("e2e-dash-inval") + "@e2e-test.openmrp.ai"}, "invalid_format", "name"},
		{"name over 255", map[string]any{"name": strings.Repeat("a", 256), "email": uniqueName("e2e-dash-inval") + "@e2e-test.openmrp.ai"}, "", "name"},
		{"commission flag not a boolean", map[string]any{"email": uniqueName("e2e-dash-inval") + "@e2e-test.openmrp.ai", "is_commission_eligible": "yes"}, "invalid_format", "is_commission_eligible"},
	}
	for _, tc := range creates {
		t.Run("create "+tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(accountUsersPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				created := jsonField(parseJSON(body), "id")
				t.Cleanup(func() { removeAccountUser(created) })
			}
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, tc.code, "invalid_request_error")
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
		})
	}

	updates := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"nothing to update", map[string]any{}, "validation_failed", ""},
		{"null email", map[string]any{"email": nil}, "invalid_format", "email"},
		{"malformed email", map[string]any{"email": "not-an-email"}, "", "email"},
		{"blank name", map[string]any{"name": ""}, "invalid_format", "name"},
		{"username too short", map[string]any{"username": "ab"}, "", "username"},
		{"null commission flag", map[string]any{"is_commission_eligible": nil}, "invalid_format", "is_commission_eligible"},
	}
	for _, tc := range updates {
		t.Run("update "+tc.name, func(t *testing.T) {
			status, body, err := apiClient.Patch(accountUsersPath+"/"+id, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, tc.code, "invalid_request_error")
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
		})
	}
	assert.Equal(t, name, jsonField(jsonObject(getContact(t, apiClient, id), "user"), "name"), "refused updates change nothing")
}

// A member may edit their own membership without the team permission, but not their own role: a role that
// reaches this endpoint through customers:update must not let its holder make themselves an admin. Without
// team:update they cannot change anyone else either.
func TestDashAccounts_AMemberWithoutTheTeamPermissionCannotChangeTheirOwnRole(t *testing.T) {
	t.Parallel()
	role := createAndCleanup(t, rolesPath, map[string]any{
		"name":        uniqueName("e2e-dash-cust-mgr"),
		"permissions": []string{"customers:read", "customers:update"},
	})
	roleID := jsonField(role, "id")
	member := newSeedAccountUser(t, roleID)
	session := member.session(t)
	other, otherName, _ := dashSuppliersTeamUser(t, SeedSalesRepRoleID)

	status, body, err := session.Patch(accountUsersPath+"/"+other, map[string]any{"name": uniqueName("e2e-dash-hijack")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	assert.Equal(t, otherName, jsonField(jsonObject(getContact(t, apiClient, other), "user"), "name"))

	status, body, err = session.Patch(accountUsersPath+"/"+member.accountUserID, map[string]any{"role_id": SeedAdminRoleID}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{400, 403}, status, "a member must not grant themselves the admin role: %s", body)
	status, body, err = apiClient.GetListRaw(accountUsersPath+"/"+member.accountUserID, url.Values{"include": {"role"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, roleID, jsonField(jsonObject(parseJSON(body), "role"), "id"), "the member's role is unchanged")
}

// A member's edit of their own membership either saves and answers with the saved record, or is refused and
// saves nothing; answering a refusal for an edit that was saved leaves the dashboard showing stale data.
func TestDashAccounts_ASelfEditAnswersWithWhatItSaved(t *testing.T) {
	t.Parallel()
	role := createAndCleanup(t, rolesPath, map[string]any{
		"name":        uniqueName("e2e-dash-self-edit"),
		"permissions": []string{"customers:read", "customers:update"},
	})
	member := newSeedAccountUser(t, jsonField(role, "id"))
	name := uniqueName("E2E Self Edit")

	status, body, err := member.session(t).Patch(accountUsersPath+"/"+member.accountUserID, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	saved := jsonField(jsonObject(getContact(t, apiClient, member.accountUserID), "user"), "name")
	if status == 200 {
		assert.Equal(t, name, saved, "an accepted edit is saved")
		return
	}
	assert.Equal(t, "E2E User", saved, "a refused edit (%d) must not be saved: %s", status, body)
}

// Adding a contact with create but not read either adds them and answers with the contact, or is refused and
// adds no one.
func TestDashAccounts_AContactCreateAnswersWithWhatItSaved(t *testing.T) {
	t.Parallel()
	supplierID := createContactsSupplier(t)
	creator := contactsRoleClient(t, supplierID, "suppliers:create")
	name, email := contactEmail("e2e-dash-create-only")

	status, body, err := creator.Post(accountUsersPath, map[string]any{"name": name, "email": email}, newIdempotencyKey())
	require.NoError(t, err)
	added := contactIDs(listContacts(t, apiClient.WithAccountID(supplierID), url.Values{"q": {name}}))
	if status == 201 {
		assert.Equal(t, []string{jsonField(parseJSON(body), "id")}, added, "an accepted create added the contact")
		return
	}
	assert.Empty(t, added, "a refused create (%d) must not add the contact: %s", status, body)
}

// Locking and unlocking a supplier's contact takes suppliers:update: read alone or the customer permissions do
// not, and update does not stretch to removal.
func TestDashAccounts_SupplierContactStatusFollowsTheSupplierPermission(t *testing.T) {
	t.Parallel()
	supplierID := createContactsSupplier(t)
	name, email := contactEmail("e2e-dash-contact-lock")
	contactID := jsonField(createContact(t, apiClient.WithAccountID(supplierID), name, email, nil), "id")
	seller := apiClient.WithAccountID(supplierID)

	reader := contactsRoleClient(t, supplierID, "suppliers:read")
	customerManager := contactsRoleClient(t, supplierID, "customers:read", "customers:update", "customers:delete")
	for label, c := range map[string]*Client{"suppliers:read": reader, "customer permissions": customerManager} {
		for _, action := range []string{"disable", "remove"} {
			status, body, err := dashSuppliersMemberAction(c, contactID, action)()
			require.NoError(t, err)
			assert.Equal(t, 403, status, "%s may not %s: %s", label, action, body)
		}
	}
	assert.Equal(t, "active", dashSuppliersMemberStatus(t, seller, contactID))

	updater := contactsRoleClient(t, supplierID, "suppliers:read", "suppliers:update")
	for _, step := range []struct{ action, status string }{
		{"disable", "disabled"}, {"disable", "disabled"}, {"activate", "active"},
	} {
		status, body, err := dashSuppliersMemberAction(updater, contactID, step.action)()
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Equal(t, step.status, dashSuppliersMemberStatus(t, seller, contactID), step.action)
	}
	status, body, err := dashSuppliersMemberAction(updater, contactID, "remove")()
	require.NoError(t, err)
	assert.Equal(t, 403, status, "removal takes suppliers:delete: %s", body)
	assert.Equal(t, "active", dashSuppliersMemberStatus(t, seller, contactID))
}

// --- Account settings ---

// The portal may read the seller's favicon, but the account record and its settings are the seller's staff's
// alone.
func TestDashAccounts_CustomerPortalCannotReadOrChangeTheSellerAccount(t *testing.T) {
	t.Parallel()
	before := accountValues(readAccount(t, apiClient, SeedAccountID))
	portal := getCustomerPortalClient()
	accountPath := accountsPath + "/" + SeedAccountID

	dashSuppliersExpectAll(t, 403, []dashSuppliersCall{
		{"retrieve", func() (int, []byte, error) { return portal.GetListRaw(accountPath, nil) }},
		{"update", func() (int, []byte, error) {
			return portal.Patch(accountPath, map[string]any{"name": before["name"]}, newIdempotencyKey())
		}},
		{"upload logo", func() (int, []byte, error) { return portal.PutBytes(accountPath+"/photo", "image/png", onePixelPNG(t)) }},
		{"upload favicon", func() (int, []byte, error) {
			return portal.PutBytes(accountPath+"/favicon", "image/png", onePixelPNG(t))
		}},
	})

	status, body, err := portal.GetListRaw(accountPath+"/favicon", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, before, accountValues(readAccount(t, apiClient, SeedAccountID)), "the account is untouched")
}

// A role that may read the account settings may not save them or replace the logo or favicon.
func TestDashAccounts_AReadOnlyRoleCannotChangeTheAccount(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "self:read")
	accountPath := accountsPath + "/" + SeedAccountID
	status, body, err := reader.GetListRaw(accountPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, SeedAccountID, jsonField(got, "id"))

	for name, call := range map[string]func() (int, []byte, error){
		"update": func() (int, []byte, error) {
			return reader.Patch(accountPath, map[string]any{"name": jsonField(got, "name")}, newIdempotencyKey())
		},
		"upload logo": func() (int, []byte, error) { return reader.PutBytes(accountPath+"/photo", "image/png", onePixelPNG(t)) },
		"upload favicon": func() (int, []byte, error) {
			return reader.PutBytes(accountPath+"/favicon", "image/png", onePixelPNG(t))
		},
	} {
		status, body, err := call()
		require.NoError(t, err)
		require.Equal(t, 403, status, "%s: %s", name, body)
		requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	}
}

// An upload's format is read from its bytes: an ICO is a favicon but not a logo, an SVG is neither, a PNG is
// taken whatever Content-Type names it, and anything over 10 MB is refused as too large.
func TestDashAccounts_UploadsJudgeTheImageByItsBytes(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	ico := append([]byte{0, 0, 1, 0, 1, 0, 1, 1, 0, 0, 1, 0, 32, 0, 48, 0, 0, 0, 22, 0, 0, 0}, make([]byte, 48)...)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)

	for _, tc := range []struct {
		name, asset, contentType string
		body                     []byte
		want                     int
		code                     string
	}{
		{"ico as a logo", "photo", "image/x-icon", ico, 400, "invalid_format"},
		{"svg as a logo", "photo", "image/svg+xml", svg, 400, "invalid_format"},
		{"svg as a favicon", "favicon", "image/svg+xml", svg, 400, "invalid_format"},
		{"oversized logo", "photo", "image/png", oversizedPNG(t), 413, "request_too_large"},
		{"oversized favicon", "favicon", "image/png", oversizedPNG(t), 413, "request_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := acct.owner.PutBytes(acct.path()+"/"+tc.asset, tc.contentType, tc.body)
			require.NoError(t, err)
			requireStatus(t, tc.want, status, body)
			requireErrorResponse(t, body, tc.code, "invalid_request_error")
		})
	}
	branding := jsonObject(readAccount(t, acct.owner, acct.id), "branding")
	assertNilField(t, branding, "logo_url")
	assertNilField(t, branding, "favicon_url")

	status, body, err := acct.owner.PutBytes(acct.path()+"/favicon", "image/x-icon", ico)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = acct.owner.PutBytes(acct.path()+"/photo", "text/plain", onePixelPNG(t))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	branding = jsonObject(readAccount(t, acct.owner, acct.id), "branding")
	assert.NotEmpty(t, jsonField(branding, "favicon_url"), "the ICO favicon is stored")
	assert.NotEmpty(t, jsonField(branding, "logo_url"), "the PNG logo is stored")
}

// Only the account acted in may change its own logo and favicon: not another tenant's, nor one of its own
// customers'.
func TestDashAccounts_UploadsToAnotherAccountAreRefused(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()
	logoOf := func() any {
		status, body, err := tenantB.GetListRaw(accountsPath+"/"+SeedTenantBAccountID+"/logo", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		if u, ok := parseJSON(body)["url"].(string); ok {
			return signedObjectPath(t, u)
		}
		return nil
	}
	before := logoOf()

	for _, accountID := range []string{SeedTenantBAccountID, SeedCustomerAccountID} {
		for _, asset := range []string{"photo", "favicon"} {
			status, body, err := apiClient.PutBytes(accountsPath+"/"+accountID+"/"+asset, "image/png", onePixelPNG(t))
			require.NoError(t, err)
			assert.Equal(t, 403, status, "%s upload to %s: %s", asset, accountID, body)
		}
	}
	assert.Equal(t, before, logoOf(), "tenant B's logo is untouched")
}

// The portal's sign-in page reads branding before anyone signs in, so it answers anonymously, finds the slug
// whatever its case, and carries only the public profile.
func TestDashAccounts_BrandingIsPublicAndCarriesOnlyThePublicProfile(t *testing.T) {
	t.Parallel()
	account := readAccount(t, apiClient, SeedAccountID)

	for _, slug := range []string{SeedAccountSlug, strings.ToUpper(SeedAccountSlug)} {
		status, body := dashSuppliersGetAnonymously(t, "/v1/settings/branding/"+slug)
		requireStatus(t, 200, status, body)
		got := parseJSON(body)
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		assert.ElementsMatch(t, []string{
			"id", "object", "name", "slug", "default_billing_address", "support_email", "logo_url", "portal_domain", "favicon_url",
		}, keys, "the public profile and nothing more")
		assert.Equal(t, SeedAccountID, jsonField(got, "id"))
		assertObjectField(t, got, "public_account")
		assert.Equal(t, jsonField(account, "name"), jsonField(got, "name"))
		assert.Equal(t, SeedAccountSlug, jsonField(got, "slug"))
		assert.Equal(t, jsonField(jsonObject(account, "branding"), "support_email"), jsonField(got, "support_email"))
		assertNilField(t, got, "default_billing_address")
	}
}

// The signed-in portal prints the seller's letterhead from this profile; it needs a signed-in caller.
func TestDashAccounts_PortalProfileIsTheSellersLetterhead(t *testing.T) {
	t.Parallel()
	status, body := dashSuppliersGetAnonymously(t, "/v1/settings/portal-profiles/"+SeedAccountSlug)
	requireStatus(t, 401, status, body)

	account := readAccount(t, apiClient, SeedAccountID)
	status, body, err := getCustomerPortalClient().GetListRaw("/v1/settings/portal-profiles/"+SeedAccountSlug, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, SeedAccountID, jsonField(got, "id"))
	assertObjectField(t, got, "portal_profile")
	assert.Equal(t, jsonField(account, "name"), jsonField(got, "name"))
	assert.Equal(t, SeedAccountSlug, jsonField(got, "slug"))
	assert.Equal(t, jsonField(jsonObject(account, "branding"), "support_email"), jsonField(got, "support_email"))
	for _, f := range []string{"logo_url", "favicon_url"} {
		assert.Contains(t, got, f)
	}
	address := jsonObject(got, "address")
	require.NotNil(t, address, "the letterhead carries the seller's billing address: %s", body)
	assertObjectField(t, address, "address")
	assert.Equal(t, jsonField(jsonObject(account, "default_billing_address"), "id"), jsonField(address, "id"))
	assertObjectField(t, jsonObject(address, "geolocation"), "geolocation")
}

// --- User profiles ---

// Someone else's profile takes team:update in an account the user belongs to: team:read is not enough, nor is
// being the customer portal, nor being another member without it.
func TestDashAccounts_EditingAnotherUsersProfileTakesTheTeamPermission(t *testing.T) {
	t.Parallel()
	target := newSeedAccountUser(t, SeedSalesRepRoleID)
	before := userValues(readUser(t, apiClient, target.id))
	peer := newSeedAccountUser(t, SeedSalesRepRoleID).session(t)

	refused := map[string]*Client{
		"team:read":       customRoleClient(t, "team:read"),
		"customer portal": getCustomerPortalClient(),
		"a peer":          peer,
	}
	for label, c := range refused {
		status, body, err := c.Patch(target.path(), map[string]any{"name": uniqueName("E2E Hijack")}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 403, status, "%s may not rename another user: %s", label, body)
		status, body, err = c.PutBytes(target.path()+"/photo", "image/png", onePixelPNG(t))
		require.NoError(t, err)
		assert.Equal(t, 403, status, "%s may not replace another user's photo: %s", label, body)
	}
	assert.Equal(t, before, userValues(readUser(t, apiClient, target.id)), "the refused edits changed nothing")

	manager := customRoleClient(t, "team:read", "team:update")
	name := uniqueName("E2E Managed")
	status, body, err := manager.Patch(target.path(), map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, name, jsonField(parseJSON(body), "name"))
	status, body, err = manager.PutBytes(target.path()+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// Once removed from the account a user is no longer its to edit: the profile, photo and read all answer 404.
func TestDashAccounts_ARemovedMemberIsNoLongerTheAccountsToEdit(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedSalesRepRoleID)
	removeAccountUser(u.accountUserID)
	require.Equal(t, "removed", dashSuppliersMemberStatus(t, apiClient, u.accountUserID))

	dashSuppliersExpectAll(t, 404, []dashSuppliersCall{
		{"retrieve", func() (int, []byte, error) { return apiClient.GetListRaw(u.path(), nil) }},
		{"rename", func() (int, []byte, error) {
			return apiClient.Patch(u.path(), map[string]any{"name": uniqueName("E2E Former")}, newIdempotencyKey())
		}},
		{"upload photo", func() (int, []byte, error) { return apiClient.PutBytes(u.path()+"/photo", "image/png", onePixelPNG(t)) }},
	})
	var name string
	require.NoError(t, authDB(t).QueryRow("SELECT name FROM user WHERE id = ?", u.id).Scan(&name))
	assert.Equal(t, "E2E User", name, "the former member's name is untouched")
}
