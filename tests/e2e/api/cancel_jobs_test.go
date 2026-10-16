//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Job cancellation (POST /v1/core/async-jobs/{id}/actions/cancel) ---
//
// Cancelling an async bulk job must guarantee that the job's writes do not land: a job
// that ends up cancelled created/updated none of its rows. The mechanism is the atomic,
// terminal-guarded job transition — a cancel that beats the worker's completion forces
// the completion (which lives inside the write transaction) to match zero rows and roll
// back. The e2e stack runs the outbox at a 10ms poll, so a pre-execution cancel cannot be
// forced deterministically; instead a large batch makes the write itself slow enough that
// an immediate cancel reliably lands mid-write, exercising the guarded rollback.

// Every async job names who raised it, from the 202 that answers the request onward.
func TestJobs_CreatedByIsAlwaysPresent(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{
		"units": []any{upsertUnitInput(uniqueName("e2e-jobby"), uniqueName("jby"), "quantity")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)
	accepted := parseJSON(body)
	assertJobCreatedByHarnessKey(t, accepted)

	job := pollJobUntilTerminal(t, jsonField(accepted, "id"))
	if created, _ := jobResultIDs(job); len(created) > 0 {
		defer cleanupUnitIDs(created)
	}
	assertJobCreatedByHarnessKey(t, job)

	status, body, err = apiClient.GetListRaw(jobsPath+"/"+jsonField(job, "id"), url.Values{"include": {"created_by"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func assertJobCreatedByHarnessKey(t *testing.T, job map[string]any) {
	t.Helper()
	createdBy := jsonObject(job, "created_by")
	require.NotNil(t, createdBy, "created_by is always present: %v", job)
	assertActorRef(t, createdBy, "api_key")
	assert.Equal(t, SeedAPIKeyID, jsonField(createdBy, "id"))
	assert.NotEmpty(t, jsonField(createdBy, "name"), "the key's name is resolved")
}

// The list returns the account's jobs newest first, narrowed by type and status.
func TestJobs_ListNewestFirstWithFilters(t *testing.T) {
	t.Parallel()

	older := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-jobl"), uniqueName("jbl"), "quantity"))
	newer := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-jobl"), uniqueName("jbl"), "quantity"))
	for _, id := range []string{older, newer} {
		job := pollJobUntilTerminal(t, id)
		require.Equal(t, "completed", jsonField(job, "status"))
		if created, _ := jobResultIDs(job); len(created) > 0 {
			defer cleanupUnitIDs(created)
		}
	}

	list, status, err := apiClient.GetList(jobsPath, url.Values{"types": {"bulk_upsert"}, "statuses": {"completed"}, "limit": {"100"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	pos := map[string]int{}
	var prev time.Time
	for i, raw := range list.Data {
		job := parseJSON(raw)
		assert.Equal(t, "async_job", jsonField(job, "object"))
		assert.Equal(t, "bulk_upsert", jsonField(job, "type"))
		assert.Equal(t, "completed", jsonField(job, "status"))
		assert.NotNil(t, jsonObject(job, "created_by"), "created_by is always present")
		// Timestamps drop trailing zeros (.31Z), so they are compared as times, not strings.
		createdAt, err := time.Parse(time.RFC3339Nano, jsonField(job, "created_at"))
		require.NoError(t, err)
		if !prev.IsZero() {
			assert.False(t, createdAt.After(prev), "newest first: %s listed after %s", createdAt, prev)
		}
		prev = createdAt
		pos[jsonField(job, "id")] = i
	}
	require.Contains(t, pos, older)
	require.Contains(t, pos, newer)
	assert.Less(t, pos[newer], pos[older], "the newer job lists first")

	for _, params := range []url.Values{
		{"types": {"export"}, "limit": {"100"}},
		{"statuses": {"queued", "running", "failed", "canceled"}, "limit": {"100"}},
	} {
		other, _, err := apiClient.GetList(jobsPath, params)
		require.NoError(t, err)
		for _, raw := range other.Data {
			id := DataItemField(raw, "id")
			assert.NotEqual(t, older, id, "filter %v must exclude a completed bulk upsert", params)
			assert.NotEqual(t, newer, id, "filter %v must exclude a completed bulk upsert", params)
		}
	}
}

func TestJobs_ListPagination(t *testing.T) {
	t.Parallel()
	acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-jobp"), uniqueName("jbp"), "quantity"))
	acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-jobp"), uniqueName("jbp"), "quantity"))
	assertCursorPaginationAdvances(t, jobsPath, nil)
}

func TestJobs_ListRejectsUnknownFilterValues(t *testing.T) {
	t.Parallel()
	for _, params := range []url.Values{{"types": {"zzz"}}, {"statuses": {"started"}}} {
		status, body, err := apiClient.GetListRaw(jobsPath, params)
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	}
}

// A job that has already finished cannot be cancelled — the transition is refused.
func TestJobs_Cancel_RejectedAfterCompletion(t *testing.T) {
	t.Parallel()

	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-cxl"), uniqueName("exl"), "quantity"))
	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))

	if created, _ := jobResultIDs(job); len(created) == 1 {
		defer cleanupUnitIDs(created)
	}

	status, body, err := apiClient.Post(jobsPath+"/"+jobID+"/actions/cancel", nil, "")
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 409 || status == 422,
		"cancelling a completed job must be rejected, got %d: %s", status, string(body))
}

// Whatever the race outcome, the invariant holds: a job that ends up cancelled wrote
// nothing. A large batch widens the write window so the immediate cancel has a real chance
// of landing mid-write — the path the atomic transition guard protects against a partial
// commit. The test never fails on the race itself (a cancel that loses simply completes);
// it fails only if a cancelled job left objects behind.
func TestJobs_Cancel_CancelledJobWritesNothing(t *testing.T) {
	t.Parallel()

	const n = 50
	rows := make([]map[string]any, n)
	names := make([]string, n)
	for i := range rows {
		names[i] = uniqueName("e2e-cxlw")
		rows[i] = upsertUnitInput(names[i], uniqueName("exlw"), "quantity")
	}

	jobID := acceptBulkUpsertUnits(t, rows...)

	// Cancel immediately — before or (with a batch this size) during the write.
	_, _, err := apiClient.Post(jobsPath+"/"+jobID+"/actions/cancel", nil, "")
	require.NoError(t, err)

	job := pollJobUntilTerminal(t, jobID)
	status := jsonField(job, "status")

	// Clean up anything created regardless of which side won the race.
	created, _ := jobResultIDs(job)
	if len(created) > 0 {
		ids := append([]string{}, created...)
		defer cleanupUnitIDs(ids)
	}

	switch status {
	case "canceled":
		// The guarded completion rolled the whole write back: no ids recorded...
		assert.Empty(t, created, "a cancelled job must not have recorded created objects")
		// ...and a representative unit must genuinely not exist.
		list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {names[0]}})
		require.NoError(t, err)
		for _, item := range list.Data {
			assert.NotEqual(t, names[0], jsonField(parseJSON(item), "name"),
				"a cancelled job's units must not have been created")
		}
	case "completed":
		// The cancel lost the race; the job ran to completion. That is a valid outcome —
		// the job simply finished before the cancel landed.
	default:
		t.Fatalf("job %s ended in unexpected status %q", jobID, status)
	}
}
