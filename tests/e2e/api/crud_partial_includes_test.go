//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Fills partial-coverage include gaps for
// resources whose CRUD/list tests cover only the
// "without include" case or omit certain fields.
//
// Tests already implemented elsewhere (e.g.,
// TestPaymentTerms_IncludeOwner, TestShippingTerms_IncludeOwner,
// TestProperties_IncludeAttributes) are NOT duplicated here.
// ──────────────────────────────────────────────

// ShippingTerm — positive test for free_shipping_service_levels (negative exists already)

func TestShippingTerms_IncludeFreeShippingServiceLevels(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedShippingTermID, url.Values{"include": {"free_shipping_service_levels"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["free_shipping_service_levels"]
	assert.True(t, ok, "free_shipping_service_levels key should be present with include")
	if sls := jsonObject(got, "free_shipping_service_levels"); sls != nil {
		assert.Equal(t, "list", jsonField(sls, "object"))
	}
}

// Priority — positive & negative owner include

func TestPriorities_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath+"/"+SeedPriorityID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	owner := jsonObject(parseJSON(body), "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
}

func TestPriorities_OwnerNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath+"/"+SeedPriorityID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["owner"], "owner should be null without ?include=owner")
}

// Unit — positive & negative owner include

func TestUnits_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitsPath+"/"+SeedUnitID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	owner := jsonObject(parseJSON(body), "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
}

func TestUnits_OwnerNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitsPath+"/"+SeedUnitID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["owner"], "owner should be null without ?include=owner")
}

// Properties — negative test for attributes (positive already exists)

func TestProperties_AttributesNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+SeedPropertyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["attributes"], "attributes should be null without ?include=attributes")
}

// AccountUser — negative case on Get endpoint + list-endpoint include

func TestAccountUsers_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountUsersPath+"/"+SeedAccountUserID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["role"], "role should be null without ?include=role")
	assert.Nil(t, got["department"], "department should be null without ?include=department")
}

// Note: AccountUsers list endpoint does not support the include parameter;
// it's only available on the Get endpoint (covered in crud_account_users_test.go).

// Customer — missing include tests

func TestCustomers_IncludeParentAccountNullWhenAbsent(t *testing.T) {
	t.Parallel()
	// Default seeded customer typically has no parent_account; the include should return null.
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"parent_account"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["parent_account"]
	assert.True(t, ok, "parent_account key should be present with ?include=parent_account")
}

func TestCustomers_IncludeChildAccounts(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"child_accounts"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// child_accounts key must be present with ?include=child_accounts. When a
	// customer has no children, this may be a list with empty data, or null.
	_, ok := got["child_accounts"]
	assert.True(t, ok, "child_accounts key should be present with ?include=child_accounts")
	if ca := jsonObject(got, "child_accounts"); ca != nil {
		assert.Equal(t, "list", jsonField(ca, "object"))
	}
}

func TestCustomers_IncludeDefaultsPriority(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"defaults.priority"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	defaults := jsonObject(parseJSON(body), "defaults")
	require.NotNil(t, defaults)
	_, ok := defaults["priority"]
	assert.True(t, ok, "defaults.priority key should be present with ?include=defaults.priority")
}

func TestCustomers_IncludeFreightPreferencesServiceLevel(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(customersPath+"/"+SeedCustomerAccountID, url.Values{"include": {"freight_preferences.service_level"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	fp := jsonObject(parseJSON(body), "freight_preferences")
	require.NotNil(t, fp)
	_, ok := fp["service_level"]
	assert.True(t, ok, "freight_preferences.service_level key should be present with ?include=freight_preferences.service_level")
}

// Static lists — SalesOrderStatus, AdjustmentType

func TestSalesOrderStatuses_IncludeOwner(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList("/v1/sales/sales-orders/statuses", url.Values{"include": {"owner"}})
	require.NoError(t, err)
	require.Equal(t, 200, status, "sales order statuses list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1)

	first := parseJSON(list.Data[0])
	_, ok := first["owner"]
	assert.True(t, ok, "owner should be present with ?include=owner")
}

func TestAdjustmentTypes_IncludeOwner(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList("/v1/finance/adjustment-types", url.Values{"include": {"owner"}})
	require.NoError(t, err)
	require.Equal(t, 200, status, "adjustment types list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1)

	first := parseJSON(list.Data[0])
	_, ok := first["owner"]
	assert.True(t, ok, "owner should be present with ?include=owner")
}
