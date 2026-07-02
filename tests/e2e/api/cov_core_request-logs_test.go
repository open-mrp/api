//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes narrow field-level gaps in the already-extensive
// tests/e2e/api/crud_request_logs_test.go coverage: exact-value assertions
// on user_agent/referrer/error_message, the omitted-optional-fields-null
// case, response-shape assertions (id prefix + timestamps) against a real
// (non-seed-fixture) request log id, and 400 validation for malformed typed
// query params. See docs/patterns/e2e-test-patterns.md.

// --- allFields: exact-value assertions on previously-unasserted fields ---

func TestCovCoreRequestLogs_ExactFieldValues_UserAgent(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+SeedReqLogInfraUserID, nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "Mozilla/5.0", jsonField(got, "user_agent"))

	status, body, err = apiClient.GetListRaw(requestLogsPath+"/"+SeedReqLogInfraAgentID, nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 200, status, body)
	got = parseJSON(body)
	assert.Equal(t, "Go-http-client/1.1", jsonField(got, "user_agent"))
}

func TestCovCoreRequestLogs_ExactFieldValues_Referrer(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+SeedReqLogInfraUserID, nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedRequestLogReferrerValue, jsonField(got, "referrer"))
}

func TestCovCoreRequestLogs_ExactFieldValues_ErrorMessage(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+SeedRequestLogErrorID, nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "validation_failed", jsonField(got, "error_code"))
	assert.Equal(t, "Name is required.", jsonField(got, "error_message"))
}

// --- omittedFields: optional non-expandable fields are null when never set ---

func TestCovCoreRequestLogs_OmittedFieldsNullByDefault(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+SeedRequestLogID, nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "api_version")
	assertNilField(t, got, "client_ip")
	assertNilField(t, got, "user_agent")
	assertNilField(t, got, "referrer")
	assertNilField(t, got, "error_code")
	assertNilField(t, got, "error_message")
	assertNilField(t, got, "idempotency_key")
}

// --- responseShape: id prefix + timestamp format on a real, non-seed-fixture log ---

func TestCovCoreRequestLogs_GetResponseShape_RealID(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-rqlog-shape")
	idemKey := newIdempotencyKey()
	createBody := map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}
	status, respBody, err := apiClient.Post(itemCategoriesPath, createBody, idemKey)
	require.NoError(t, err)
	skipOnNonClientError(t, itemCategoriesPath, status)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	createdID := jsonField(created, "id")
	require.NotEmpty(t, createdID)
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + createdID) })

	var logID string
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		list, _, err := apiClient.GetList(requestLogsPath, url.Values{
			"idempotency_key": {idemKey},
			"limit":           {"1"},
		})
		if err != nil {
			return err
		}
		if len(list.Data) == 0 {
			return fmt.Errorf("no request log yet for idempotency key %s", idemKey)
		}
		m := parseJSON(list.Data[0])
		logID = jsonField(m, "id")
		if logID == "" {
			return fmt.Errorf("log item missing id")
		}
		return nil
	})
	require.NotEmpty(t, logID)
	assertIDFormat(t, logID, "rq")

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+logID, nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, logID, jsonField(got, "id"))
	assertObjectField(t, got, "request_log")
	assertValidTimestamp(t, jsonField(got, "occurred_at"), "occurred_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
}

// --- validation: malformed typed query params return 400, never 5xx ---

func TestCovCoreRequestLogs_ListInvalidStartDate(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath, url.Values{"start_date": {"not-a-date"}})
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "start_date")
}

func TestCovCoreRequestLogs_ListInvalidEndDate(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath, url.Values{"end_date": {"not-a-date"}})
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "end_date")
}

func TestCovCoreRequestLogs_ListInvalidStatusCodes(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath, url.Values{"status_codes": {"abc"}})
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "status_codes")
}

func TestCovCoreRequestLogs_ListInvalidMinLatencyUs(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(requestLogsPath, url.Values{"min_latency_us": {"abc"}})
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "min_latency_us")
}
