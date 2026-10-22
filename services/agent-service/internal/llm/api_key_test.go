package llm

import (
	"context"
	"errors"
	"testing"
)

func TestResolveAPIKey(t *testing.T) {
	if got, err := resolveAPIKey(WithAPIKey(context.Background(), "sk-or-account"), "sk-or-deployment"); err != nil || got != "sk-or-account" {
		t.Errorf("the run's key must win over the deployment's, got %q, %v", got, err)
	}
	if got, err := resolveAPIKey(context.Background(), "sk-or-deployment"); err != nil || got != "sk-or-deployment" {
		t.Errorf("a run without a key falls back to the deployment's, got %q, %v", got, err)
	}
	if _, err := resolveAPIKey(context.Background(), ""); !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("no key anywhere must be ErrNoAPIKey, got %v", err)
	}
}
