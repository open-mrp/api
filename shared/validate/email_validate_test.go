package validate

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		hasError bool
		errorKey string
	}{
		// Valid emails
		{
			name:     "valid simple email",
			email:    "test@example.com",
			hasError: false,
		},
		{
			name:     "valid email with subdomain",
			email:    "user@mail.example.com",
			hasError: false,
		},
		{
			name:     "valid email with numbers",
			email:    "user123@example123.com",
			hasError: false,
		},
		{
			name:     "valid email with special characters",
			email:    "user.name+tag@example-domain.co.uk",
			hasError: false,
		},
		{
			name:     "valid email with multiple dots",
			email:    "firstname.lastname@company.example.com",
			hasError: false,
		},
		{
			name:     "valid email with underscore",
			email:    "user_name@example.com",
			hasError: false,
		},
		{
			name:     "valid email with hyphen in domain",
			email:    "user@example-domain.com",
			hasError: false,
		},
		{
			name:     "valid email with all special characters",
			email:    "user!#$%&'*+/=?^_`{|}~-@example.com",
			hasError: false,
		},
		{
			name:     "valid email with long domain",
			email:    "test@very-long-domain-name-that-is-still-valid.com",
			hasError: false,
		},
		{
			name:     "valid email with country code TLD",
			email:    "user@example.co.uk",
			hasError: false,
		},

		// Invalid emails - empty
		{
			name:     "empty email",
			email:    "",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - missing @
		{
			name:     "email without @",
			email:    "userexample.com",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - multiple @
		{
			name:     "email with multiple @",
			email:    "user@@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with multiple @ in different positions",
			email:    "user@example@com",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - missing domain
		{
			name:     "email without domain",
			email:    "user@",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - missing local part
		{
			name:     "email without local part",
			email:    "@example.com",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - invalid characters
		{
			name:     "email with spaces",
			email:    "user name@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with invalid special characters",
			email:    "user()@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with brackets",
			email:    "user[]@example.com",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - domain issues
		{
			name:     "email with domain starting with hyphen",
			email:    "user@-example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with domain ending with hyphen",
			email:    "user@example-.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with consecutive dots in domain",
			email:    "user@example..com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with domain starting with dot",
			email:    "user@.example.com",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - TLD issues
		{
			name:     "email without TLD",
			email:    "user@example",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with TLD starting with hyphen",
			email:    "user@example.-com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with single character TLD",
			email:    "user@example.c",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with TLD containing numbers",
			email:    "user@example.c0m",
			hasError: true,
			errorKey: "email",
		},

		// Edge cases
		{
			name:     "email with only @",
			email:    "@",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with just @ and domain",
			email:    "@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with just local part and @",
			email:    "user@",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - length issues
		{
			name:     "email exceeding 254 characters",
			email:    "a" + strings.Repeat("a", 250) + "@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "local part exceeding 64 characters",
			email:    strings.Repeat("a", 65) + "@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "domain exceeding 253 characters",
			email:    "user@" + strings.Repeat("a", 250) + ".com",
			hasError: true,
			errorKey: "email",
		},

		// Invalid emails - local part issues
		{
			name:     "email with consecutive dots in local part",
			email:    "user..name@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with local part starting with dot",
			email:    ".username@example.com",
			hasError: true,
			errorKey: "email",
		},
		{
			name:     "email with local part ending with dot",
			email:    "username.@example.com",
			hasError: true,
			errorKey: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := New()
			ValidateEmail(validator, tt.email)

			if tt.hasError {
				if validator.Valid() {
					t.Errorf("expected validation to fail for email: %s", tt.email)
				}
				if _, exists := validator.Errors[tt.errorKey]; !exists {
					t.Errorf("expected error key '%s' to exist in validation errors", tt.errorKey)
				}
			} else {
				if !validator.Valid() {
					t.Errorf("expected validation to pass for email: %s, but got errors: %v", tt.email, validator.Errors)
				}
			}
		})
	}
}

func TestValidateEmail_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		expectedErrors map[string]string
	}{
		{
			name:  "empty email error message",
			email: "",
			expectedErrors: map[string]string{
				"email": "must be provided",
			},
		},
		{
			name:  "invalid email format error message",
			email: "invalid-email",
			expectedErrors: map[string]string{
				"email": "must be a valid email address",
			},
		},
		{
			name:  "empty email should only show 'must be provided' error",
			email: "",
			expectedErrors: map[string]string{
				"email": "must be provided",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := New()
			ValidateEmail(validator, tt.email)

			if validator.Valid() {
				t.Errorf("expected validation to fail for email: %s", tt.email)
			}

			for expectedKey, expectedMessage := range tt.expectedErrors {
				if actualMessage, exists := validator.Errors[expectedKey]; !exists {
					t.Errorf("expected error key '%s' to exist in validation errors", expectedKey)
				} else if actualMessage != expectedMessage {
					t.Errorf("expected error message '%s' for key '%s', but got '%s'", expectedMessage, expectedKey, actualMessage)
				}
			}
		})
	}
}

func TestEmailRX_RegexPattern(t *testing.T) {
	// Test the regex pattern directly to ensure it works as expected
	validEmails := []string{
		"test@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user123@example123.com",
		"user_name@example.com",
		"user@example-domain.com",
		"user@sub.example.com",
		"user@very-long-domain-name.com",
		"user!#$%&'*+/=?^_`{|}~-@example.com",
		"user@example.co.uk",
	}

	invalidEmails := []string{
		"",
		"userexample.com",
		"user@@example.com",
		"user@",
		"@example.com",
		"user name@example.com",
		"user()@example.com",
		"user@-example.com",
		"user@example-.com",
		"user@example..com",
		"user@.example.com",
		"user@example",     // Now correctly invalid - no TLD
		"user@example.c",   // Single character TLD
		"user@example.c0m", // TLD with numbers
		"@",
	}

	for _, email := range validEmails {
		if !EmailRX.MatchString(email) {
			t.Errorf("expected email '%s' to match regex pattern", email)
		}
	}

	for _, email := range invalidEmails {
		if EmailRX.MatchString(email) {
			t.Errorf("expected email '%s' to not match regex pattern", email)
		}
	}
}

func TestValidateEmail_ValidatorReuse(t *testing.T) {
	// Test that the same validator instance can be reused for multiple validations
	validator := New()

	// First validation - should pass
	ValidateEmail(validator, "valid@example.com")
	if !validator.Valid() {
		t.Errorf("expected first validation to pass")
	}

	// Second validation - should fail
	ValidateEmail(validator, "invalid-email")
	if validator.Valid() {
		t.Errorf("expected second validation to fail")
	}

	// Check that both errors are present
	if len(validator.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(validator.Errors))
	}

	if _, exists := validator.Errors["email"]; !exists {
		t.Errorf("expected 'email' error to exist")
	}
}

func TestValidateEmail_ValidatorState(t *testing.T) {
	// Test that validator state is properly maintained
	validator := New()

	// Initially should be valid
	if !validator.Valid() {
		t.Errorf("expected validator to be valid initially")
	}

	// Add an error
	ValidateEmail(validator, "")
	if validator.Valid() {
		t.Errorf("expected validator to be invalid after adding error")
	}

	// Check error count
	if len(validator.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(validator.Errors))
	}
}

func TestValidateEmail_EnhancedValidation(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		expectedErrors map[string]string
	}{
		{
			name:  "email with consecutive dots in local part",
			email: "user..name@example.com",
			expectedErrors: map[string]string{
				"email": "local part cannot contain consecutive dots",
			},
		},
		{
			name:  "email with local part starting with dot",
			email: ".username@example.com",
			expectedErrors: map[string]string{
				"email": "local part cannot start with a dot",
			},
		},
		{
			name:  "email with local part ending with dot",
			email: "username.@example.com",
			expectedErrors: map[string]string{
				"email": "local part cannot end with a dot",
			},
		},
		{
			name:  "email with single character TLD",
			email: "user@example.c",
			expectedErrors: map[string]string{
				"email": "top-level domain must be at least 2 characters",
			},
		},
		{
			name:  "email with TLD containing numbers",
			email: "user@example.c0m",
			expectedErrors: map[string]string{
				"email": "top-level domain must contain only letters",
			},
		},
		{
			name:  "email exceeding 254 characters",
			email: strings.Repeat("a", 60) + "@" + strings.Repeat("a", 200) + ".com",
			expectedErrors: map[string]string{
				"email": "must not exceed 254 characters",
			},
		},
		{
			name:  "local part exceeding 64 characters",
			email: strings.Repeat("a", 65) + "@example.com",
			expectedErrors: map[string]string{
				"email": "local part must not exceed 64 characters",
			},
		},
		{
			name:  "domain exceeding 253 characters",
			email: "u@" + strings.Repeat("a", 251) + ".com",
			expectedErrors: map[string]string{
				"email": "domain must not exceed 253 characters",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := New()
			ValidateEmail(validator, tt.email)

			if validator.Valid() {
				t.Errorf("expected validation to fail for email: %s", tt.email)
			}

			for expectedKey, expectedMessage := range tt.expectedErrors {
				if actualMessage, exists := validator.Errors[expectedKey]; !exists {
					t.Errorf("expected error key '%s' to exist in validation errors", expectedKey)
				} else if actualMessage != expectedMessage {
					t.Errorf("expected error message '%s' for key '%s', but got '%s'", expectedMessage, expectedKey, actualMessage)
				}
			}
		})
	}
}
