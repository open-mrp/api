package validate

import (
	"strings"
	"testing"
)

func TestValidatePasswordPlaintext(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hasError bool
		errorKey string
	}{
		// Valid passwords
		{
			name:     "valid password with all requirements",
			password: "Password123!",
			hasError: false,
		},
		{
			name:     "valid password with minimum length",
			password: "Pass1!@#",
			hasError: false,
		},
		{
			name:     "valid password with maximum length",
			password: strings.Repeat("A", 70) + "1!",
			hasError: false,
		},
		{
			name:     "valid password with multiple numbers",
			password: "Password123456!",
			hasError: false,
		},
		{
			name:     "valid password with multiple special characters",
			password: "Password123!@#$%",
			hasError: false,
		},
		{
			name:     "valid password with mixed case",
			password: "MyP@ssw0rd",
			hasError: false,
		},
		{
			name:     "valid password with underscore",
			password: "Password_123",
			hasError: false,
		},
		{
			name:     "valid password with hyphen",
			password: "Password-123",
			hasError: false,
		},
		{
			name:     "valid password with brackets",
			password: "Password[123]",
			hasError: false,
		},
		{
			name:     "valid password with quotes",
			password: "Password\"123\"",
			hasError: false,
		},

		// Invalid passwords - empty
		{
			name:     "empty password",
			password: "",
			hasError: true,
			errorKey: "password",
		},

		// Invalid passwords - length issues
		{
			name:     "password too short",
			password: "Pass1!",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password too long",
			password: strings.Repeat("A", 70) + "123!",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password exactly 8 characters but missing number",
			password: "Password!",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password exactly 8 characters but missing special character",
			password: "Password1",
			hasError: true,
			errorKey: "password",
		},

		// Invalid passwords - missing numbers
		{
			name:     "password without numbers",
			password: "Password!",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password with only letters and special chars",
			password: "MyPassword!@#",
			hasError: true,
			errorKey: "password",
		},

		// Invalid passwords - missing special characters
		{
			name:     "password without special characters",
			password: "Password123",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password with only letters and numbers",
			password: "MyPassword123456",
			hasError: true,
			errorKey: "password",
		},

		// Invalid passwords - missing both number and special character
		{
			name:     "password with only letters",
			password: "Password",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password with only lowercase letters",
			password: "password",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password with only uppercase letters",
			password: "PASSWORD",
			hasError: true,
			errorKey: "password",
		},

		// Edge cases
		{
			name:     "password with only spaces",
			password: "        ",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password with only numbers",
			password: "12345678",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password with only special characters",
			password: "!@#$%^&*",
			hasError: true,
			errorKey: "password",
		},
		{
			name:     "password with unicode characters",
			password: "Pássw0rd!",
			hasError: false, // Should be valid if it meets all requirements
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := New()
			ValidatePasswordPlaintext(validator, tt.password)

			if tt.hasError {
				if validator.Valid() {
					t.Errorf("expected validation to fail for password: %s", tt.password)
				}
				if _, exists := validator.Errors[tt.errorKey]; !exists {
					t.Errorf("expected error key '%s' to exist in validation errors", tt.errorKey)
				}
			} else {
				if !validator.Valid() {
					t.Errorf("expected validation to pass for password: %s, but got errors: %v", tt.password, validator.Errors)
				}
			}
		})
	}
}

func TestValidatePasswordPlaintext_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		password       string
		expectedErrors map[string]string
	}{
		{
			name:     "empty password error message",
			password: "",
			expectedErrors: map[string]string{
				"password": "must be provided",
			},
		},
		{
			name:     "password too short error message",
			password: "Pass1!",
			expectedErrors: map[string]string{
				"password": "must be at least 8 characters long",
			},
		},
		{
			name:     "password too long error message",
			password: strings.Repeat("A", 70) + "123!",
			expectedErrors: map[string]string{
				"password": "must not be more than 72 characters long",
			},
		},
		{
			name:     "password missing number error message",
			password: "Password!",
			expectedErrors: map[string]string{
				"password": "must contain at least one number",
			},
		},
		{
			name:     "password missing special character error message",
			password: "Password123",
			expectedErrors: map[string]string{
				"password": "must contain at least one special character",
			},
		},
		{
			name:     "password missing both number and special character",
			password: "Password",
			expectedErrors: map[string]string{
				"password": "must contain at least one number",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := New()
			ValidatePasswordPlaintext(validator, tt.password)

			if validator.Valid() {
				t.Errorf("expected validation to fail for password: %s", tt.password)
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

func TestValidatePasswordPlaintext_RegexPatterns(t *testing.T) {
	// Test the regex patterns directly
	validNumbers := []string{
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"123", "abc1def", "1abc", "abc1",
	}

	invalidNumbers := []string{
		"", "abc", "!@#", "ABC", "   ",
	}

	validSpecialChars := []string{
		"!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "_", "+", "-", "=",
		"[", "]", "{", "}", ";", ":", "'", "\"", "\\", "|", ",", ".", "<", ">", "/", "?",
		"abc!", "!abc", "a!b", "!@#", "abc!@#",
	}

	invalidSpecialChars := []string{
		"", "abc", "123", "ABC", "   ", "~", "`",
	}

	// Test number regex
	for _, test := range validNumbers {
		if !hasNumber.MatchString(test) {
			t.Errorf("expected '%s' to match number regex", test)
		}
	}

	for _, test := range invalidNumbers {
		if hasNumber.MatchString(test) {
			t.Errorf("expected '%s' to not match number regex", test)
		}
	}

	// Test special character regex
	for _, test := range validSpecialChars {
		if !hasSpecialChar.MatchString(test) {
			t.Errorf("expected '%s' to match special character regex", test)
		}
	}

	for _, test := range invalidSpecialChars {
		if hasSpecialChar.MatchString(test) {
			t.Errorf("expected '%s' to not match special character regex", test)
		}
	}
}

func TestValidatePasswordPlaintext_ValidatorReuse(t *testing.T) {
	// Test that the same validator instance can be reused for multiple validations
	validator := New()

	// First validation - should pass
	ValidatePasswordPlaintext(validator, "ValidPass123!")
	if !validator.Valid() {
		t.Errorf("expected first validation to pass")
	}

	// Second validation - should fail
	ValidatePasswordPlaintext(validator, "invalid")
	if validator.Valid() {
		t.Errorf("expected second validation to fail")
	}

	// Check that errors are present
	if len(validator.Errors) == 0 {
		t.Errorf("expected errors to be present")
	}

	if _, exists := validator.Errors["password"]; !exists {
		t.Errorf("expected 'password' error to exist")
	}
}

func TestValidatePasswordPlaintext_ValidatorState(t *testing.T) {
	// Test that validator state is properly maintained
	validator := New()

	// Initially should be valid
	if !validator.Valid() {
		t.Errorf("expected validator to be valid initially")
	}

	// Add an error
	ValidatePasswordPlaintext(validator, "")
	if validator.Valid() {
		t.Errorf("expected validator to be invalid after adding error")
	}

	// Check error count
	if len(validator.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(validator.Errors))
	}
}

func TestValidatePasswordPlaintext_LengthBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hasError bool
	}{
		{
			name:     "password exactly 8 characters with all requirements",
			password: "Pass1!@#",
			hasError: false,
		},
		{
			name:     "password exactly 72 characters with all requirements",
			password: strings.Repeat("A", 69) + "1!@",
			hasError: false,
		},
		{
			name:     "password 7 characters (too short)",
			password: "Pass1!@",
			hasError: true,
		},
		{
			name:     "password 73 characters (too long)",
			password: strings.Repeat("A", 70) + "1!@#",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := New()
			ValidatePasswordPlaintext(validator, tt.password)

			if tt.hasError {
				if validator.Valid() {
					t.Errorf("expected validation to fail for password length: %d", len(tt.password))
				}
			} else {
				if !validator.Valid() {
					t.Errorf("expected validation to pass for password length: %d, but got errors: %v", len(tt.password), validator.Errors)
				}
			}
		})
	}
}

func TestValidatePasswordPlaintext_SpecialCharacterEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hasError bool
	}{
		{
			name:     "password with backslash",
			password: "Pass123\\",
			hasError: false,
		},
		{
			name:     "password with forward slash",
			password: "Pass123/",
			hasError: false,
		},
		{
			name:     "password with pipe",
			password: "Pass123|",
			hasError: false,
		},
		{
			name:     "password with comma",
			password: "Pass123,",
			hasError: false,
		},
		{
			name:     "password with period",
			password: "Pass123.",
			hasError: false,
		},
		{
			name:     "password with tilde (not in special char regex)",
			password: "Pass123~",
			hasError: true,
		},
		{
			name:     "password with backtick (not in special char regex)",
			password: "Pass123`",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := New()
			ValidatePasswordPlaintext(validator, tt.password)

			if tt.hasError {
				if validator.Valid() {
					t.Errorf("expected validation to fail for password: %s", tt.password)
				}
			} else {
				if !validator.Valid() {
					t.Errorf("expected validation to pass for password: %s, but got errors: %v", tt.password, validator.Errors)
				}
			}
		})
	}
}
