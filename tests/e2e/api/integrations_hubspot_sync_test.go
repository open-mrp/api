//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HubSpot backfill sync coverage (/v1/settings/integrations/hubspot/sync).
//
// The seed (0014_e2e_extras.sql) provides one review_pending job
// (SeedHubspotSyncJobID) with one pending company review
// (SeedHubspotCompanyReviewID). The endpoints mutate DB state synchronously and
// only publish the preview/execute commands to an async worker, so the whole
// area is testable without live HubSpot credentials.
//
// Resolving the single seeded review is destructive and `make e2e` reuses the
// seeded DB across runs, so this suite is written to be re-run tolerant: it
// derives expectations from the observed state rather than assuming the review
// starts pending or the job starts in-flight. Read-only assertions run first,
// then the (idempotent-outcome) link mutation, then execute.

const hubspotSyncPath = "/v1/settings/integrations/hubspot/sync"

// hubspotInFlightStatuses are the job statuses that block starting a new sync.
var hubspotInFlightStatuses = map[string]bool{
	"previewing":     true,
	"review_pending": true,
	"executing":      true,
}

// getHubspotJob fetches a sync job by id and returns (parsed, statusCode).
func getHubspotJob(t *testing.T, c *Client, jobID string) (map[string]any, int) {
	t.Helper()
	resp, err := c.GetFull(hubspotSyncPath+"/"+jobID, nil)
	require.NoError(t, err)
	return parseJSON(resp.Body), resp.StatusCode
}

// hubspotReviewByID scans the job's review list (optionally status-filtered) and
// returns the review with the given id, or nil.
func hubspotReviewByID(t *testing.T, c *Client, jobID, reviewID string, status string) map[string]any {
	t.Helper()
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	list, code, err := c.GetList(hubspotSyncPath+"/"+jobID+"/company-reviews", params)
	require.NoError(t, err)
	require.Equal(t, 200, code, "listing company reviews")
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == reviewID {
			return parseJSON(raw)
		}
	}
	return nil
}

func TestHubspotSync_ReadThenResolve(t *testing.T) {
	// Sequential (no t.Parallel): the subtests share and mutate the single seeded
	// review/job, so they must run in order.

	t.Run("get-current-returns-seed-job", func(t *testing.T) {
		resp, err := apiClient.GetFull(hubspotSyncPath+"/current", nil)
		require.NoError(t, err)
		requireStatus(t, 200, resp.StatusCode, resp.Body)
		job := parseJSON(resp.Body)
		assert.Equal(t, SeedHubspotSyncJobID, jsonField(job, "id"),
			"current sync is the seeded in-flight job (no new job is ever created for the seed account)")
		assertObjectField(t, job, "hubspot_sync_job")
		assertValidTimestamp(t, jsonField(job, "created_at"), "created_at")

		// The preview report from the seed is surfaced and well-formed.
		report := jsonObject(job, "report")
		require.NotNil(t, report, "the seeded review_pending job carries a preview report")
		assertObjectField(t, report, "hubspot_sync_report")
		assert.Equal(t, "120", jsonField(report, "customers_total"))
	})

	t.Run("get-job-by-id", func(t *testing.T) {
		job, code := getHubspotJob(t, apiClient, SeedHubspotSyncJobID)
		requireStatus(t, 200, code, nil)
		assert.Equal(t, SeedHubspotSyncJobID, jsonField(job, "id"))
		assertObjectField(t, job, "hubspot_sync_job")
	})

	t.Run("get-unknown-job-404", func(t *testing.T) {
		_, code := getHubspotJob(t, apiClient, "igjb_doesnotexist0000")
		assert.Equal(t, 404, code, "unknown sync job should 404")
	})

	t.Run("list-reviews-contains-seed-and-filters", func(t *testing.T) {
		// Unfiltered list contains the seed review.
		review := hubspotReviewByID(t, apiClient, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID, "")
		require.NotNil(t, review, "the company-review queue contains the seeded review")
		assertObjectField(t, review, "hubspot_company_review")
		// The review embeds its parent job and the matched customer as nested
		// objects (mirrored in the SDK/frontend), not flat *_id reference fields.
		assert.Equal(t, SeedHubspotSyncJobID, jsonField(jsonObject(review, "job"), "id"))
		assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(review, "customer"), "id"))
		// Candidate matches are surfaced as a nested list.
		candidates, ok := listData(review, "candidates")
		require.True(t, ok, "review carries a candidates list")
		assert.NotEmpty(t, candidates, "the seeded review has at least one candidate company")

		// Status filtering is consistent with the review's current status,
		// whatever it is on this run.
		current := jsonField(review, "status")
		require.Contains(t, []string{"pending", "resolved", "skipped"}, current)
		assert.NotNil(t, hubspotReviewByID(t, apiClient, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID, current),
			"filtering by the review's own status (%s) includes it", current)
		for _, other := range []string{"pending", "resolved", "skipped"} {
			if other == current {
				continue
			}
			assert.Nil(t, hubspotReviewByID(t, apiClient, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID, other),
				"filtering by a different status (%s) excludes the %s review", other, current)
		}
	})

	t.Run("list-reviews-unknown-job-404", func(t *testing.T) {
		code, body, err := apiClient.GetListRaw(hubspotSyncPath+"/igjb_doesnotexist0000/company-reviews", nil)
		require.NoError(t, err)
		assert.Equal(t, 404, code, "listing reviews for an unknown job should 404 (job ownership check): %s", string(body))
	})

	t.Run("start-while-in-flight-conflicts", func(t *testing.T) {
		job, _ := getHubspotJob(t, apiClient, SeedHubspotSyncJobID)
		if !hubspotInFlightStatuses[jsonField(job, "status")] {
			t.Skipf("seed job is %q (not in-flight on this run); skipping conflict assertion to avoid creating a second job", jsonField(job, "status"))
		}
		status, body, err := apiClient.Post(hubspotSyncPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		requireErrorResponse(t, body, "", "invalid_request_error")
	})

	t.Run("execute-with-pending-reviews-rejected", func(t *testing.T) {
		// Only meaningful while a pending review still blocks the sync.
		if hubspotReviewByID(t, apiClient, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID, "pending") == nil {
			t.Skip("no pending review on this run; covered by the resolve+execute subtests below")
		}
		status, body, err := apiClient.Post(hubspotSyncPath+"/"+SeedHubspotSyncJobID+"/actions/execute", map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		requireErrorResponse(t, body, "", "invalid_request_error")
	})

	t.Run("link-missing-id-rejected", func(t *testing.T) {
		// Gateway validation (resolved_hubspot_id is required) rejects before any
		// mutation, so this is safe to run on the still-pending review.
		path := fmt.Sprintf("%s/%s/company-reviews/%s/actions/link", hubspotSyncPath, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID)
		status, body, err := apiClient.Post(path, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		if errObj["param"] != nil {
			assert.Equal(t, "resolved_hubspot_id", errObj["param"], "validation error should name the missing field")
		}
	})

	t.Run("resolve-actions-on-unknown-review-404", func(t *testing.T) {
		unknown := "igrv_doesnotexist0000"
		skipPath := fmt.Sprintf("%s/%s/company-reviews/%s/actions/skip", hubspotSyncPath, SeedHubspotSyncJobID, unknown)
		status, body, err := apiClient.Post(skipPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status, "skipping an unknown review should 404: %s", string(body))

		createPath := fmt.Sprintf("%s/%s/company-reviews/%s", hubspotSyncPath, SeedHubspotSyncJobID, unknown)
		status, body, err = apiClient.Post(createPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status, "create-new on an unknown review should 404: %s", string(body))
	})

	t.Run("cross-account-isolation", func(t *testing.T) {
		// Tenant B cannot see the seed account's job; the job's existence is not leaked.
		tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
		_, code := getHubspotJob(t, tenantB, SeedHubspotSyncJobID)
		assert.Contains(t, []int{403, 404}, code, "another tenant must not retrieve the seed account's sync job, got %d", code)

		resp, err := tenantB.GetFull(hubspotSyncPath+"/current", nil)
		require.NoError(t, err)
		assert.Contains(t, []int{403, 404}, resp.StatusCode,
			"tenant B has not started a sync, so current should 403/404, got %d: %s", resp.StatusCode, string(resp.Body))
	})

	t.Run("link-resolves-review", func(t *testing.T) {
		// Link always sets the review to resolved regardless of its prior state
		// (the service overwrites resolution), so the outcome is deterministic.
		path := fmt.Sprintf("%s/%s/company-reviews/%s/actions/link", hubspotSyncPath, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID)
		status, body, err := apiClient.Post(path, map[string]any{"resolved_hubspot_id": "hs_company_1001"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		review := parseJSON(body)
		assertObjectField(t, review, "hubspot_company_review")
		assert.Equal(t, "resolved", jsonField(review, "status"))
		assert.Equal(t, "link", jsonField(review, "resolution"))
		assert.Equal(t, "hs_company_1001", jsonField(review, "resolved_hubspot_id"))

		// The list now reflects the resolution.
		assert.NotNil(t, hubspotReviewByID(t, apiClient, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID, "resolved"),
			"the resolved review appears under status=resolved")
		assert.Nil(t, hubspotReviewByID(t, apiClient, SeedHubspotSyncJobID, SeedHubspotCompanyReviewID, "pending"),
			"the resolved review no longer appears under status=pending")
	})

	t.Run("execute-after-reviews-resolved", func(t *testing.T) {
		job, _ := getHubspotJob(t, apiClient, SeedHubspotSyncJobID)
		jobStatus := jsonField(job, "status")
		status, body, err := apiClient.Post(hubspotSyncPath+"/"+SeedHubspotSyncJobID+"/actions/execute", map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)

		// With all reviews resolved, execute is accepted only when the job is
		// awaiting review or retrying a failed run; otherwise (already executing/
		// completed on a re-run) it is rejected. Either way it must not 5xx.
		if jobStatus == "review_pending" || jobStatus == "failed" {
			requireStatus(t, 200, status, body)
			assertObjectField(t, parseJSON(body), "hubspot_sync_job")
		} else {
			requireStatus(t, 400, status, body)
			requireErrorResponse(t, body, "", "invalid_request_error")
		}
	})

	t.Run("list-sync-records-returns-seed-mapping", func(t *testing.T) {
		list, code, err := apiClient.GetList(hubspotSyncPath+"/records", url.Values{"augno_type": {"customer"}})
		require.NoError(t, err)
		require.Equal(t, 200, code, "listing sync records")

		var seedRecord map[string]any
		for _, raw := range list.Data {
			rec := parseJSON(raw)
			assertObjectField(t, rec, "hubspot_sync_record")
			if jsonField(rec, "id") == SeedHubspotSyncRecordID {
				seedRecord = rec
			}
		}
		require.NotNil(t, seedRecord, "the seeded sync record is listed")
		assert.Equal(t, "customer", jsonField(seedRecord, "augno_type"))
		assert.Equal(t, "companies", jsonField(seedRecord, "hubspot_type"))
		assert.Equal(t, "hs_company_1001", jsonField(seedRecord, "hubspot_id"))
		// augno_name is resolved by joining the mapped customer, not stored on the record.
		assert.NotEmpty(t, jsonField(seedRecord, "augno_name"), "the mapped customer's name is resolved")
	})

	t.Run("list-sync-records-cross-account-isolation", func(t *testing.T) {
		tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
		list, code, err := tenantB.GetList(hubspotSyncPath+"/records", url.Values{"augno_type": {"customer"}})
		require.NoError(t, err)
		require.Equal(t, 200, code, "tenant B lists its own (empty) records, never the seed account's")
		for _, raw := range list.Data {
			assert.NotEqual(t, SeedHubspotSyncRecordID, DataItemField(raw, "id"),
				"tenant B must not see the seed account's sync record")
		}
	})
}
