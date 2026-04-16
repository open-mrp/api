//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const settlementsPath = "/v1/finance/settlements"

// ──────────────────────────────────────────────
// Settlement — Include Tests
// ──────────────────────────────────────────────
//
// Settlement GET endpoint whitelists: allocations.
// (responsible_user is always populated as a summary on this endpoint.)

func TestSettlements_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(settlementsPath+"/"+SeedSettlementID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["allocations"], "allocations should be null without ?include=allocations")
}

func TestSettlements_IncludeAllocations(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(settlementsPath+"/"+SeedSettlementID, url.Values{"include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["allocations"]
	assert.True(t, ok, "allocations key should be present with ?include=allocations")
	if a := jsonObject(got, "allocations"); a != nil {
		assert.Equal(t, "list", jsonField(a, "object"))
	}
}
