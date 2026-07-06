package ws

import (
	"strings"
	"testing"
	"time"
)

func TestWSTicketRoundTrip(t *testing.T) {
	t.Parallel()
	secret := []byte("test-secret")
	now := time.Now()

	ticket, expiresAt, err := MintTicket(secret, "us_123", "acct_456", now)
	if err != nil {
		t.Fatalf("MintTicket() error: %v", err)
	}
	if !expiresAt.After(now) {
		t.Fatalf("expiresAt %v not after now %v", expiresAt, now)
	}

	userID, accountID, err := VerifyTicket(secret, ticket, now)
	if err != nil {
		t.Fatalf("VerifyTicket() error: %v", err)
	}
	if userID != "us_123" || accountID != "acct_456" {
		t.Fatalf("VerifyTicket() = (%q, %q), want (us_123, acct_456)", userID, accountID)
	}
}

func TestWSTicketRejectsExpired(t *testing.T) {
	t.Parallel()
	secret := []byte("test-secret")
	now := time.Now()

	ticket, _, err := MintTicket(secret, "us_123", "acct_456", now)
	if err != nil {
		t.Fatalf("MintTicket() error: %v", err)
	}

	if _, _, err := VerifyTicket(secret, ticket, now.Add(wsTicketTTL+time.Second)); err == nil {
		t.Fatal("VerifyTicket() accepted an expired ticket")
	}
}

func TestWSTicketRejectsTampering(t *testing.T) {
	t.Parallel()
	now := time.Now()

	ticket, _, err := MintTicket([]byte("test-secret"), "us_123", "acct_456", now)
	if err != nil {
		t.Fatalf("MintTicket() error: %v", err)
	}

	// Wrong secret.
	if _, _, err := VerifyTicket([]byte("other-secret"), ticket, now); err == nil {
		t.Fatal("VerifyTicket() accepted a ticket signed with a different secret")
	}

	// Payload from one ticket with the signature of another.
	tampered, _, err := MintTicket([]byte("test-secret"), "us_attacker", "acct_456", now)
	if err != nil {
		t.Fatalf("MintTicket() error: %v", err)
	}
	tamperedPayload := tampered[:strings.IndexByte(tampered, '.')]
	originalSig := ticket[strings.IndexByte(ticket, '.')+1:]
	if _, _, err := VerifyTicket([]byte("test-secret"), tamperedPayload+"."+originalSig, now); err == nil {
		t.Fatal("VerifyTicket() accepted a payload/signature mismatch")
	}

	// Garbage.
	if _, _, err := VerifyTicket([]byte("test-secret"), "not-a-ticket", now); err == nil {
		t.Fatal("VerifyTicket() accepted garbage")
	}
}
