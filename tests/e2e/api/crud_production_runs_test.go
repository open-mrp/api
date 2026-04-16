//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productionRunsPath = "/v1/operations/production-runs"

// ──────────────────────────────────────────────
// ProductionRun — Include Tests
// ──────────────────────────────────────────────

func TestProductionRuns_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionRunsPath+"/"+SeedProductionRunID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["responsible_user"], "responsible_user should be null without ?include=responsible_user")
}

func TestProductionRuns_IncludeResponsibleUser(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productionRunsPath+"/"+SeedProductionRunID, url.Values{"include": {"responsible_user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// responsible_user may legitimately be null if unassigned
	_, ok := got["responsible_user"]
	assert.True(t, ok, "responsible_user key should be present with ?include=responsible_user")
	if u := jsonObject(got, "responsible_user"); u != nil {
		assert.Equal(t, "account_user", jsonField(u, "object"))
		assert.NotEmpty(t, jsonField(u, "id"))
	}
}
