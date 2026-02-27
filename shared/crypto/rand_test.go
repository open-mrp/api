package crypto

import (
	"strings"
	"testing"
)

func TestRandAlphanumericString(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		wantErr  bool
		validate func(string) bool
	}{
		{
			name:    "length 10",
			length:  10,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 10 && containsOnlyAlphanum(key)
			},
		},
		{
			name:    "length 32",
			length:  32,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 32 && containsOnlyAlphanum(key)
			},
		},
		{
			name:    "length 1",
			length:  1,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 1 && containsOnlyAlphanum(key)
			},
		},
		{
			name:    "length 0",
			length:  0,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := RandAlphanumericString(tt.length)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RandAlphanumericString() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("RandAlphanumericString() unexpected error = %v", err)
				return
			}
			if !tt.validate(key) {
				t.Errorf("RandAlphanumericString() generated key '%s' failed validation", key)
			}
		})
	}
}

func TestRandAlphanumericString_Uniqueness(t *testing.T) {
	key1, err := RandAlphanumericString(16)
	if err != nil {
		t.Fatalf("RandAlphanumericString() unexpected error = %v", err)
	}

	key2, err := RandAlphanumericString(16)
	if err != nil {
		t.Fatalf("RandAlphanumericString() unexpected error = %v", err)
	}

	if key1 == key2 {
		t.Errorf("RandAlphanumericString() generated identical keys: %s", key1)
	}
}

func containsOnlyAlphanum(s string) bool {
	for _, char := range s {
		if !strings.ContainsRune(charset, char) {
			return false
		}
	}
	return true
}
