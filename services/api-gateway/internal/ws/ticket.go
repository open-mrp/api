package ws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// wsTicketTTL bounds how long a minted ticket can be redeemed. Tickets exist only to carry an already-validated identity across the cross-origin WebSocket handshake from custom portal domains (where the auth cookie is not sent), so the window is deliberately tight.
const wsTicketTTL = 60 * time.Second

type ticketPayload struct {
	UserID    string `json:"uid"`
	AccountID string `json:"aid"`
	ExpiresAt int64  `json:"exp"`
}

// MintTicket signs a short-lived WebSocket connection ticket for an already-authenticated user/account pair.
func MintTicket(secret []byte, userID, accountID string, now time.Time) (ticket string, expiresAt time.Time, err error) {
	expiresAt = now.Add(wsTicketTTL)
	payload, err := json.Marshal(ticketPayload{UserID: userID, AccountID: accountID, ExpiresAt: expiresAt.Unix()})
	if err != nil {
		return "", time.Time{}, err
	}

	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + signTicket(secret, encoded), expiresAt, nil
}

// VerifyTicket validates a ticket's signature and expiry and returns the identity it carries.
func VerifyTicket(secret []byte, ticket string, now time.Time) (userID, accountID string, err error) {
	encoded, sig, ok := strings.Cut(ticket, ".")
	if !ok {
		return "", "", fmt.Errorf("malformed ws ticket")
	}
	if !hmac.Equal([]byte(signTicket(secret, encoded)), []byte(sig)) {
		return "", "", fmt.Errorf("invalid ws ticket signature")
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", fmt.Errorf("malformed ws ticket payload: %w", err)
	}
	var payload ticketPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", fmt.Errorf("malformed ws ticket payload: %w", err)
	}
	if now.Unix() > payload.ExpiresAt {
		return "", "", fmt.Errorf("expired ws ticket")
	}
	if payload.UserID == "" || payload.AccountID == "" {
		return "", "", fmt.Errorf("incomplete ws ticket")
	}

	return payload.UserID, payload.AccountID, nil
}

func signTicket(secret []byte, encodedPayload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
