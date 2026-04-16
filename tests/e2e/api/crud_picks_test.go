//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const picksPath = "/v1/operations/picks"

// ──────────────────────────────────────────────
// Pick — Include Tests
// ──────────────────────────────────────────────
//
// Pick GET endpoint whitelists: lines.
// (departments is a registered include but not allowed on this endpoint.)

func TestPicks_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["lines"], "lines should be null without ?include=lines")
}

func TestPicks_IncludeLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines should be present with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}
