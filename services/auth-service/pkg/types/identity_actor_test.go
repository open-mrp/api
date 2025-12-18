package types

import "testing"

func TestParseIdentityActorTypeValid(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected IdentityActorType
	}{
		{
			name:     "internal",
			code:     string(IdentityActorTypeInternal),
			expected: IdentityActorTypeInternal,
		},
		{
			name:     "customer",
			code:     string(IdentityActorTypeCustomer),
			expected: IdentityActorTypeCustomer,
		},
		{
			name:     "supplier",
			code:     string(IdentityActorTypeSupplier),
			expected: IdentityActorTypeSupplier,
		},
		{
			name:     "unassigned",
			code:     string(IdentityActorTypeUnassigned),
			expected: IdentityActorTypeUnassigned,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseIdentityActorType(tt.code)
			if !ok {
				t.Fatalf("expected code %q to be valid", tt.code)
			}
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestParseIdentityActorTypeInvalid(t *testing.T) {
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseIdentityActorType(tt.code)
			if ok {
				t.Fatalf("expected code %q to be invalid, got %q", tt.code, got)
			}
			if got != "" {
				t.Fatalf("expected zero-value IdentityActorType, got %q", got)
			}
		})
	}
}
