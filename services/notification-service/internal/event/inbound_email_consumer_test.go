package event

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInboundEmail_PlainText(t *testing.T) {
	raw := strings.Join([]string{
		"From: Jane Customer <jane@theirco.com>",
		"To: support@augno-test.com",
		"Subject: Where is my order?",
		"Message-ID: <abc123@theirco.com>",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Hi, I haven't received order #42.",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "inbound/x")
	require.NoError(t, err)
	assert.Equal(t, "support@augno-test.com", in.Recipient)
	assert.Equal(t, "jane@theirco.com", in.From)
	assert.Equal(t, "Jane Customer", in.FromName)
	assert.Equal(t, "Where is my order?", in.Subject)
	assert.Equal(t, "abc123@theirco.com", in.RfcMessageID, "angle brackets stripped")
	assert.Contains(t, in.TextBody, "order #42")
	assert.Equal(t, "inbound/x", in.RawS3Key)
}

func TestParseInboundEmail_MultipartPrefersPlain(t *testing.T) {
	raw := strings.Join([]string{
		"From: jane@theirco.com",
		"To: support@augno-test.com",
		"Subject: multipart",
		"Message-ID: <m1@theirco.com>",
		"Content-Type: multipart/alternative; boundary=BOUND",
		"",
		"--BOUND",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"plain body wins",
		"--BOUND",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<p>html body</p>",
		"--BOUND--",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "inbound/y")
	require.NoError(t, err)
	assert.Equal(t, "plain body wins", in.TextBody)
}

func TestParseInboundEmail_HTMLOnlyStripped(t *testing.T) {
	raw := strings.Join([]string{
		"From: jane@theirco.com",
		"To: support@augno-test.com",
		"Subject: html only",
		"Message-ID: <m2@theirco.com>",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<html><body><p>Hello <b>there</b></p><script>ignored()</script></body></html>",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "inbound/z")
	require.NoError(t, err)
	assert.Contains(t, in.TextBody, "Hello")
	assert.Contains(t, in.TextBody, "there")
	assert.NotContains(t, in.TextBody, "ignored", "script contents stripped")
	assert.NotContains(t, in.TextBody, "<", "tags stripped")
}

func TestParseInboundEmail_Threading(t *testing.T) {
	raw := strings.Join([]string{
		"From: jane@theirco.com",
		"To: support@augno-test.com",
		"Subject: Re: ticket",
		"Message-ID: <reply@theirco.com>",
		"In-Reply-To: <orig@augno-test.com>",
		"References: <root@augno-test.com> <orig@augno-test.com>",
		"Content-Type: text/plain",
		"",
		"following up",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "inbound/r")
	require.NoError(t, err)
	assert.Equal(t, "orig@augno-test.com", in.InReplyTo)
	assert.Equal(t, []string{"root@augno-test.com", "orig@augno-test.com"}, in.References)
}

func TestParseInboundEmail_DeliveredToWins(t *testing.T) {
	raw := strings.Join([]string{
		"From: jane@theirco.com",
		"Delivered-To: support@augno-test.com",
		"To: jane@theirco.com, cc@elsewhere.com",
		"Subject: envelope recipient",
		"Message-ID: <m3@theirco.com>",
		"Content-Type: text/plain",
		"",
		"body",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "inbound/d")
	require.NoError(t, err)
	assert.Equal(t, "support@augno-test.com", in.Recipient, "Delivered-To preferred over To")
}
