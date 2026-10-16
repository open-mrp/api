//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A member removed from the account stays the sales rep or responsible user of the records that name them, as
// history: the account user reads status removed and its user still shows who they were.
func TestRemovedSalesRep_IncludesStayReadable(t *testing.T) {
	t.Parallel()
	repID := accountUsersMember(t, SeedSalesRepRoleID)

	customerName := uniqueName("e2e-removed-rep-cust")
	body := validCustomerBody(customerName)
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

	paymentID := jsonField(createPayment(t, orderCustomerID, "12.50", nil, map[string]any{"responsible_user_id": repID}), "id")

	status, raw, err = apiClient.GetListRaw(accountUsersPath+"/"+repID, url.Values{"include": {"user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	repUser := jsonObject(parseJSON(raw), "user")
	require.NotNil(t, repUser)
	repUserID, repEmail := jsonField(repUser, "id"), jsonField(repUser, "email")
	require.NotEmpty(t, repEmail)

	status, raw, err = apiClient.Put(accountUsersPath+"/"+repID+"/actions/remove", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	// Records the caller may read carry their rep's user along, whatever the caller may read of the team.
	callers := map[string]*Client{
		"admin": apiClient,
		"role without team read": customRoleClient(t, "customers:read", "sales_rep_territories:read", "transactions:read"),
	}
	for name, client := range callers {
		t.Run(name, func(t *testing.T) {
			assertRemovedRep := func(t *testing.T, rep map[string]any) {
				t.Helper()
				require.NotNil(t, rep, "the removed member is still named")
				assert.Equal(t, repID, jsonField(rep, "id"))
				assert.Equal(t, "account_user", jsonField(rep, "object"))
				assert.Equal(t, "removed", jsonField(rep, "status"))
				user := jsonObject(rep, "user")
				require.NotNil(t, user, "a removed member's user still shows who they were")
				assert.Equal(t, repUserID, jsonField(user, "id"))
				assert.Equal(t, "user", jsonField(user, "object"))
				assert.Equal(t, repEmail, jsonField(user, "email"))
			}

			t.Run("customer defaults.sales_rep.user", func(t *testing.T) {
				status, raw, err := client.GetListRaw(customersPath+"/"+customerID, url.Values{"include": {"defaults.sales_rep,defaults.sales_rep.user"}})
				require.NoError(t, err)
				requireStatus(t, 200, status, raw)
				assertRemovedRep(t, jsonObject(jsonObject(parseJSON(raw), "defaults"), "sales_rep"))
			})

			t.Run("customer list defaults.sales_rep.user", func(t *testing.T) {
				status, raw, err := client.GetListRaw(customersPath, url.Values{"include": {"defaults.sales_rep,defaults.sales_rep.user"}, "q": {customerName}})
				require.NoError(t, err)
				requireStatus(t, 200, status, raw)
				var row map[string]any
				for _, d := range parseJSON(raw)["data"].([]any) {
					if m := d.(map[string]any); jsonField(m, "id") == customerID {
						row = m
					}
				}
				require.NotNil(t, row)
				assertRemovedRep(t, jsonObject(jsonObject(row, "defaults"), "sales_rep"))
			})

			t.Run("territory sales_rep.user", func(t *testing.T) {
				status, raw, err := client.GetListRaw(territoriesPath()+"/"+territoryID, url.Values{"include": {"sales_rep,sales_rep.user"}})
				require.NoError(t, err)
				requireStatus(t, 200, status, raw)
				assertRemovedRep(t, jsonObject(parseJSON(raw), "sales_rep"))
			})

			t.Run("transaction responsible_user.user", func(t *testing.T) {
				status, raw, err := client.GetListRaw(financeTransactionsPath+"/"+paymentID, url.Values{"include": {"responsible_user,responsible_user.user"}})
				require.NoError(t, err)
				requireStatus(t, 200, status, raw)
				assertRemovedRep(t, jsonObject(parseJSON(raw), "responsible_user"))
			})
		})
	}

	t.Run("order sales_rep", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"sales_rep"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		rep := jsonObject(parseJSON(raw), "sales_rep")
		require.NotNil(t, rep, "the removed rep is still the order's rep")
		assert.Equal(t, "actor", jsonField(rep, "object"))
		assert.Equal(t, repID, jsonField(rep, "id"))
	})

	t.Run("role without team read cannot read the member directly", func(t *testing.T) {
		client := callers["role without team read"]
		status, raw, err := client.GetListRaw(accountUsersPath+"/"+repID, nil)
		require.NoError(t, err)
		requireStatus(t, 403, status, raw)
		status, raw, err = client.GetListRaw("/v1/identity/users/"+repUserID, nil)
		require.NoError(t, err)
		requireStatus(t, 403, status, raw)
	})

	t.Run("account user include user", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw(accountUsersPath+"/"+repID, url.Values{"include": {"user"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		au := parseJSON(raw)
		assert.Equal(t, "removed", jsonField(au, "status"))
		assert.Equal(t, repUserID, jsonField(jsonObject(au, "user"), "id"))
	})

	// The user an include hands out is the one its own retrieve returns to a team reader.
	t.Run("user retrieve", func(t *testing.T) {
		status, raw, err := apiClient.GetListRaw("/v1/identity/users/"+repUserID, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		user := parseJSON(raw)
		assert.Equal(t, repUserID, jsonField(user, "id"))
		assert.Equal(t, repEmail, jsonField(user, "email"))
	})

	t.Run("removed member's profile is not editable", func(t *testing.T) {
		status, raw, err := apiClient.Patch("/v1/identity/users/"+repUserID, map[string]any{"name": "Renamed"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, raw)
	})
}
