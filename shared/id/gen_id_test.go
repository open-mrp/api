package id

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name           string
		prefix         IDPrefix
		expectedPrefix string
		validateFormat func(t *testing.T, id string)
	}{
		{
			name:           "user prefix",
			prefix:         UserIDPrefix,
			expectedPrefix: "us",
			validateFormat: func(t *testing.T, id string) {
				if !strings.HasPrefix(id, "us_") {
					t.Errorf("expected ID to start with 'us_', got: %s", id)
				}
			},
		},
		{
			name:           "account prefix",
			prefix:         AccountIDPrefix,
			expectedPrefix: "ac",
			validateFormat: func(t *testing.T, id string) {
				if !strings.HasPrefix(id, "ac_") {
					t.Errorf("expected ID to start with 'ac_', got: %s", id)
				}
			},
		},
		{
			name:           "session prefix",
			prefix:         SessionIDPrefix,
			expectedPrefix: "sess",
			validateFormat: func(t *testing.T, id string) {
				if !strings.HasPrefix(id, "sess_") {
					t.Errorf("expected ID to start with 'sess_', got: %s", id)
				}
			},
		},
		{
			name:           "product prefix",
			prefix:         ProductIDPrefix,
			expectedPrefix: "pd",
			validateFormat: func(t *testing.T, id string) {
				if !strings.HasPrefix(id, "pd_") {
					t.Errorf("expected ID to start with 'pd_', got: %s", id)
				}
			},
		},
		{
			name:           "request prefix",
			prefix:         RequestIDPrefix,
			expectedPrefix: "rq",
			validateFormat: func(t *testing.T, id string) {
				if !strings.HasPrefix(id, "rq_") {
					t.Errorf("expected ID to start with 'rq_', got: %s", id)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GenID(tt.prefix, nil)
			if err != nil {
				t.Fatalf("GenID() returned error: %v", err)
			}

			if id == "" {
				t.Error("GenID() returned empty string")
			}

			// Validate format: prefix_baseID
			parts := strings.Split(id, "_")
			if len(parts) != 2 {
				t.Errorf("expected ID format 'prefix_baseID', got: %s", id)
			}

			if parts[0] != string(tt.expectedPrefix) {
				t.Errorf("expected prefix '%s', got: %s", tt.expectedPrefix, parts[0])
			}

			baseID := parts[1]
			if len(baseID) != 12 {
				t.Errorf("expected baseID length 12, got: %d", len(baseID))
			}

			// Validate baseID contains only valid characters (0-9, a-z)
			validPattern := regexp.MustCompile(`^[0-9a-z]{12}$`)
			if !validPattern.MatchString(baseID) {
				t.Errorf("baseID contains invalid characters, got: %s", baseID)
			}

			if tt.validateFormat != nil {
				tt.validateFormat(t, id)
			}
		})
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	prefix := UserIDPrefix
	generated := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		id, err := GenID(prefix, nil)
		if err != nil {
			t.Fatalf("GenID() returned error: %v", err)
		}

		if generated[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		generated[id] = true
	}

	if len(generated) != iterations {
		t.Errorf("expected %d unique IDs, got: %d", iterations, len(generated))
	}
}

func TestGenerateID_DifferentPrefixes(t *testing.T) {
	prefixes := []IDPrefix{
		UserIDPrefix,
		AccountIDPrefix,
		SessionIDPrefix,
		ProductIDPrefix,
	}

	ids := make([]string, len(prefixes))
	for i, prefix := range prefixes {
		id, err := GenID(prefix, nil)
		if err != nil {
			t.Fatalf("GenID() returned error for prefix %s: %v", prefix, err)
		}
		ids[i] = id
	}

	// Verify all IDs are different
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[i] == ids[j] {
				t.Errorf("duplicate ID generated for different prefixes: %s", ids[i])
			}
		}
	}

	// Verify each ID has the correct prefix
	for i, prefix := range prefixes {
		expectedPrefix := string(prefix) + "_"
		if !strings.HasPrefix(ids[i], expectedPrefix) {
			t.Errorf("ID %s does not have expected prefix %s", ids[i], expectedPrefix)
		}
	}
}

func TestGenerateID_Format(t *testing.T) {
	prefix := UserIDPrefix
	id, err := GenID(prefix, nil)
	if err != nil {
		t.Fatalf("GenID() returned error: %v", err)
	}

	// Verify format: prefix_baseID
	if !strings.Contains(id, "_") {
		t.Error("ID must contain underscore separator")
	}

	parts := strings.Split(id, "_")
	if len(parts) != 2 {
		t.Errorf("expected exactly one underscore, got: %s", id)
	}

	prefixPart := parts[0]
	baseIDPart := parts[1]

	if prefixPart != string(prefix) {
		t.Errorf("expected prefix part '%s', got: %s", prefix, prefixPart)
	}

	if len(baseIDPart) != 12 {
		t.Errorf("expected baseID length 12, got: %d", len(baseIDPart))
	}

	// Verify baseID is alphanumeric lowercase
	validPattern := regexp.MustCompile(`^[0-9a-z]{12}$`)
	if !validPattern.MatchString(baseIDPart) {
		t.Errorf("baseID must contain only lowercase alphanumeric characters, got: %s", baseIDPart)
	}
}

func TestGenerateID_ErrorHandling(t *testing.T) {
	// This test verifies that GenID handles errors properly.
	// Note: In practice, generateNanoID is unlikely to fail,
	// but we test that the function doesn't panic and handles errors gracefully.
	prefix := UserIDPrefix

	// Generate multiple IDs to ensure no errors occur
	for i := 0; i < 10; i++ {
		id, err := GenID(prefix, nil)
		if err != nil {
			if id != "" {
				t.Error("expected empty ID when error occurs")
			}
			// Verify error is not nil and has expected structure
			if err.Error() == "" {
				t.Error("expected error to have a message")
			}
		} else {
			if id == "" {
				t.Error("expected non-empty ID when no error occurs")
			}
		}
	}
}

func TestGenerateID_WithLength(t *testing.T) {
	tests := []struct {
		name           string
		prefix         IDPrefix
		length         *IDLength
		expectedLength int
	}{
		{
			name:           "default length (nil)",
			prefix:         UserIDPrefix,
			length:         nil,
			expectedLength: 12,
		},
		{
			name:           "length 12",
			prefix:         UserIDPrefix,
			length:         func() *IDLength { l := IDLength12; return &l }(),
			expectedLength: 12,
		},
		{
			name:           "length 19",
			prefix:         AccountIDPrefix,
			length:         func() *IDLength { l := IDLength19; return &l }(),
			expectedLength: 19,
		},
		{
			name:           "length 22",
			prefix:         SessionIDPrefix,
			length:         func() *IDLength { l := IDLength22; return &l }(),
			expectedLength: 22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GenID(tt.prefix, tt.length)
			if err != nil {
				t.Fatalf("GenID() returned error: %v", err)
			}

			if id == "" {
				t.Error("GenID() returned empty string")
			}

			// Validate format: prefix_baseID
			parts := strings.Split(id, "_")
			if len(parts) != 2 {
				t.Errorf("expected ID format 'prefix_baseID', got: %s", id)
			}

			baseID := parts[1]
			if len(baseID) != tt.expectedLength {
				t.Errorf("expected baseID length %d, got: %d", tt.expectedLength, len(baseID))
			}

			// Validate baseID contains only valid characters (0-9, a-z)
			validPattern := regexp.MustCompile(`^[0-9a-z]+$`)
			if !validPattern.MatchString(baseID) {
				t.Errorf("baseID contains invalid characters, got: %s", baseID)
			}
		})
	}
}
