//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the client to 1.0.forge-preview.2 to exercise the job transformer.
// preview.3 merged a job's per-row failures into `results` and moved the whole-job
// failure to `error`; a pinned client must still see the two separate arrays, the
// `error_summary` string, and the hoisted `created_by_*` fields it was written against.

const preview2APIVersion = "1.0.forge-preview.2"

// getJobAs reads a job through a client pinned to an older API version.
func getJobAs(t *testing.T, client *Client, jobID string) map[string]any {
	t.Helper()
	status, body, err := client.GetListRaw(jobsPath+"/"+jobID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func TestVersionCompat_Jobs_CarriesThePreview2Shape(t *testing.T) {
	t.Parallel()

	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-vc-job"), uniqueName("evcj"), "quantity"))
	// Poll on the current version so the helper reads the shape it knows, then re-read
	// the settled job through the pinned client.
	pollJobUntilTerminal(t, jobID)

	job := getJobAs(t, apiClient.WithAPIVersion(preview2APIVersion), jobID)
	created, _ := jobResultIDs(job)
	defer cleanupUnitIDs(created)

	assert.Equal(t, "job", jsonField(job, "object"))
	// preview.2 spelled the bulk types without a separator.
	assert.Equal(t, "bulkupsert", jsonField(job, "type"))
	// resource_type arrived in preview.3 and must not leak backwards.
	_, hasResourceType := job["resource_type"]
	assert.False(t, hasResourceType, "preview.2 had no resource_type")
	_, hasError := job["error"]
	assert.False(t, hasError, "preview.2 had no error field")

	// The creator is hoisted back onto the job, not nested in an actor.
	_, hasCreatedBy := job["created_by"]
	assert.False(t, hasCreatedBy, "preview.2 had no created_by object")
	for _, key := range []string{"created_by_id", "created_by_name", "created_by_username", "created_by_email"} {
		_, present := job[key]
		assert.True(t, present, "%s must be present on preview.2 job responses", key)
	}

	// results is a bare array of {index, id, action}, not a List envelope.
	results := jsonArray(job, "results")
	require.Len(t, results, 1, "the written row belongs in results")
	entry, ok := results[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "created", entry["action"], "preview.2 called the outcome an action")
	assert.NotEmpty(t, jsonField(entry, "id"), "preview.2 carried the bare resource id")
	_, hasResource := entry["resource"]
	assert.False(t, hasResource, "preview.2 had no resource entity")

	_, hasErrors := job["errors"]
	assert.True(t, hasErrors, "preview.2 always carried an errors field")
	assert.Nil(t, job["error_summary"], "a job that completed has no summary")
}

// A rejected row moved into results in preview.3; a pinned client must still find it in
// the separate errors array, keyed by its request index, and out of results.
func TestVersionCompat_Jobs_RowFailureStaysInTheErrorsArray(t *testing.T) {
	t.Parallel()

	// "each" is a quantity unit, so it cannot be the base of a mass group — the row is
	// rejected in the execute phase and the job still completes.
	jobID := acceptBulkUpsertUnitGroups(t, map[string]any{
		"name":      uniqueName("e2e-vc-job-bad"),
		"type":      "mass",
		"base_unit": map[string]any{"id": "each"},
	})
	pollJobUntilTerminal(t, jobID)

	job := getJobAs(t, apiClient.WithAPIVersion(preview2APIVersion), jobID)

	assert.Empty(t, jsonArray(job, "results"), "a rejected row never belonged in preview.2's results")

	errors := jsonArray(job, "errors")
	require.Len(t, errors, 1, "the rejected row belongs in preview.2's errors")
	entry, ok := errors[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0), entry["index"], "the row error keeps its request index")
	errObj, ok := entry["error"].(map[string]any)
	require.True(t, ok, "the entry wraps the canonical error object")
	assert.Contains(t, jsonField(errObj, "message"), "Base unit type does not match")
}

// The transformer must not touch the current version's responses.
func TestVersionCompat_Jobs_LatestIsUnaffected(t *testing.T) {
	t.Parallel()

	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-vc-job-latest"), uniqueName("evcjl"), "quantity"))
	job := pollJobUntilTerminal(t, jobID)
	created, _ := jobResultIDs(job)
	defer cleanupUnitIDs(created)

	assert.Equal(t, "bulk_upsert", jsonField(job, "type"))
	assert.Equal(t, "unit", jsonField(job, "resource_type"))

	results := jobResults(job)
	require.Len(t, results, 1)
	assert.Equal(t, "created", results[0]["status"])
	assert.NotEmpty(t, jobResultResourceID(results[0]))

	_, hasErrors := job["errors"]
	assert.False(t, hasErrors, "row failures live in results on the current version")
	_, hasSummary := job["error_summary"]
	assert.False(t, hasSummary, "error_summary was replaced by error")
}

// created_by is expandable now, so it is null until asked for — and a preview.2 client,
// which cannot ask, still gets the creator via the transformer's forced include.
func TestVersionCompat_Jobs_CreatedByIsGatedOnLatestButForcedForPreview2(t *testing.T) {
	t.Parallel()

	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-vc-job-cb"), uniqueName("evcjc"), "quantity"))
	job := pollJobUntilTerminal(t, jobID)
	created, _ := jobResultIDs(job)
	defer cleanupUnitIDs(created)

	assert.Nil(t, job["created_by"], "created_by is expandable, so it is null unless requested")

	status, body, err := apiClient.GetListRaw(jobsPath+"/"+jobID, map[string][]string{"include": {"created_by"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	expanded := jsonObject(parseJSON(body), "created_by")
	require.NotNil(t, expanded, "?include=created_by must expand the creator")
	assert.Equal(t, "actor", jsonField(expanded, "object"))
	assert.NotEmpty(t, jsonField(expanded, "id"))
}
