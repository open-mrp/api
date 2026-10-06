//go:build e2e

package api_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultE2EObjectStoreAddr is where the host reaches the stack's object store: minio-e2e, published on 9002.
const defaultE2EObjectStoreAddr = "127.0.0.1:9002"

// accepts an export and polls its job through to completion
func completedExportJob(t *testing.T, path string, filters map[string]any) map[string]any {
	t.Helper()
	return completedExportJobAs(t, apiClient, path, filters)
}

// completedExportJobAs is completedExportJob for a caller other than the seed account's admin.
func completedExportJobAs(t *testing.T, client *Client, path string, filters map[string]any) map[string]any {
	t.Helper()
	if filters == nil {
		filters = map[string]any{}
	}

	status, body, err := client.Post(path, filters, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, body)

	accepted := parseJSON(body)
	assert.Equal(t, "job", jsonField(accepted, "object"), "202 returns the canonical job resource")
	assert.Equal(t, "export", jsonField(accepted, "type"))
	jobID := jsonField(accepted, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	return awaitExportJob(t, client, jobID)
}

// exportJobWaitTimeout is how long an export job may take to finish. Export jobs share one worker and
// queue behind each other, so under a full run's load one waits well past the usual async wait.
const exportJobWaitTimeout = 90 * time.Second

// polls an accepted export's job until it settles, failing the test unless it completes
func awaitExportJob(t *testing.T, client *Client, jobID string) map[string]any {
	t.Helper()
	var job map[string]any
	eventually(t, exportJobWaitTimeout, e2eAsyncPollInterval, func() error {
		resp, err := client.GetFull(jobsPath+"/"+jobID, nil)
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

// downloadExportFile fetches a completed export's file and the name it downloads under. The link is signed for the object store's in-network host, so the request keeps that host while dialing the port the stack publishes (E2E_OBJECT_STORE_ADDR overrides it).
func downloadExportFile(t *testing.T, job map[string]any) (filename string, body []byte) {
	t.Helper()
	export := jsonObject(job, "export")
	require.NotNil(t, export, "a completed export links its file: %v", job)
	signed := jsonField(export, "url")
	link, err := url.Parse(signed)
	require.NoError(t, err)

	addr := envOr("E2E_OBJECT_STORE_ADDR", defaultE2EObjectStoreAddr)
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}}
	resp, err := client.Get(signed)
	require.NoError(t, err, "downloading the export (is the stack up with minio-e2e published on %s?)", addr)
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "the signed link serves the file: %s", string(body))
	return path.Base(link.Path), body
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
