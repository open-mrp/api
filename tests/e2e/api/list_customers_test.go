//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const customersPath = "/v1/sales/customers"

func TestListCustomers_ReturnsSeededData(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded customer")

	// Paginate until found: seed rows are the oldest and fall off the
	// first page as repeated e2e runs accumulate data.
	assert.NotNil(t, listFindByField(t, customersPath, nil, "name", SeedCustomerName),
		"Seeded customer %q not found in list", SeedCustomerName)
}

func TestListCustomers_SearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"q": {"Global"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Global' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "global"),
			"Search result %q should contain 'global'", name,
		)
	}
}

func TestListCustomers_SearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{"q": {"zzzznotacustomer99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense search should return empty data")
}

func TestListCustomers_FilterByCustomerGroup_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{
		"customer_group_ids": {"acgp_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense customer group filter should return empty data")
}

func TestListCustomers_FilterByCustomerGroup(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(customersPath, url.Values{
		"customer_group_ids": {SeedCustomerGroupID},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Filter by customer group should return at least 1 result")
}
