//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

const feedbackReportsPath = "/v1/core/feedback-reports"

// --- CRUD ---

func TestFeedbackReports_CreateResponseShape(t *testing.T) {
	t.Parallel()
	message := uniqueName("e2e-feedback-report")

	status, body, err := loginAsSeedUser(t).Post(feedbackReportsPath, map[string]any{
		"message":  message,
		"page":     "/dashboard/items",
		"severity": "urgent",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	report := parseJSON(body)
	assertIDFormat(t, jsonField(report, "id"), id.FeedbackReportIDPrefix)
	assertObjectField(t, report, "feedback_report")
	assert.Equal(t, message, jsonField(report, "message"))
	assert.Equal(t, "/dashboard/items", jsonField(report, "page"))
	assert.Equal(t, "urgent", jsonField(report, "severity"))
	assertNilField(t, report, "user")
	assertValidTimestamp(t, jsonField(report, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(report, "updated_at"), "updated_at")
}

func TestFeedbackReports_OmittedFields(t *testing.T) {
	t.Parallel()
	message := uniqueName("e2e-feedback-report-no-page")

	status, body, err := loginAsSeedUser(t).Post(feedbackReportsPath, map[string]any{
		"message": message,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	report := parseJSON(body)
	assert.Equal(t, message, jsonField(report, "message"))
	assertNilField(t, report, "page")
	assert.Equal(t, "medium", jsonField(report, "severity"), "severity must default to medium when omitted")
}

func TestFeedbackReports_SeverityValues(t *testing.T) {
	t.Parallel()
	user := loginAsSeedUser(t)

	for _, severity := range []string{"urgent", "medium", "low"} {
		message := uniqueName("e2e-feedback-report-" + severity)

		status, body, err := user.Post(feedbackReportsPath, map[string]any{
			"message":  message,
			"severity": severity,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		created := parseJSON(body)
		assert.Equal(t, severity, jsonField(created, "severity"))

		found := listFindByField(t, feedbackReportsPath, nil, "id", jsonField(created, "id"))
		require.NotNil(t, found, "the %s report must be listed", severity)
		assert.Equal(t, severity, DataItemField(found, "severity"))
	}
}

func TestFeedbackReports_InvalidSeverity(t *testing.T) {
	t.Parallel()
	user := loginAsSeedUser(t)

	status, body, err := user.Post(feedbackReportsPath, map[string]any{
		"message":  uniqueName("e2e-feedback-report-bad-severity"),
		"severity": "critical",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "severity")

	status, body, err = user.Post(feedbackReportsPath, map[string]any{
		"message":  uniqueName("e2e-feedback-report-null-severity"),
		"severity": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestFeedbackReports_MissingMessage(t *testing.T) {
	t.Parallel()
	user := loginAsSeedUser(t)

	status, body, err := user.Post(feedbackReportsPath, map[string]any{
		"page": "/dashboard/items",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "message")

	status, body, err = user.Post(feedbackReportsPath, map[string]any{
		"message": "   ",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestFeedbackReports_Idempotency(t *testing.T) {
	t.Parallel()
	user := loginAsSeedUser(t)
	key := newIdempotencyKey()
	payload := map[string]any{"message": uniqueName("e2e-feedback-report-idem")}

	status, body, err := user.Post(feedbackReportsPath, payload, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	first := jsonField(parseJSON(body), "id")

	status, body, err = user.Post(feedbackReportsPath, payload, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	assert.Equal(t, first, jsonField(parseJSON(body), "id"), "the same key must replay the same report")
}

// An API key has no user behind it, so it cannot submit feedback.
func TestFeedbackReports_CreateRequiresAUser(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(feedbackReportsPath, map[string]any{
		"message": uniqueName("e2e-feedback-report-key"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
}

// --- List ---

func TestFeedbackReports_List(t *testing.T) {
	t.Parallel()
	message := uniqueName("e2e-feedback-report-list")

	status, body, err := loginAsSeedUser(t).Post(feedbackReportsPath, map[string]any{
		"message": message,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	reportID := jsonField(parseJSON(body), "id")

	status, body, err = apiClient.GetListRaw(feedbackReportsPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	list := parseJSON(body)
	assertObjectField(t, list, "list")
	require.NotNil(t, list["page_info"], "page_info must be present: %s", string(body))

	found := listFindByField(t, feedbackReportsPath, nil, "id", reportID)
	require.NotNil(t, found, "the new report must be listed")
	assert.Equal(t, message, DataItemField(found, "message"))
	assert.Equal(t, "medium", DataItemField(found, "severity"))
}

// --- Expandable Fields ---

func TestFeedbackReports_UserExpandable(t *testing.T) {
	t.Parallel()
	user := loginAsSeedUser(t)
	message := uniqueName("e2e-feedback-report-user")

	status, body, err := user.Post(feedbackReportsPath, map[string]any{
		"message": message,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	reportID := jsonField(parseJSON(body), "id")

	found := listFindByField(t, feedbackReportsPath, url.Values{"include": {"user"}}, "id", reportID)
	require.NotNil(t, found, "the new report must be listed")
	report := parseJSON(found)
	expanded := jsonObject(report, "user")
	require.NotNil(t, expanded, "user must be expanded when included: %s", string(found))
	assert.Equal(t, SeedUserID, jsonField(expanded, "id"))
	assertObjectField(t, expanded, "user")

	found = listFindByField(t, feedbackReportsPath, nil, "id", reportID)
	require.NotNil(t, found)
	assertNilField(t, parseJSON(found), "user")
}
