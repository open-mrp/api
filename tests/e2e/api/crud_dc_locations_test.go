//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dcLocationsPath = "/v1/operations/dc-locations"

// ──────────────────────────────────────────────
// DCLocation — Customer inline
// ──────────────────────────────────────────────
//
// DCLocation always inlines the customer summary. The customer field is NOT
// expandable — it is always populated from denormalized data.

func TestDCLocations_CustomerAlwaysInline(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(dcLocationsPath+"/"+SeedDCLocationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cust := jsonObject(got, "customer")
	require.NotNil(t, cust, "customer should always be inline on GET")
	assert.Equal(t, "customer", jsonField(cust, "object"))
	assert.NotEmpty(t, jsonField(cust, "id"))
}
