package apikey

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAPIKey_JSONExcludesSensitiveFields(t *testing.T) {
	t.Parallel()
	key := &APIKey{
		ID:             1,
		KeyID:          "secret-key-id",
		TypeID:         "ak_01gf7a8200eaj8fke1xvw4h50x",
		Name:           "Test Key",
		SecretHash:     []byte("super-secret-hash"),
		OwnerAccountID: "acct_123",
		RoleID:         "rl_123",
		RoleName:       "Admin",
		RoleType:       "admin",
		RedactedValue:  "mrp_sk_prod_****kuIb",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	data, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("failed to marshal APIKey: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Sensitive fields must NOT be present
	for _, field := range []string{"KeyID", "key_id", "SecretHash", "secret_hash"} {
		if _, ok := raw[field]; ok {
			t.Errorf("sensitive field %q should not be present in JSON output", field)
		}
	}

	// Non-sensitive fields MUST be present
	for _, field := range []string{"TypeID", "Name", "RedactedValue"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("expected field %q to be present in JSON output", field)
		}
	}
}
