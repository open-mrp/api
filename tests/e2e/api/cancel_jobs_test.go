//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Job cancellation (POST /v1/core/jobs/{id}/cancel) ---
//
// Cancelling an async bulk job must guarantee that the job's writes do not land: a job
// that ends up cancelled created/updated none of its rows. The mechanism is the atomic,
// terminal-guarded job transition — a cancel that beats the worker's completion forces
// the completion (which lives inside the write transaction) to match zero rows and roll
// back. The e2e stack runs the outbox at a 10ms poll, so a pre-execution cancel cannot be
// forced deterministically; instead a large batch makes the write itself slow enough that
// an immediate cancel reliably lands mid-write, exercising the guarded rollback.

// A job that has already finished cannot be cancelled — the transition is refused.
func TestJobs_Cancel_RejectedAfterCompletion(t *testing.T) {
	t.Parallel()

	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-cxl"), uniqueName("exl"), "quantity"))
	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))

	if created := jsonArray(jsonObject(job, "results"), "created_ids"); len(created) == 1 {
		defer cleanupUnitIDs([]string{created[0].(string)})
	}

	status, body, err := apiClient.Post(jobsPath+"/"+jobID+"/cancel", nil, "")
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
	_, _, err := apiClient.Post(jobsPath+"/"+jobID+"/cancel", nil, "")
	require.NoError(t, err)

	job := pollJobUntilTerminal(t, jobID)
	status := jsonField(job, "status")

	// Clean up anything created regardless of which side won the race.
	created := jsonArray(jsonObject(job, "results"), "created_ids")
	if len(created) > 0 {
		ids := make([]string, 0, len(created))
		for _, c := range created {
			ids = append(ids, c.(string))
		}
		defer cleanupUnitIDs(ids)
	}

	switch status {
	case "cancelled":
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
