//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A member removed from the account stays the sales rep of the customers and orders that name them, as history, but
// their user profile is no longer read through the account: sales_rep.user is null rather than a 403.
func TestRemovedSalesRep_IncludesStayReadable(t *testing.T) {
	t.Parallel()
	repID := accountUsersMember(t, SeedSalesRepRoleID)

	body := validCustomerBody(uniqueName("e2e-removed-rep-cust"))
	customerDefaults(body)["sales_rep_id"] = repID
	customerID := jsonField(createAndCleanup(t, customersPath, body), "id")

	orderCustomerID := setupOrderCustomer(t)
	status, raw, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, orderCustomerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	orderID := jsonField(parseJSON(raw), "id")
	deleteOrder(t, orderID)
	status, raw, err = apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{"sales_rep_id": repID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	territoryStatus, territoryBody, err := apiClient.Post(territoriesPath(), map[string]any{
		"state":         "TX",
		"start_zipcode": 76001,
		"end_zipcode":   76999,
		"sales_rep_id":  repID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, territoryStatus, territoryBody)
	territoryID := jsonField(parseJSON(territoryBody), "id")
	t.Cleanup(func() { apiClient.Delete(territoriesPath() + "/" + territoryID) })

	// While the rep is a member, their user resolves.
	status, raw, err = apiClient.GetListRaw(customersPath+"/"+customerID, url.Values{"include": {"defaults.sales_rep,defaults.sales_rep.user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	require.NotNil(t, jsonObject(jsonObject(jsonObject(parseJSON(raw), "defaults"), "sales_rep"), "user"))

	status, raw, err = apiClient.Put(accountUsersPath+"/"+repID+"/actions/remove", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	t.Run("customer defaults.sales_rep.user", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(customersPath+"/"+customerID, url.Values{"include": {"defaults.sales_rep,defaults.sales_rep.user"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		rep := jsonObject(jsonObject(parseJSON(raw), "defaults"), "sales_rep")
		require.NotNil(t, rep, "the removed rep is still the customer's default rep")
		assert.Equal(t, repID, jsonField(rep, "id"))
		assert.Equal(t, "removed", jsonField(rep, "status"))
		assert.Nil(t, rep["user"], "a removed member's user is not read through the account")
	})

	t.Run("customer list defaults.sales_rep.user", func(t *testing.T) {
		row := listFindByField(t, customersPath, url.Values{"include": {"defaults.sales_rep,defaults.sales_rep.user"}}, "id", customerID)
		require.NotNil(t, row)
		rep := jsonObject(jsonObject(parseJSON(row), "defaults"), "sales_rep")
		require.NotNil(t, rep)
		assert.Equal(t, repID, jsonField(rep, "id"))
		assert.Nil(t, rep["user"])
	})

	t.Run("territory sales_rep.user", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(territoriesPath()+"/"+territoryID, url.Values{"include": {"sales_rep,sales_rep.user"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		rep := jsonObject(parseJSON(raw), "sales_rep")
		require.NotNil(t, rep)
		assert.Equal(t, repID, jsonField(rep, "id"))
		assert.Nil(t, rep["user"])
	})

	t.Run("order sales_rep", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"sales_rep"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		rep := jsonObject(parseJSON(raw), "sales_rep")
		require.NotNil(t, rep, "the removed rep is still the order's rep")
		assert.Equal(t, "actor", jsonField(rep, "object"))
		assert.Equal(t, repID, jsonField(rep, "id"))
	})

	t.Run("account user include user", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(accountUsersPath+"/"+repID, url.Values{"include": {"user"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		au := parseJSON(raw)
		assert.Equal(t, "removed", jsonField(au, "status"))
		assert.NotNil(t, au["user"], "reading the account user itself still shows who it was")
	})
}
