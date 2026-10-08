//go:build e2e

package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	requestDemoPath    = "/v1/core/actions/request-demo"
	submitFeedbackPath = "/v1/core/actions/submit-feedback"
)

// queuedSendEmail is the one send-email command the outbox holds whose payload contains marker.
func queuedSendEmail(t *testing.T, marker string) map[string]any {
	t.Helper()
	rows, err := authDB(t).Query(`
		SELECT CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR)
		FROM message_outbox
		WHERE id > ? AND routing_key = 'notification.cmd.send_email'
		AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR) LIKE ?`,
		outboxFloor(t), "%"+marker+"%")
	require.NoError(t, err)
	defer rows.Close()

	var emails []map[string]any
	for rows.Next() {
		var data string
		require.NoError(t, rows.Scan(&data))
		emails = append(emails, parseJSON([]byte(data)))
	}
	require.NoError(t, rows.Err())
	require.Len(t, emails, 1, "one email per submission")
	return emails[0]
}

func emailParam(t *testing.T, email map[string]any, name string) string {
	t.Helper()
	params, ok := email["params"].(map[string]any)
	require.True(t, ok, "email params: %v", email)
	value, _ := params[name].(string)
	return value
}

// postAnonymously POSTs body with no credentials and no account, as a visitor's browser would.
func postAnonymously(t *testing.T, path string, body any) (int, []byte) {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, apiClient.baseURL+path, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenMRP-Version", apiClient.apiVersion)
	req.Header.Set("Idempotency-Key", newIdempotencyKey())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, respBody
}

// A demo request comes from the marketing site, before the requester has any account.
func TestUtils_RequestDemoNeedsNoSignIn(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	company := uniqueName("e2e-demo-co")

	status, body := postAnonymously(t, requestDemoPath, map[string]any{
		"name":         "Demo Requester",
		"email":        "demo@e2e-test.openmrp.ai",
		"company":      company,
		"phone_number": "555-0100",
		"message":      "Show me scanning.",
	})
	requireStatus(t, 200, status, body)

	email := queuedSendEmail(t, company)
	assert.Equal(t, "demo_request", email["template_id"])
	assert.Equal(t, "Demo Requester", emailParam(t, email, "Name"))
	assert.Equal(t, "demo@e2e-test.openmrp.ai", emailParam(t, email, "Email"))
	assert.Equal(t, "555-0100", emailParam(t, email, "PhoneNumber"))
	assert.Equal(t, "Show me scanning.", emailParam(t, email, "Message"))
}

// Feedback names who sent it, with the address a reply goes to.
func TestUtils_SubmitFeedbackCarriesTheSendersAddress(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	answer := uniqueName("e2e-feedback")

	status, body, err := loginAsSeedUser(t).Post(submitFeedbackPath, map[string]any{
		"question": "How was this page?",
		"answer":   answer,
		"page_url": "http://localhost:3000/dashboard/items",
	}, "")
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var userEmail string
	require.NoError(t, authDB(t).QueryRow("SELECT email FROM user WHERE id = ?", SeedUserID).Scan(&userEmail))

	email := queuedSendEmail(t, answer)
	assert.Equal(t, "dashboard_feedback", email["template_id"])
	assert.Equal(t, userEmail, emailParam(t, email, "UserEmail"))
	assert.Equal(t, SeedUserID, emailParam(t, email, "ActorID"))
	assert.Equal(t, SeedAccountID, emailParam(t, email, "AccountID"))
	assert.Equal(t, "How was this page?", emailParam(t, email, "Question"))
	assert.Equal(t, "http://localhost:3000/dashboard/items", emailParam(t, email, "PageURL"))

	status, body = postAnonymously(t, submitFeedbackPath, map[string]any{
		"question": "How was this page?",
		"answer":   answer,
	})
	assert.Equal(t, 401, status, "feedback needs a signed-in sender: %s", body)
}

// An API key has no address, so its feedback goes without one.
func TestUtils_SubmitFeedbackFromAnAPIKey(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	answer := uniqueName("e2e-feedback-key")

	status, body, err := apiClient.Post(submitFeedbackPath, map[string]any{
		"question": "How was this page?",
		"answer":   answer,
	}, "")
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	email := queuedSendEmail(t, answer)
	assert.Equal(t, "", emailParam(t, email, "UserEmail"))
}
