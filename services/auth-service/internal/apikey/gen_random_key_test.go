package apikey

import (
	"strings"
	"testing"
)

func TestGenRandomKey(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		wantErr  bool
		validate func(string) bool
	}{
		{
			name:    "generate key of length 10",
			length:  10,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 10 && containsOnlyCharset(key)
			},
		},
		{
			name:    "generate key of length 32",
			length:  32,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 32 && containsOnlyCharset(key)
			},
		},
		{
			name:    "generate key of length 1",
			length:  1,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 1 && containsOnlyCharset(key)
			},
		},
		{
			name:    "generate key of length 0",
			length:  0,
			wantErr: false,
			validate: func(key string) bool {
				return len(key) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := genRandString(tt.length)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenRandomKey() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GenRandomKey() unexpected error = %v", err)
				return
			}

			if !tt.validate(key) {
				t.Errorf("GenRandomKey() generated key '%s' failed validation", key)
			}
		})
	}
}

func TestGenRandomKey_Uniqueness(t *testing.T) {
	// Test that multiple calls generate different keys
	key1, err := genRandString(16)
	if err != nil {
		t.Fatalf("GenRandomKey() unexpected error = %v", err)
	}

	key2, err := genRandString(16)
	if err != nil {
		t.Fatalf("GenRandomKey() unexpected error = %v", err)
	}

	if key1 == key2 {
		t.Errorf("GenRandomKey() generated identical keys: %s", key1)
	}
}

// Helper function to check if a string contains only characters from the charset
func containsOnlyCharset(s string) bool {
	for _, char := range s {
		if !strings.ContainsRune(keyCharset, char) {
			return false
		}
	}
	return true
}
