//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// preview.6 made a production run's batch_count nullable, since it is withheld from anyone outside the seller's account. preview.5 always carried a number, and the seller's own callers, the only ones who may request a run, still read one on both versions.

func seedOrderProductionRunID(t *testing.T) string {
	t.Helper()
	related := jsonObject(getSalesOrder(t, SeedSalesOrderID, url.Values{"include": {"related.production_run"}}), "related")
	runID := jsonField(jsonObject(related, "production_run"), "id")
	require.NotEmpty(t, runID, "the seed order links to a production run")
	return runID
}

func TestVersionCompat_ProductionRuns_BatchCountStaysANumberForTheSeller(t *testing.T) {
	t.Parallel()
	runID := seedOrderProductionRunID(t)

	for _, apiVersion := range []string{preview5APIVersion, defaultAPIVersion} {
		client := apiClient.WithAPIVersion(apiVersion)

		run := parseJSON(mustGetAs(t, client, productionRunsPath+"/"+runID, nil))
		_, isNumber := run["batch_count"].(float64)
		assert.True(t, isNumber, "%s: a retrieved run's batch_count is a number: %v", apiVersion, run["batch_count"])

		list, status, err := client.GetList(productionRunsPath, url.Values{"limit": {"20"}})
		require.NoError(t, err)
		require.Equal(t, 200, status)
		require.NotEmpty(t, list.Data)
		for _, raw := range list.Data {
			listed := parseJSON(raw)
			_, isNumber := listed["batch_count"].(float64)
			assert.True(t, isNumber, "%s: listed run %s carries a numeric batch_count", apiVersion, jsonField(listed, "id"))
		}
	}
}
