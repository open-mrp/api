//go:build e2e

package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A preview.9 caller still runs analytics reports with PUT on the report's own route.
func TestVersionCompatAnalytics_Preview9Put(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	now := time.Now().UTC()
	window := map[string]any{"starts_at": rfc3339(now.AddDate(0, -1, 0)), "ends_at": rfc3339(now)}

	status, raw, err := preview.Put(analyzeSalesPath, window)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)

	status, raw, err = apiClient.QueryAnalytics(analyzeSalesPath, nil, window)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)

	status, raw, err = apiClient.Put(analyzeSalesPath, window)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, status, "forge has no PUT report route: %s", raw)
}

// A preview.9 caller still generates a pack list from /v1/core/records with the shipment in the body.
func TestVersionCompatAnalytics_Preview9PackList(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	status, raw, err := preview.Post("/v1/core/records/actions/generate-pack-list", map[string]any{"shipment_id": SeedShipmentID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assertObjectField(t, parseJSON(raw), "pack_list")

	status, raw, err = preview.Post("/v1/core/records/actions/generate-pack-list", map[string]any{"shipment_id": "sh_01k0a87w33emw8pmkz1mf80000"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, raw)
}
