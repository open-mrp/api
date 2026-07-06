package event

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInboundEmail_DecodesBase64Body(t *testing.T) {
	plain := "Begin forwarded message:\nFrom: carl@customer.com\nWhat was on my last order?"
	// Wrap the base64 at 76 columns with CRLF folding, exactly like a mail client encodes it.
	b64 := base64.StdEncoding.EncodeToString([]byte(plain))
	var wrapped strings.Builder
	for i := 0; i < len(b64); i += 76 {
		end := min(i+76, len(b64))
		wrapped.WriteString(b64[i:end])
		wrapped.WriteString("\r\n")
	}

	// Multipart forward with a base64-encoded text/plain part — the shape a forwarded email arrives in.
	raw := strings.Join([]string{
		"From: forwarder@acme.com",
		"To: support@acme.com",
		"Subject: Fwd: What was on my last order",
		"Message-ID: <b64@x>",
		"Content-Type: multipart/alternative; boundary=BOUND",
		"",
		"--BOUND",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: base64",
		"",
		wrapped.String(),
		"--BOUND--",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "k")
	require.NoError(t, err)
	assert.Contains(t, in.TextBody, "What was on my last order", "base64 body must be decoded to readable text")
	assert.Contains(t, in.TextBody, "carl@customer.com")
	assert.NotContains(t, in.TextBody, b64[:20], "the raw base64 must not survive into the body")
}

func TestParseInboundEmail_DecodesQuotedPrintableBody(t *testing.T) {
	raw := strings.Join([]string{
		"From: cust@x.com",
		"To: support@acme.com",
		"Subject: hi",
		"Message-ID: <qp@x>",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: quoted-printable",
		"",
		"Price is 40=C2=A0dollars =3D cheap=",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "k")
	require.NoError(t, err)
	assert.Contains(t, in.TextBody, "40")
	assert.Contains(t, in.TextBody, "= cheap", "quoted-printable =3D must decode to '='")
	assert.NotContains(t, in.TextBody, "=3D", "raw quoted-printable escapes must not survive")
}

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
	assert.Equal(t, []string{"support@augno-test.com"}, in.Recipients)
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

func TestParseInboundEmail_CollectsAllCandidateRecipients(t *testing.T) {
	// Every delivery/recipient header contributes a candidate, delivery headers first, de-duplicated.
	// Under forwarding the original inbox address survives only in To/Cc while a forward target lands in
	// Delivered-To, so ingestion must see them all rather than trusting one "delivered" header.
	raw := strings.Join([]string{
		"From: jane@theirco.com",
		"Delivered-To: in_01hf@inbound.augno.com",
		"To: support@acme.com, cc@elsewhere.com",
		"Cc: support@acme.com", // duplicate collapses
		"Subject: envelope recipient",
		"Message-ID: <m3@theirco.com>",
		"Content-Type: text/plain",
		"",
		"body",
		"",
	}, "\r\n")

	in, err := parseInboundEmail([]byte(raw), "inbound/d")
	require.NoError(t, err)
	assert.Equal(t, []string{
		"in_01hf@inbound.augno.com", // Delivered-To first
		"support@acme.com",          // To
		"cc@elsewhere.com",          // To
	}, in.Recipients, "delivery headers first, duplicates dropped")
}

func TestParseInboundEmail_SenderFromForwardedHeaders(t *testing.T) {
	mkRaw := func(headers ...string) []byte {
		return []byte(strings.Join(append(headers,
			"Subject: hi", "Message-ID: <m@x>", "Content-Type: text/plain", "", "body", "",
		), "\r\n"))
	}

	// Direct mail: From is the author.
	in, err := parseInboundEmail(mkRaw(`From: "Jane Doe" <jane@theirco.com>`), "k")
	require.NoError(t, err)
	assert.Equal(t, "jane@theirco.com", in.From)
	assert.Equal(t, "Jane Doe", in.FromName)

	// A mailing list / forwarder rewrote From to itself; the real author survives in X-Original-Sender.
	in, err = parseInboundEmail(mkRaw(
		"From: support@theirco.com",
		`X-Original-Sender: "Carl Customer" <carl@customer.com>`,
	), "k")
	require.NoError(t, err)
	assert.Equal(t, "carl@customer.com", in.From, "prefer the original author over the forwarder")
	assert.Equal(t, "Carl Customer", in.FromName)

	// Google consumer forwarding: original envelope sender is the first token of X-Forwarded-For.
	in, err = parseInboundEmail(mkRaw(
		"From: forwarder@gmail.com",
		"X-Forwarded-For: dana@customer.com forwarder@gmail.com",
	), "k")
	require.NoError(t, err)
	// From parses, so it wins over X-Forwarded-For — X-Forwarded-For is only a fallback when From is
	// unparseable. Assert From is honored (no original-author header present).
	assert.Equal(t, "forwarder@gmail.com", in.From)

	// From unparseable → fall back to Reply-To.
	in, err = parseInboundEmail(mkRaw(
		"From: (garbage no address)",
		"Reply-To: real@customer.com",
	), "k")
	require.NoError(t, err)
	assert.Equal(t, "real@customer.com", in.From)
}
