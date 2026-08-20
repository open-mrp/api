//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover nested user expansion on account users embedded in other
// resources (e.g. ?include=defaults.sales_rep.user on customers). The embedded
// account_user is loaded via the account-user loader, and its `user`
// sub-resource resolves recursively to the full underlying user.

// assertExpandedAccountUserWithUser asserts the account_user shape with its
// nested user expanded.
func assertExpandedAccountUserWithUser(t *testing.T, au map[string]any, wantAccountUserID string) {
	t.Helper()

	require.NotNil(t, au, "embedded account_user should be expanded")
	assert.Equal(t, "account_user", jsonField(au, "object"))
	assert.Equal(t, wantAccountUserID, jsonField(au, "id"))

	user := jsonObject(au, "user")
	require.NotNil(t, user, "nested user should be expanded")
	assert.Equal(t, "user", jsonField(user, "object"))
	userID := jsonField(user, "id")
	assert.NotEmpty(t, userID, "nested user must have an id")
	assert.NotEqual(t, jsonField(au, "id"), userID, "user.id must differ from the account_user id")
	assert.NotEmpty(t, jsonField(user, "name"), "seed account user's underlying user should have a name")
	assert.NotEmpty(t, jsonField(user, "created_at"))
	assert.NotEmpty(t, jsonField(user, "updated_at"))
}

func TestNestedUserInclude_CustomerDefaultsSalesRep(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cust-nestusr")
	body := validCustomerBody(name)
	body["default_sales_rep_id"] = SeedAccountUserID

	createResp, err := apiClient.PostFull(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)
	id := jsonField(parseJSON(createResp.Body), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(customersPath + "/" + id)

	// Without the nested include the sales_rep's user stays null.
	status, getBody, err := apiClient.GetListRaw(customersPath+"/"+id, url.Values{"include": {"defaults.sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, getBody)
	sr := jsonObject(jsonObject(parseJSON(getBody), "defaults"), "sales_rep")
	require.NotNil(t, sr, "defaults.sales_rep should be expanded")
	assert.Equal(t, SeedAccountUserID, jsonField(sr, "id"))
	assert.Nil(t, sr["user"], "sales_rep.user should be null without ?include=defaults.sales_rep.user")

	// With the nested include the full user is attached.
	status, getBody, err = apiClient.GetListRaw(customersPath+"/"+id, url.Values{"include": {"defaults.sales_rep,defaults.sales_rep.user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, getBody)
	sr = jsonObject(jsonObject(parseJSON(getBody), "defaults"), "sales_rep")
	assertExpandedAccountUserWithUser(t, sr, SeedAccountUserID)
}

func TestNestedUserInclude_TerritorySalesRep(t *testing.T) {
	t.Parallel()
	createStatus, createBody, err := apiClient.Post(territoriesPath(), map[string]any{
		"state":         "TX",
		"start_zipcode": 75001,
		"end_zipcode":   75999,
		"sales_rep_id":  SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(territoriesPath() + "/" + id)

	status, body, err := apiClient.GetListRaw(territoriesPath()+"/"+id, url.Values{"include": {"sales_rep,sales_rep.user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	sr := jsonObject(parseJSON(body), "sales_rep")
	assertExpandedAccountUserWithUser(t, sr, SeedAccountUserID)
}

func TestNestedUserInclude_TransactionResponsibleUser(t *testing.T) {
	t.Parallel()
	createStatus, createBody, err := apiClient.Post(transactionsPath, map[string]any{
		"customer_id":         SeedCustomerAccountID,
		"type":                "payment",
		"amount":              "125.00",
		"responsible_user_id": SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(transactionsPath + "/" + id)

	status, body, err := apiClient.GetListRaw(transactionsPath+"/"+id, url.Values{"include": {"responsible_user,responsible_user.user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	ru := jsonObject(parseJSON(body), "responsible_user")
	assertExpandedAccountUserWithUser(t, ru, SeedAccountUserID)
}

// The seeded transaction stores a legacy user id (us_…) in
// responsible_user_id; the read query must resolve it to the account_user so
// the include works for legacy rows too.
func TestNestedUserInclude_TransactionLegacyUserIDRow(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(transactionsPath+"/"+SeedTransactionID, url.Values{"include": {"responsible_user,responsible_user.user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	ru := jsonObject(parseJSON(body), "responsible_user")
	assertExpandedAccountUserWithUser(t, ru, SeedAccountUserID)
}

func TestNestedUserInclude_ProductionRunResponsibleUser(t *testing.T) {
	t.Parallel()
	createStatus, createBody, err := apiClient.Post(productionRunsPath, map[string]any{
		"responsible_user_id": SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(productionRunsPath + "/" + id)

	status, body, err := apiClient.GetListRaw(productionRunsPath+"/"+id, url.Values{"include": {"responsible_user,responsible_user.user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	ru := jsonObject(parseJSON(body), "responsible_user")
	assertExpandedAccountUserWithUser(t, ru, SeedAccountUserID)

	// The list endpoint supports the same includes on summaries. Paginate
	// until found: rows accumulate across repeated e2e runs.
	item := listFindByField(t, productionRunsPath,
		url.Values{"include": {"responsible_user,responsible_user.user"}}, "id", id)
	require.NotNil(t, item, "created production run should appear in the list")
	assertExpandedAccountUserWithUser(t, jsonObject(parseJSON(item), "responsible_user"), SeedAccountUserID)
}

func TestNestedUserInclude_SettlementResponsibleUser(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(settlementsPath+"/"+SeedSettlementID, url.Values{"include": {"responsible_user,responsible_user.user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	ru := jsonObject(parseJSON(body), "responsible_user")
	assertExpandedAccountUserWithUser(t, ru, SeedAccountUserID)
}
