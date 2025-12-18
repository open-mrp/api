package password

import (
	"context"
	"testing"
)

func TestCompareHashAndPassword(t *testing.T) {
	// First, we need to create some hashed passwords for testing
	password1 := "correctPassword123!"
	password2 := "anotherPassword456!"
	hash1, _ := HashPassword(context.Background(), password1)
	hash2, _ := HashPassword(context.Background(), password2)

	tests := []struct {
		name     string
		password string
		hash     string
		result   bool
		wantErr  bool
	}{
		{
			name:     "Correct password",
			password: password1,
			hash:     hash1,
			result:   true,
			wantErr:  false,
		},
		{
			name:     "Incorrect password",
			password: "wrongPassword",
			hash:     hash1,
			result:   false,
			wantErr:  false,
		},
		{
			name:     "Password doesn't match different hash",
			password: password1,
			hash:     hash2,
			result:   false,
			wantErr:  false,
		},
		{
			name:     "Empty password",
			password: "",
			hash:     hash1,
			result:   false,
			wantErr:  false,
		},
		{
			name:     "Invalid hash",
			password: password1,
			hash:     "invalidhash",
			result:   false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passwordMatches, err := CompareHashAndPassword(context.Background(), tt.password, tt.hash)

			// Check if we got an error when we expected one
			if tt.wantErr && err == nil {
				t.Errorf("CompareHashAndPassword() expected error but got none")
				return
			}

			// Check if we got an error when we didn't expect one
			if !tt.wantErr && err != nil {
				t.Errorf("CompareHashAndPassword() unexpected error = %v", err)
				return
			}

			if passwordMatches != tt.result {
				t.Errorf("CompareHashAndPassword() passwordMatches = %v, want %v", passwordMatches, tt.result)
			}
		})
	}
}
