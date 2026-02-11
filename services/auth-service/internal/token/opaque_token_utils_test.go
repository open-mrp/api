package token

import (
	"context"
	"encoding/hex"
	"testing"
)

func TestOpaqueTokenUtils_GenSuccess(t *testing.T) {
	token, err := GenOpaqueToken(context.Background())
	if err != nil {
		t.Fatalf("Gen() unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("Gen() returned empty token")
	}
	if len(token) != 64 {
		t.Fatalf("expected hex token length 64, got %d", len(token))
	}

	raw, decodeErr := hex.DecodeString(token)
	if decodeErr != nil {
		t.Fatalf("token is not valid hex: %v", decodeErr)
	}
	if len(raw) != 32 {
		t.Fatalf("expected 32 raw bytes, got %d", len(raw))
	}
}

func TestOpaqueTokenUtils_GenProducesUniqueTokens(t *testing.T) {
	token1, err := GenOpaqueToken(context.Background())
	if err != nil {
		t.Fatalf("Gen() first call unexpected error: %v", err)
	}
	token2, err := GenOpaqueToken(context.Background())
	if err != nil {
		t.Fatalf("Gen() second call unexpected error: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected different tokens across successive generations")
	}
}
