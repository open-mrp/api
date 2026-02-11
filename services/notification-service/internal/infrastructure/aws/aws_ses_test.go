package aws

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRawEmail(t *testing.T) {
	t.Run("should generate valid raw email with HTML body", func(t *testing.T) {
		subject := "Test Subject"
		body := "<h1>Hello World</h1>"
		recipients := []string{"test@example.com"}
		sender := "noreply@example.com"
		isHtml := true

		rawEmail, err := generateRawEmail(rawEmailInput{
			Subject:     subject,
			Body:        body,
			Recipients:  recipients,
			SenderEmail: sender,
			IsHtml:      isHtml,
		})
		assert.NoError(t, err)

		emailStr := string(rawEmail)
		assert.Contains(t, emailStr, "From: noreply@example.com")
		assert.Contains(t, emailStr, "To: test@example.com")
		assert.Contains(t, emailStr, "Subject: Test Subject")
		assert.Contains(t, emailStr, "Content-Type: multipart/mixed; boundary=\"NextPart\"")
		assert.Contains(t, emailStr, "Content-Type: text/html; charset=utf-8")
		assert.Contains(t, emailStr, "Content-Transfer-Encoding: base64")

		// Verify body is base64 encoded
		encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
		assert.Contains(t, emailStr, encodedBody)
	})

	t.Run("should generate valid raw email with plain text body", func(t *testing.T) {
		subject := "Test Subject"
		body := "Hello World"
		recipients := []string{"test@example.com"}
		sender := "noreply@example.com"
		isHtml := false

		rawEmail, err := generateRawEmail(rawEmailInput{
			Subject:     subject,
			Body:        body,
			Recipients:  recipients,
			SenderEmail: sender,
			IsHtml:      isHtml,
		})
		assert.NoError(t, err)

		emailStr := string(rawEmail)
		assert.Contains(t, emailStr, "Content-Type: text/plain; charset=utf-8")

		encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
		assert.Contains(t, emailStr, encodedBody)
	})

	t.Run("should include Reply-To header when provided", func(t *testing.T) {
		subject := "Test Subject"
		body := "Body"
		recipients := []string{"test@example.com"}
		sender := "noreply@example.com"
		replyTo := "support@example.com"

		rawEmail, err := generateRawEmail(rawEmailInput{
			Subject:     subject,
			Body:        body,
			Recipients:  recipients,
			SenderEmail: sender,
			IsHtml:      false,
			ReplyTo:     &replyTo,
		})
		assert.NoError(t, err)

		emailStr := string(rawEmail)
		assert.Contains(t, emailStr, "Reply-To: support@example.com")
	})

	t.Run("should handle attachments", func(t *testing.T) {
		subject := "Test Subject"
		body := "Body"
		recipients := []string{"test@example.com"}
		sender := "noreply@example.com"
		attachment := []byte("test attachment content")
		filename := "test.txt"

		rawEmail, err := generateRawEmail(rawEmailInput{
			Subject:     subject,
			Body:        body,
			Recipients:  recipients,
			Attachment:  attachment,
			Filename:    &filename,
			SenderEmail: sender,
			IsHtml:      false,
		})
		assert.NoError(t, err)

		emailStr := string(rawEmail)
		assert.Contains(t, emailStr, "Content-Type: application/octet-stream; name=\"test.txt\"")
		assert.Contains(t, emailStr, "Content-Disposition: attachment; filename=\"test.txt\"")

		encodedAttachment := base64.StdEncoding.EncodeToString(attachment)
		assert.Contains(t, emailStr, encodedAttachment)
	})

	t.Run("should split long lines in base64 encoding", func(t *testing.T) {
		// Create a long body that will exceed 76 characters when base64 encoded
		longBody := strings.Repeat("a", 100)
		// "a" * 100 base64 encoded is roughly 136 chars.
		// It should be split into 2 lines (76 + rest).

		encodedBody := base64.StdEncoding.EncodeToString([]byte(longBody))
		expectedSplit := splitString(encodedBody, 76)

		recipients := []string{"test@example.com"}
		rawEmail, err := generateRawEmail(rawEmailInput{
			Subject:     "Subject",
			Body:        longBody,
			Recipients:  recipients,
			SenderEmail: "sender",
			IsHtml:      false,
		})
		assert.NoError(t, err)

		emailStr := string(rawEmail)
		assert.Contains(t, emailStr, expectedSplit)
		// Ensure \r\n is present in the split
		assert.True(t, strings.Contains(expectedSplit, "\r\n"))
	})
}

func TestSplitString(t *testing.T) {
	t.Run("should split string into chunks", func(t *testing.T) {
		s := "abcdefghij"
		chunked := splitString(s, 3)
		expected := "abc\r\ndef\r\nghi\r\nj"
		assert.Equal(t, expected, chunked)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		chunked := splitString("", 76)
		assert.Equal(t, "", chunked)
	})

	t.Run("should handle string shorter than chunk size", func(t *testing.T) {
		s := "short"
		chunked := splitString(s, 10)
		assert.Equal(t, "short", chunked)
	})
}
