package types

import "testing"

func TestParseIdentityRelationTypeValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		code     string
		expected IdentityRelationType
	}{
		{
			name:     "internal",
			code:     string(IdentityRelationTypeInternal),
			expected: IdentityRelationTypeInternal,
		},
		{
			name:     "customer",
			code:     string(IdentityRelationTypeCustomer),
			expected: IdentityRelationTypeCustomer,
		},
		{
			name:     "supplier",
			code:     string(IdentityRelationTypeSupplier),
			expected: IdentityRelationTypeSupplier,
		},
		{
			name:     "unassigned",
			code:     string(IdentityRelationTypeUnassigned),
			expected: IdentityRelationTypeUnassigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseIdentityRelationType(tt.code)
			if !ok {
				t.Fatalf("expected code %q to be valid", tt.code)
			}
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestParseIdentityRelationTypeInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		code string
	}{
		{
			name: "empty",
			code: "",
		},
		{
			name: "unknown",
			code: "partner",
		},
		{
			name: "uppercase",
			code: "INTERNAL",
		},
		{
			name: "whitespace",
			code: " customer ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseIdentityRelationType(tt.code)
			if ok {
				t.Fatalf("expected code %q to be invalid, got %q", tt.code, got)
			}
			if got != "" {
				t.Fatalf("expected zero-value IdentityRelationType, got %q", got)
			}
		})
	}
}
