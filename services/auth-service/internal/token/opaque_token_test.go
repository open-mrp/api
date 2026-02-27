package token

import (
	"context"
	"testing"
)

func TestGenOpaqueToken_Success(t *testing.T) {
	token, err := GenOpaqueToken(context.Background())
	if err != nil {
		t.Fatalf("GenOpaqueToken() unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("GenOpaqueToken() returned empty token")
	}
	if len(token) != 32 {
		t.Fatalf("expected token length 32, got %d", len(token))
	}
}

func TestGenOpaqueToken_Uniqueness(t *testing.T) {
	token1, err := GenOpaqueToken(context.Background())
	if err != nil {
		t.Fatalf("GenOpaqueToken() first call unexpected error: %v", err)
	}
	token2, err := GenOpaqueToken(context.Background())
	if err != nil {
		t.Fatalf("GenOpaqueToken() second call unexpected error: %v", err)
	}
	if token1 == token2 {
		t.Fatal("expected different tokens across successive generations")
	}
}
