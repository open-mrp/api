//go:build e2e

package api_test

import (
	"net/url"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const customerAccessPath = "/v1/sales/product-line-access/customers"

// accessFixture is a customer and a group with no product line access yet, and two product lines this
// account owns to grant them.
type accessFixture struct {
	customerID, groupID string
	lineA, lineB        string
}

func newAccessFixture(t *testing.T) accessFixture {
	t.Helper()
	line := func(prefix string) string {
		return jsonField(createAndCleanup(t, productLinesPath, map[string]any{
			"name":              uniqueName(prefix),
			"unit_group_id":     SeedUnitGroupID,
			"commission_policy": "commission_applied",
			"freight_policy":    "billed_freight",
		}), "id")
	}
	return accessFixture{
		customerID: customerInGroup(t, SeedCustomerGroupID),
		groupID:    newAccountGroup(t),
		lineA:      line("e2e-pla-a"),
		lineB:      line("e2e-pla-b"),
	}
}

// grantAccess creates an access record. The record is keyed by its customer or group, so it carries
// no id of its own; callers revoke it themselves.
func grantAccess(t *testing.T, path string, body map[string]any) {
	t.Helper()
	status, resp, err := apiClient.Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
}

// grantedLines is the sorted product line IDs an access record grants.
func grantedLines(t *testing.T, record map[string]any) []string {
	t.Helper()
	lines, ok := record["product_lines"].(map[string]any)
	require.True(t, ok, "product_lines is a list: %v", record)
	data, _ := lines["data"].([]any)
	ids := make([]string, 0, len(data))
	for _, l := range data {
		ids = append(ids, l.(map[string]any)["id"].(string))
	}
	sort.Strings(ids)
	return ids
}

// errorParam is the request field an error response names.
func errorParam(body []byte) string {
	apiErr, _ := parseJSON(body)["error"].(map[string]any)
	return jsonField(apiErr, "param")
}

func sorted(ids ...string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}

func readAccess(t *testing.T, path string) (int, map[string]any) {
	t.Helper()
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "read must not 5xx: %s", body)
	if status != 200 {
		return status, nil
	}
	return status, parseJSON(body)
}

// listedAccess walks every page of an access list, limit at a time, and returns the records by the ID
// keyField's object carries, failing on a record seen twice.
func listedAccess(t *testing.T, path, keyField string, limit int) map[string]map[string]any {
	t.Helper()
	seen := map[string]map[string]any{}
	params := url.Values{"limit": {itoa(limit)}}
	for page := 0; page < 100; page++ {
		list, _, err := apiClient.GetList(path, params)
		require.NoError(t, err)
		for _, raw := range list.Data {
			record := parseJSON(raw)
			owner, _ := record[keyField].(map[string]any)
			id, _ := owner["id"].(string)
			require.NotContains(t, seen, id, "a record appears on one page only")
			seen[id] = record
		}
		cursor := list.PageInfo.NextCursor()
		if !list.PageInfo.HasNextPage || cursor == nil {
			return seen
		}
		params.Set("cursor", *cursor)
	}
	t.Fatalf("%s did not end within 100 pages", path)
	return nil
}

// A customer's access is granted, read, replaced and revoked as one set of product lines.
func TestCustomerProductLineAccess_GrantReplaceRevoke(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := customerAccessPath + "/" + f.customerID

	status, body, err := apiClient.Post(customerAccessPath, map[string]any{
		"customer_id":      f.customerID,
		"product_line_ids": []string{f.lineA},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	assert.Equal(t, "customer_product_line_access", created["object"])
	customer, _ := created["customer"].(map[string]any)
	assert.Equal(t, f.customerID, customer["id"])
	assert.NotEmpty(t, customer["name"], "the customer is named, as the access list shows it")
	assert.NotEmpty(t, customer["number"], "the customer's number is shown with its name")
	assert.Equal(t, []string{f.lineA}, grantedLines(t, created))
	expectAuditEvent(t, f.customerID, "customer_product_line_access", "create")

	status, read := readAccess(t, path)
	require.Equal(t, 200, status)
	assert.Equal(t, []string{f.lineA}, grantedLines(t, read))

	// The set is replaced, not merged.
	status, body, err = apiClient.Patch(path, map[string]any{"product_line_ids": []string{f.lineB, f.lineA}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, sorted(f.lineA, f.lineB), grantedLines(t, parseJSON(body)))
	status, body, err = apiClient.Patch(path, map[string]any{"product_line_ids": []string{f.lineB}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{f.lineB}, grantedLines(t, parseJSON(body)))
	_, read = readAccess(t, path)
	assert.Equal(t, []string{f.lineB}, grantedLines(t, read), "the replacement is what a read returns")
	expectAuditEvent(t, f.customerID, "customer_product_line_access", "update")

	listed := listedAccess(t, customerAccessPath, "customer", 100)
	require.Contains(t, listed, f.customerID, "a customer with access is listed")
	assert.Equal(t, []string{f.lineB}, grantedLines(t, listed[f.customerID]))

	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	expectAuditEvent(t, f.customerID, "customer_product_line_access", "delete")

	status, _ = readAccess(t, path)
	assert.Equal(t, 404, status, "revoked access has no record to read")
	assert.NotContains(t, listedAccess(t, customerAccessPath, "customer", 100), f.customerID,
		"a customer without access is not listed")

	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	assert.Equal(t, 410, status, "revoking twice reports the record gone: %s", body)

	// Revoked access can be granted afresh.
	status, body, err = apiClient.Post(customerAccessPath, map[string]any{
		"customer_id":      f.customerID,
		"product_line_ids": []string{f.lineA},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })
}

// A customer holds one set of product lines; a second grant is refused, not merged.
func TestCustomerProductLineAccess_ASecondGrantConflicts(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := customerAccessPath + "/" + f.customerID
	grantAccess(t, customerAccessPath, map[string]any{"customer_id": f.customerID, "product_line_ids": []string{f.lineA}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })

	status, body, err := apiClient.Post(customerAccessPath, map[string]any{
		"customer_id":      f.customerID,
		"product_line_ids": []string{f.lineB},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status, "a second grant conflicts: %s", body)
	assert.Equal(t, "customer_id", errorParam(body))

	_, read := readAccess(t, path)
	assert.Equal(t, []string{f.lineA}, grantedLines(t, read), "the refused grant changed nothing")
}

// An access record is its product lines. Granting none would leave nothing to read or edit, so it is
// refused, and an edit to none leaves the grant as it was; revoking is a delete.
func TestCustomerProductLineAccess_AGrantOfNoProductLinesIsRefused(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := customerAccessPath + "/" + f.customerID

	status, body, err := apiClient.Post(customerAccessPath, map[string]any{
		"customer_id":      f.customerID,
		"product_line_ids": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an empty grant is refused: %s", body)
	assert.Equal(t, "product_line_ids", errorParam(body))
	status, _ = readAccess(t, path)
	assert.Equal(t, 404, status, "the refused grant created nothing")

	grantAccess(t, customerAccessPath, map[string]any{"customer_id": f.customerID, "product_line_ids": []string{f.lineA, f.lineB}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })

	status, body, err = apiClient.Patch(path, map[string]any{"product_line_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an edit to no product lines is refused: %s", body)
	assert.Equal(t, "product_line_ids", errorParam(body))

	status, body, err = apiClient.Patch(path, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an edit naming nothing is refused: %s", body)

	status, read := readAccess(t, path)
	require.Equal(t, 200, status)
	assert.Equal(t, sorted(f.lineA, f.lineB), grantedLines(t, read), "the refused edits changed nothing")
}

// Only product lines this account owns can be granted, and a refused line leaves the grant untouched.
func TestCustomerProductLineAccess_AnUnknownProductLineIsRefused(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := customerAccessPath + "/" + f.customerID

	status, body, err := apiClient.Post(customerAccessPath, map[string]any{
		"customer_id":      f.customerID,
		"product_line_ids": []string{f.lineA, "pdln_doesnotexist00"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "an unknown product line is refused: %s", body)
	status, _ = readAccess(t, path)
	assert.Equal(t, 404, status, "a refused grant creates nothing, not even its known lines")

	grantAccess(t, customerAccessPath, map[string]any{"customer_id": f.customerID, "product_line_ids": []string{f.lineA}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })

	status, body, err = apiClient.Patch(path, map[string]any{"product_line_ids": []string{f.lineB, "pdln_doesnotexist00"}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "an edit with an unknown product line is refused: %s", body)
	_, read := readAccess(t, path)
	assert.Equal(t, []string{f.lineA}, grantedLines(t, read), "the refused edit changed nothing")

	status, body, err = apiClient.Post(customerAccessPath, map[string]any{
		"customer_id":      "ac_doesnotexist0000",
		"product_line_ids": []string{f.lineA},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "access for an unknown customer is refused: %s", body)
}

// Another account cannot read, grant, edit or revoke this account's customer access.
func TestCustomerProductLineAccess_IsTheAccountsOwn(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := customerAccessPath + "/" + f.customerID
	grantAccess(t, customerAccessPath, map[string]any{"customer_id": f.customerID, "product_line_ids": []string{f.lineA}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })
	other := getTenantBClient()

	status, body, err := other.GetListRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "read: %s", body)
	status, body, err = other.Patch(path, map[string]any{"product_line_ids": []string{f.lineB}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "edit: %s", body)
	status, body, err = other.Delete(path)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "revoke: %s", body)
	status, body, err = other.Post(customerAccessPath, map[string]any{
		"customer_id":      f.customerID,
		"product_line_ids": []string{f.lineB},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "grant to another account's customer: %s", body)

	_, read := readAccess(t, path)
	assert.Equal(t, []string{f.lineA}, grantedLines(t, read), "nothing the other account tried changed the grant")
}

// The dashboard reads every record by walking pages; each record comes once at any page size.
func TestCustomerProductLineAccess_PagesWalkEveryRecordOnce(t *testing.T) {
	t.Parallel()
	var ids []string
	for range 3 {
		f := newAccessFixture(t)
		grantAccess(t, customerAccessPath, map[string]any{"customer_id": f.customerID, "product_line_ids": []string{f.lineA}})
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customerAccessPath + "/" + f.customerID) })
		ids = append(ids, f.customerID)
	}
	listed := listedAccess(t, customerAccessPath, "customer", 1)
	for _, id := range ids {
		assert.Contains(t, listed, id)
	}
}

// A group's access is granted, replaced and revoked as one set, like a customer's.
func TestAccountGroupProductLineAccess_GrantReplaceRevoke(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := accountGroupAccessPath + "/" + f.groupID

	status, body, err := apiClient.Post(accountGroupAccessPath, map[string]any{
		"account_group_id": f.groupID,
		"product_line_ids": []string{f.lineA, f.lineB},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	assert.Equal(t, "account_group_product_line_access", created["object"])
	group, _ := created["account_group"].(map[string]any)
	assert.Equal(t, f.groupID, group["id"])
	assert.NotEmpty(t, group["name"])
	assert.Equal(t, sorted(f.lineA, f.lineB), grantedLines(t, created))
	expectAuditEvent(t, f.groupID, "account_group_product_line_access", "create")

	status, body, err = apiClient.Patch(path, map[string]any{"product_line_ids": []string{f.lineB}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{f.lineB}, grantedLines(t, parseJSON(body)))
	_, read := readAccess(t, path)
	assert.Equal(t, []string{f.lineB}, grantedLines(t, read))
	expectAuditEvent(t, f.groupID, "account_group_product_line_access", "update")

	listed := listedAccess(t, accountGroupAccessPath, "account_group", 1)
	require.Contains(t, listed, f.groupID, "a group with access is listed")
	assert.Equal(t, []string{f.lineB}, grantedLines(t, listed[f.groupID]))

	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, _ = readAccess(t, path)
	assert.Equal(t, 404, status, "revoked access has no record to read")
	assert.NotContains(t, listedAccess(t, accountGroupAccessPath, "account_group", 100), f.groupID)
}

func TestAccountGroupProductLineAccess_ASecondGrantConflicts(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := accountGroupAccessPath + "/" + f.groupID
	grantAccess(t, accountGroupAccessPath, map[string]any{"account_group_id": f.groupID, "product_line_ids": []string{f.lineA}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })

	status, body, err := apiClient.Post(accountGroupAccessPath, map[string]any{
		"account_group_id": f.groupID,
		"product_line_ids": []string{f.lineB},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status, "a second grant conflicts: %s", body)
	assert.Equal(t, "account_group_id", errorParam(body))
	_, read := readAccess(t, path)
	assert.Equal(t, []string{f.lineA}, grantedLines(t, read))
}

// Emptying a group's product lines used to delete every grant and answer 200, leaving a record that read
// as empty but refused every later edit as not found. An edit to none is now refused and changes nothing.
func TestAccountGroupProductLineAccess_AGrantOfNoProductLinesIsRefused(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := accountGroupAccessPath + "/" + f.groupID

	status, body, err := apiClient.Post(accountGroupAccessPath, map[string]any{
		"account_group_id": f.groupID,
		"product_line_ids": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an empty grant is refused: %s", body)
	assert.Equal(t, "product_line_ids", errorParam(body))

	grantAccess(t, accountGroupAccessPath, map[string]any{"account_group_id": f.groupID, "product_line_ids": []string{f.lineA, f.lineB}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })

	status, body, err = apiClient.Patch(path, map[string]any{"product_line_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an edit to no product lines is refused: %s", body)
	assert.Equal(t, "product_line_ids", errorParam(body))

	status, read := readAccess(t, path)
	require.Equal(t, 200, status)
	assert.Equal(t, sorted(f.lineA, f.lineB), grantedLines(t, read), "the refused edit changed nothing")

	// The record stays editable.
	status, body, err = apiClient.Patch(path, map[string]any{"product_line_ids": []string{f.lineA}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestAccountGroupProductLineAccess_AnUnknownProductLineIsRefused(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := accountGroupAccessPath + "/" + f.groupID

	status, body, err := apiClient.Post(accountGroupAccessPath, map[string]any{
		"account_group_id": f.groupID,
		"product_line_ids": []string{"pdln_doesnotexist00"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "an unknown product line is refused: %s", body)
	status, _ = readAccess(t, path)
	assert.Equal(t, 404, status, "the refused grant created nothing")

	status, body, err = apiClient.Post(accountGroupAccessPath, map[string]any{
		"account_group_id": "acgp_doesnotexist00",
		"product_line_ids": []string{f.lineA},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "access for an unknown group is refused: %s", body)
}

func TestAccountGroupProductLineAccess_IsTheAccountsOwn(t *testing.T) {
	t.Parallel()
	f := newAccessFixture(t)
	path := accountGroupAccessPath + "/" + f.groupID
	grantAccess(t, accountGroupAccessPath, map[string]any{"account_group_id": f.groupID, "product_line_ids": []string{f.lineA}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path) })
	other := getTenantBClient()

	status, body, err := other.GetListRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "read: %s", body)
	status, body, err = other.Patch(path, map[string]any{"product_line_ids": []string{f.lineB}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "edit: %s", body)
	status, body, err = other.Delete(path)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "revoke: %s", body)

	_, read := readAccess(t, path)
	assert.Equal(t, []string{f.lineA}, grantedLines(t, read))
}
