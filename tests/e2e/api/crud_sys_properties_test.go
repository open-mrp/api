//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sysPropertiesPath = "/v1/settings/properties"

// A PATCH without a value is refused and leaves the counter where it is: zeroing it would reissue every
// number the counter had handed out. The seeded counter is the account's transaction numbers, which
// other tests draw from, so its value is never changed here.
func TestSysProperties_UpdateWithoutValueKeepsTheCounter(t *testing.T) {
	t.Parallel()
	path := sysPropertiesPath + "/" + SeedSysPropertyID

	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	before := parseJSON(body)["value"]

	status, body, err = apiClient.Patch(path, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	status, body, err = apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, before, parseJSON(body)["value"], "the refused PATCH left the counter alone")

	status, body, err = apiClient.Patch(path, map[string]any{"value": before}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, before, parseJSON(body)["value"])
}
