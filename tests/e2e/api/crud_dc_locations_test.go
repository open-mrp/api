//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dcLocationsPath = "/v1/operations/dc-locations"

// ──────────────────────────────────────────────
// DCLocation — Include Tests
// ──────────────────────────────────────────────
//
// DCLocation list endpoint always returns `customer` as a summary. The Get
// endpoint exposes `customer` as an expandable include.

func TestDCLocations_CustomerNullWithoutIncludeOnGet(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(dcLocationsPath+"/"+SeedDCLocationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// On the Get endpoint, customer is an expandable include; it should be null without ?include=customer.
	// Note: on the List endpoint, customer is always populated as a summary.
	assert.Nil(t, got["customer"], "customer should be null without ?include=customer on GET")
}

func TestDCLocations_IncludeCustomer(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(dcLocationsPath+"/"+SeedDCLocationID, url.Values{"include": {"customer"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cust := jsonObject(got, "customer")
	require.NotNil(t, cust, "customer should be present with ?include=customer")
	assert.Equal(t, "customer", jsonField(cust, "object"))
	assert.NotEmpty(t, jsonField(cust, "id"))
}
