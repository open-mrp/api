//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// accepts an export and polls its job through to completion. The file itself is out of
// reach — test mode's object store discards the bytes core-service's tests parse.
func completedExportJob(t *testing.T, path string, filters map[string]any) map[string]any {
	t.Helper()
	if filters == nil {
		filters = map[string]any{}
	}

	status, body, err := apiClient.Post(path, filters, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, body)

	accepted := parseJSON(body)
	assert.Equal(t, "job", jsonField(accepted, "object"), "202 returns the canonical job resource")
	assert.Equal(t, "export", jsonField(accepted, "type"))
	jobID := jsonField(accepted, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	var job map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		resp, err := apiClient.GetFull(jobsPath+"/"+jobID, nil)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("export job %s GET returned status %d", jobID, resp.StatusCode)
		}
		job = parseJSON(resp.Body)
		switch status := jsonField(job, "status"); status {
		case "completed":
			return nil
		case "failed", "cancelled":
			t.Fatalf("export job %s %s: %s", jobID, status, string(resp.Body))
			return nil
		default:
			return fmt.Errorf("export job %s not finished yet (status %q)", jobID, status)
		}
	})

	return job
}

func TestExports_EveryResourceRendersThroughAJob(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{"units", unitsPath},
		{"unit groups", unitGroupsPath},
		{"product lines", productLinesPath},
		{"item categories", itemCategoriesPath},
		{"departments", departmentsPath},
		{"storage locations", locationsPath},
		{"machines", machinesPath},
		{"scanning stations", scanningStationsPath},
		{"production runs", productionRunsPath},
		{"production steps", productionStepsPath},
		{"parts", partsPath},
		{"products", productsPath},
		{"materials", materialsPath},
		{"properties", propertiesPath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			job := completedExportJob(t, tc.path+"/actions/export", nil)
			export, ok := job["export"].(map[string]any)
			require.True(t, ok, "a completed export must carry its download: %v", job)
			assert.NotEmpty(t, export["url"])
		})
	}
}

// A filter reaches the worker: it is stored on the job at accept and read back when the
// render runs, so a request that narrows still produces a file.
func TestExports_AcceptCarriesFiltersThroughToTheWorker(t *testing.T) {
	sku := uniqueName("e2e-export-filter")
	createdIDs, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":      sku,
		"category": map[string]any{"id": SeedMaterialCategoryID},
	})
	t.Cleanup(func() { cleanupMaterialIDs(createdIDs) })

	job := completedExportJob(t, materialsPath+"/actions/export", map[string]any{"q": sku})
	export, ok := job["export"].(map[string]any)
	require.True(t, ok, "a completed export must carry its download: %v", job)
	assert.NotEmpty(t, export["url"])
}

// Reading a job answers with the job, never a redirect to its file — a client polling one
// must not be sent somewhere else on the poll that happens to succeed.
func TestExports_ReadingAJobNeverRedirects(t *testing.T) {
	job := completedExportJob(t, materialsPath+"/actions/export", nil)
	jobID := jsonField(job, "id")

	code, _, raw, err := apiClient.GetWithoutFollowingRedirects(jobsPath + "/" + jobID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code, "a completed export must still read as a job: %s", string(raw))
	assert.Equal(t, "job", jsonField(parseJSON(raw), "object"))
}

// Only an export carries a file, so a bulk job's export stays null.
func TestExports_ABulkJobHasNoDownload(t *testing.T) {
	job := bulkUpsertMaterialsJob(t, map[string]any{
		"sku":      uniqueName("e2e-export-bulkjob"),
		"category": map[string]any{"id": SeedMaterialCategoryID},
	})
	createdIDs, _ := jobResultIDs(job)
	t.Cleanup(func() { cleanupMaterialIDs(createdIDs) })

	assert.Nil(t, job["export"], "only an export job carries a download")
}
