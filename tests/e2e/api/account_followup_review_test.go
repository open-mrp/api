//go:build e2e

package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountFollowupReviewPath = "/account-followups/review"

// plantFollowupForReview inserts a follow-up as drafting would leave it, awaiting review behind token.
// Drafting calls an LLM, so the e2e stack does not run it; the review flow starts from its result.
func plantFollowupForReview(t *testing.T, token string, expiresIn time.Duration) (id, email string) {
	t.Helper()
	suffix := uuid.New().String()[:12]
	id = "acfu_e2e" + suffix
	accountID := "ac_e2e_followup_" + suffix
	email = "e2e-followup-" + suffix + "@customer-example.org"

	_, err := authDB(t).Exec(`
		INSERT INTO account_followup (
			id, account_id, user_id, account_name, registrant_name, registrant_email, registered_at,
			status, scheduled_for, attempts, activity_summary, engagement, internal_summary,
			draft_subject, draft_body, review_token_hash, review_token_expires_at, drafted_at
		) VALUES (
			?, ?, 'us_e2e_followup', 'E2E Follow-up Co', 'Sam Example', ?, NOW(3) - INTERVAL 1 DAY,
			'pending_review', NOW(3), 1, '{"total_requests":1,"sessions":[],"created":[{"resource":"operations/locations","count":2}]}', 'medium', 'Set up two locations.',
			'your openmrp setup', 'Hi Sam,\n\nThanks for trying OpenMRP.', UNHEX(SHA2(?, 256)), NOW(3) + INTERVAL ? SECOND, NOW(3)
		)`, id, accountID, email, token, int(expiresIn.Seconds()))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = authDB(t).Exec("DELETE FROM account_followup WHERE id = ?", id)
	})
	return id, email
}

// reviewRequest calls the review page without following redirects, so the post/redirect/get shape is visible.
func reviewRequest(t *testing.T, method string, query url.Values, form url.Values) (int, string, http.Header) {
	t.Helper()
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	target := apiClient.baseURL + accountFollowupReviewPath
	if query != nil {
		target += "?" + query.Encode()
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, target, body)
	require.NoError(t, err)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(raw), resp.Header
}

func followupState(t *testing.T, id string) (status, finalSubject, finalBody string) {
	t.Helper()
	var subject, body *string
	require.NoError(t, authDB(t).QueryRow("SELECT status, final_subject, final_body FROM account_followup WHERE id = ?", id).Scan(&status, &subject, &body))
	if subject != nil {
		finalSubject = *subject
	}
	if body != nil {
		finalBody = *body
	}
	return status, finalSubject, finalBody
}

type queuedEmail struct {
	To         []string `json:"to"`
	Subject    string   `json:"subject"`
	From       *string  `json:"from"`
	Bcc        []string `json:"bcc"`
	ThreadNote *struct {
		To []string `json:"to"`
	} `json:"thread_note"`
}

// followupEmailsQueued returns the send-email commands platform-service enqueued to the registrant. Outbox payloads carry the email base64-encoded, so they are decoded here rather than matched in SQL.
func followupEmailsQueued(t *testing.T, registrantEmail string) []queuedEmail {
	t.Helper()
	rows, err := authDB(t).Query(
		"SELECT payload FROM message_outbox WHERE service_name = 'platform-service' AND routing_key = 'notification.cmd.send_email' AND created_at > NOW(3) - INTERVAL 15 MINUTE",
	)
	require.NoError(t, err)
	defer rows.Close()

	var emails []queuedEmail
	for rows.Next() {
		var raw []byte
		require.NoError(t, rows.Scan(&raw))
		var msg struct {
			Data []byte `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &msg))
		var email queuedEmail
		if json.Unmarshal(msg.Data, &email) == nil && slices.Contains(email.To, registrantEmail) {
			emails = append(emails, email)
		}
	}
	require.NoError(t, rows.Err())
	return emails
}

func TestAccountFollowupReview_ApproveSendsOnce(t *testing.T) {
	t.Parallel()
	token := "e2e-review-" + uuid.New().String()
	id, email := plantFollowupForReview(t, token, time.Hour)

	// Opening the link shows the draft and changes nothing.
	status, page, header := reviewRequest(t, http.MethodGet, url.Values{"token": {token}}, nil)
	require.Equal(t, http.StatusOK, status, page)
	assert.Contains(t, page, "Sam Example")
	assert.Contains(t, page, "Set up two locations.")
	assert.Contains(t, page, "Created: operations/locations (2)")
	assert.Contains(t, page, `value="your openmrp setup"`)
	assert.Equal(t, "no-referrer", header.Get("Referrer-Policy"))
	followupStatus, _, _ := followupState(t, id)
	assert.Equal(t, "pending_review", followupStatus)
	assert.Empty(t, followupEmailsQueued(t, email))

	// Approving with edits sends the edited text and redirects to the result.
	approve := url.Values{"token": {token}, "action": {"approve"}, "subject": {"edited subject"}, "body": {"Hi Sam,\r\n\r\nEdited body."}}
	status, page, header = reviewRequest(t, http.MethodPost, nil, approve)
	require.Equal(t, http.StatusSeeOther, status, page)
	assert.Equal(t, accountFollowupReviewPath+"?token="+url.QueryEscape(token), header.Get("Location"))

	followupStatus, finalSubject, finalBody := followupState(t, id)
	assert.Equal(t, "sent", followupStatus)
	assert.Equal(t, "edited subject", finalSubject)
	assert.Equal(t, "Hi Sam,\n\nEdited body.", finalBody)
	emails := followupEmailsQueued(t, email)
	require.Len(t, emails, 1)
	assert.Equal(t, "edited subject", emails[0].Subject)
	require.NotNil(t, emails[0].From)
	assert.Contains(t, *emails[0].From, "@openmrp.ai")
	assert.NotEmpty(t, emails[0].Bcc, "the reviewer keeps a copy")
	require.NotNil(t, emails[0].ThreadNote)
	assert.Equal(t, emails[0].Bcc, emails[0].ThreadNote.To, "the context note lands in the thread the reviewer's copy started")

	// A second submit (double click, second tab) sends nothing more.
	status, page, _ = reviewRequest(t, http.MethodPost, nil, approve)
	require.Equal(t, http.StatusSeeOther, status, page)
	assert.Len(t, followupEmailsQueued(t, email), 1)

	// The link now shows what went out, with no form to send it again.
	status, page, _ = reviewRequest(t, http.MethodGet, url.Values{"token": {token}}, nil)
	require.Equal(t, http.StatusOK, status, page)
	assert.Contains(t, page, "Edited body.")
	assert.NotContains(t, page, "<form")
}

func TestAccountFollowupReview_Skip(t *testing.T) {
	t.Parallel()
	token := "e2e-review-" + uuid.New().String()
	id, email := plantFollowupForReview(t, token, time.Hour)

	status, page, _ := reviewRequest(t, http.MethodPost, nil, url.Values{"token": {token}, "action": {"skip"}})
	require.Equal(t, http.StatusSeeOther, status, page)

	followupStatus, _, _ := followupState(t, id)
	assert.Equal(t, "skipped", followupStatus)
	assert.Empty(t, followupEmailsQueued(t, email))
}

func TestAccountFollowupReview_InvalidInputKeepsEdits(t *testing.T) {
	t.Parallel()
	token := "e2e-review-" + uuid.New().String()
	id, email := plantFollowupForReview(t, token, time.Hour)

	status, page, _ := reviewRequest(t, http.MethodPost, nil, url.Values{"token": {token}, "action": {"approve"}, "subject": {"line one\nline two"}, "body": {"my edited body"}})
	require.Equal(t, http.StatusUnprocessableEntity, status, page)
	assert.Contains(t, page, "Subject must be a single line")
	assert.Contains(t, page, "my edited body")

	followupStatus, _, _ := followupState(t, id)
	assert.Equal(t, "pending_review", followupStatus)
	assert.Empty(t, followupEmailsQueued(t, email))
}

func TestAccountFollowupReview_BadLinks(t *testing.T) {
	t.Parallel()

	t.Run("unknown token", func(t *testing.T) {
		t.Parallel()
		status, page, _ := reviewRequest(t, http.MethodGet, url.Values{"token": {"no-such-token-" + uuid.New().String()}}, nil)
		require.Equal(t, http.StatusNotFound, status, page)
		assert.Contains(t, page, "This review link is invalid.")
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()
		token := "e2e-review-" + uuid.New().String()
		id, email := plantFollowupForReview(t, token, -time.Minute)

		status, page, _ := reviewRequest(t, http.MethodGet, url.Values{"token": {token}}, nil)
		require.Equal(t, http.StatusGone, status, page)
		assert.Contains(t, page, "This review link has expired.")

		status, page, _ = reviewRequest(t, http.MethodPost, nil, url.Values{"token": {token}, "action": {"approve"}, "subject": {"s"}, "body": {"b"}})
		require.Equal(t, http.StatusGone, status, page)
		followupStatus, _, _ := followupState(t, id)
		assert.Equal(t, "pending_review", followupStatus)
		assert.Empty(t, followupEmailsQueued(t, email))
	})
}
