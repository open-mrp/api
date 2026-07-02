package validate

import (
	"strings"
	"testing"
	"time"

	"github.com/augno/api/shared/field"
)

type passwordTestStruct struct {
	Password string `validate:"password"`
}

type requiredPasswordTestStruct struct {
	Password string `validate:"required,password"`
}

func TestValidatePasswordTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		password string
		hasError bool
	}{
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
			password: strings.Repeat("A", 69) + "a1!",
			hasError: false,
		},
		{
			name:     "password too short",
			password: "Pass1!",
			hasError: true,
		},
		{
			name:     "password too long",
			password: strings.Repeat("A", 69) + "a123!",
			hasError: true,
		},
		{
			name:     "password without lowercase",
			password: "PASSWORD123!",
			hasError: true,
		},
		{
			name:     "password without uppercase",
			password: "password123!",
			hasError: true,
		},
		{
			name:     "password without numbers",
			password: "Password!@#",
			hasError: true,
		},
		{
			name:     "password without special characters",
			password: "Password123",
			hasError: true,
		},
		{
			name:     "empty password - passes because password tag allows empty (use required)",
			password: "",
			hasError: false,
		},
		{
			name:     "valid password with unicode",
			password: "Pássw0rd!",
			hasError: false,
		},
		{
			name:     "password exactly 72 characters with all requirements",
			password: strings.Repeat("A", 68) + "a1!@",
			hasError: false,
		},
		{
			name:     "password 7 characters (too short)",
			password: "Pass1!@",
			hasError: true,
		},
		{
			name:     "password 73 characters (too long)",
			password: strings.Repeat("A", 69) + "a1!@",
			hasError: true,
		},
		{
			name:     "password with tilde (not in special char regex)",
			password: "Password123~",
			hasError: true,
		},
		{
			name:     "password with backtick (not in special char regex)",
			password: "Password123`",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &passwordTestStruct{Password: tt.password}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for password: %s", tt.password)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for password: %s, but got error: %v", tt.password, err)
				}
			}
		})
	}
}

func TestValidatePasswordTagWithRequired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		password string
		hasError bool
	}{
		{
			name:     "valid password",
			password: "Password123!",
			hasError: false,
		},
		{
			name:     "empty password - fails because required",
			password: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &requiredPasswordTestStruct{Password: tt.password}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for password: %s", tt.password)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for password: %s, but got error: %v", tt.password, err)
				}
			}
		})
	}
}

type identifierTestStruct struct {
	Identifier string `validate:"identifier"`
}

type requiredIdentifierTestStruct struct {
	Identifier string `validate:"required,identifier"`
}

func TestValidateIdentifierTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		identifier string
		hasError   bool
	}{
		{
			name:       "valid email",
			identifier: "test@example.com",
			hasError:   false,
		},
		{
			name:       "valid username - minimum length",
			identifier: "abc",
			hasError:   false,
		},
		{
			name:       "valid username - with underscore",
			identifier: "john_doe",
			hasError:   false,
		},
		{
			name:       "valid username - alphanumeric",
			identifier: "user123",
			hasError:   false,
		},
		{
			name:       "invalid email - missing domain",
			identifier: "test@",
			hasError:   true,
		},
		{
			name:       "invalid email - missing local part",
			identifier: "@example.com",
			hasError:   true,
		},
		{
			name:       "invalid username - too short",
			identifier: "ab",
			hasError:   true,
		},
		{
			name:       "valid username - with hyphen",
			identifier: "john-doe",
			hasError:   false,
		},
		{
			name:       "invalid username - contains space",
			identifier: "john doe",
			hasError:   true,
		},
		{
			name:       "empty identifier - passes because tag allows empty (use required)",
			identifier: "",
			hasError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &identifierTestStruct{Identifier: tt.identifier}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for identifier: %s", tt.identifier)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for identifier: %s, but got error: %v", tt.identifier, err)
				}
			}
		})
	}
}

func TestValidateIdentifierTagWithRequired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		identifier string
		hasError   bool
	}{
		{
			name:       "valid email",
			identifier: "test@example.com",
			hasError:   false,
		},
		{
			name:       "empty identifier - fails because required",
			identifier: "",
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &requiredIdentifierTestStruct{Identifier: tt.identifier}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for identifier: %s", tt.identifier)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for identifier: %s, but got error: %v", tt.identifier, err)
				}
			}
		})
	}
}

type customEmailTestStruct struct {
	Email string `validate:"custom_email"`
}

type requiredCustomEmailTestStruct struct {
	Email string `validate:"required,custom_email"`
}

func TestValidateCustomEmailTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		email    string
		hasError bool
	}{
		{
			name:     "valid email",
			email:    "test@example.com",
			hasError: false,
		},
		{
			name:     "valid email with subdomain",
			email:    "test@mail.example.com",
			hasError: false,
		},
		{
			name:     "valid email with plus",
			email:    "test+tag@example.com",
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
			name:     "valid email with country code TLD",
			email:    "user@example.co.uk",
			hasError: false,
		},
		{
			name:     "invalid email - no at sign",
			email:    "testexample.com",
			hasError: true,
		},
		{
			name:     "invalid email - no domain",
			email:    "test@",
			hasError: true,
		},
		{
			name:     "invalid email - multiple @",
			email:    "user@@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - missing local part",
			email:    "@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - consecutive dots in local part",
			email:    "test..user@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - starts with dot",
			email:    ".test@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - ends with dot in local part",
			email:    "test.@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - no TLD",
			email:    "user@example",
			hasError: true,
		},
		{
			name:     "invalid email - single character TLD",
			email:    "user@example.c",
			hasError: true,
		},
		{
			name:     "invalid email - TLD containing numbers",
			email:    "user@example.c0m",
			hasError: true,
		},
		{
			name:     "invalid email - exceeding 254 characters",
			email:    "a" + strings.Repeat("a", 250) + "@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - local part exceeding 64 characters",
			email:    strings.Repeat("a", 65) + "@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - domain exceeding 253 characters",
			email:    "user@" + strings.Repeat("a", 250) + ".com",
			hasError: true,
		},
		{
			name:     "empty email - passes because tag allows empty (use required)",
			email:    "",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &customEmailTestStruct{Email: tt.email}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for email: %s", tt.email)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for email: %s, but got error: %v", tt.email, err)
				}
			}
		})
	}
}

func TestValidateCustomEmailTagWithRequired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		email    string
		hasError bool
	}{
		{
			name:     "valid email",
			email:    "test@example.com",
			hasError: false,
		},
		{
			name:     "empty email - fails because required",
			email:    "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &requiredCustomEmailTestStruct{Email: tt.email}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for email: %s", tt.email)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for email: %s, but got error: %v", tt.email, err)
				}
			}
		})
	}
}

type multipleOfTestStruct struct {
	Importance field.Optional[float64] `json:"importance,omitzero" validate:"omitempty,min=0,max=1,multiple_of=0.1"`
}

func TestValidateMultipleOfTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    field.Optional[float64]
		hasError bool
	}{
		{name: "unset passes", value: field.Optional[float64]{}, hasError: false},
		{name: "zero passes", value: field.Some(0.0), hasError: false},
		{name: "one tenth passes", value: field.Some(0.1), hasError: false},
		{name: "point three passes", value: field.Some(0.3), hasError: false},
		{name: "point eight passes", value: field.Some(0.8), hasError: false},
		{name: "one passes", value: field.Some(1.0), hasError: false},
		{name: "hundredth fails", value: field.Some(0.05), hasError: true},
		{name: "non-tenth fails", value: field.Some(0.15), hasError: true},
		{name: "above max fails", value: field.Some(1.1), hasError: true},
		{name: "negative fails", value: field.Some(-0.1), hasError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(&multipleOfTestStruct{Importance: tt.value})
			if tt.hasError && err == nil {
				t.Errorf("expected validation to fail, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("expected validation to pass, got: %v", err)
			}
		})
	}
}

type maxDaysAheadTestStruct struct {
	RevokeAt field.Optional[time.Time] `json:"revoke_at,omitzero" validate:"omitempty,max_days_ahead=30"`
}

func TestValidateMaxDaysAheadTag(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name     string
		value    field.Optional[time.Time]
		hasError bool
	}{
		{name: "unset passes", value: field.Optional[time.Time]{}, hasError: false},
		{name: "now passes", value: field.Some(now), hasError: false},
		{name: "within window passes", value: field.Some(now.Add(29 * 24 * time.Hour)), hasError: false},
		{name: "past passes (collapses to immediate)", value: field.Some(now.Add(-48 * time.Hour)), hasError: false},
		{name: "beyond 30 days fails", value: field.Some(now.Add(31 * 24 * time.Hour)), hasError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(&maxDaysAheadTestStruct{RevokeAt: tt.value})
			if tt.hasError && err == nil {
				t.Errorf("expected validation to fail, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("expected validation to pass, got: %v", err)
			}
		})
	}
}

func TestValidateMaxDaysAheadErrorMessage(t *testing.T) {
	t.Parallel()
	req := &maxDaysAheadTestStruct{RevokeAt: field.Some(time.Now().Add(60 * 24 * time.Hour))}
	err := Validate(req)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	expected := "must be no more than 30 days in the future"
	if !strings.Contains(err.PublicMessage, expected) {
		t.Errorf("expected message to contain '%s', got: %s", expected, err.PublicMessage)
	}
}

func TestValidatePasswordErrorMessage(t *testing.T) {
	t.Parallel()
	req := &requiredPasswordTestStruct{Password: "short"}
	err := Validate(req)

	if err == nil {
		t.Fatal("expected validation to fail")
	}

	expectedMessage := "must be 8-72 characters and contain at least one lowercase letter, one uppercase letter, one number, and one special character"
	if !strings.Contains(err.PublicMessage, expectedMessage) {
		t.Errorf("expected error message to contain '%s', but got: %s", expectedMessage, err.PublicMessage)
	}
}

func TestValidateIdentifierErrorMessage(t *testing.T) {
	t.Parallel()
	req := &requiredIdentifierTestStruct{Identifier: "ab"}
	err := Validate(req)

	if err == nil {
		t.Fatal("expected validation to fail")
	}

	expectedMessage := "must be a valid email address or username"
	if !strings.Contains(err.PublicMessage, expectedMessage) {
		t.Errorf("expected error message to contain '%s', but got: %s", expectedMessage, err.PublicMessage)
	}
}

type usernameTestStruct struct {
	Username string `validate:"username"`
}

type requiredUsernameTestStruct struct {
	Username string `validate:"required,username"`
}

func TestValidateUsernameTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		hasError bool
	}{
		{
			name:     "valid username - alphanumeric",
			username: "user123",
			hasError: false,
		},
		{
			name:     "valid username - minimum length",
			username: "abc",
			hasError: false,
		},
		{
			name:     "valid username - with underscore",
			username: "john_doe",
			hasError: false,
		},
		{
			name:     "valid username - with hyphen",
			username: "john-doe",
			hasError: false,
		},
		{
			name:     "valid username - uppercase",
			username: "JohnDoe",
			hasError: false,
		},
		{
			name:     "valid username - mixed case with symbols",
			username: "John_Doe-123",
			hasError: false,
		},
		{
			name:     "valid username - maximum length (255)",
			username: strings.Repeat("a", 255),
			hasError: false,
		},
		{
			name:     "invalid username - too short (2 chars)",
			username: "ab",
			hasError: true,
		},
		{
			name:     "invalid username - too long (256 chars)",
			username: strings.Repeat("a", 256),
			hasError: true,
		},
		{
			name:     "invalid username - contains space",
			username: "john doe",
			hasError: true,
		},
		{
			name:     "invalid username - contains at sign",
			username: "john@doe",
			hasError: true,
		},
		{
			name:     "invalid username - contains dot",
			username: "john.doe",
			hasError: true,
		},
		{
			name:     "invalid username - contains special characters",
			username: "john!doe",
			hasError: true,
		},
		{
			name:     "empty username - passes because tag allows empty (use required)",
			username: "",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &usernameTestStruct{Username: tt.username}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for username: %s", tt.username)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for username: %s, but got error: %v", tt.username, err)
				}
			}
		})
	}
}

func TestValidateUsernameTagWithRequired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		hasError bool
	}{
		{
			name:     "valid username",
			username: "john_doe",
			hasError: false,
		},
		{
			name:     "empty username - fails because required",
			username: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &requiredUsernameTestStruct{Username: tt.username}
			err := Validate(req)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected validation to fail for username: %s", tt.username)
				}
			} else {
				if err != nil {
					t.Errorf("expected validation to pass for username: %s, but got error: %v", tt.username, err)
				}
			}
		})
	}
}

func TestValidateUsernameErrorMessage(t *testing.T) {
	t.Parallel()
	req := &usernameTestStruct{Username: "a b"}
	err := Validate(req)

	if err == nil {
		t.Fatal("expected validation to fail")
	}

	expectedMessage := "must be 3-255 characters and contain only letters, numbers, underscores, and hyphens"
	if !strings.Contains(err.PublicMessage, expectedMessage) {
		t.Errorf("expected error message to contain '%s', but got: %s", expectedMessage, err.PublicMessage)
	}
}

func TestValidateCustomEmailErrorMessage(t *testing.T) {
	t.Parallel()
	req := &requiredCustomEmailTestStruct{Email: "invalid"}
	err := Validate(req)

	if err == nil {
		t.Fatal("expected validation to fail")
	}

	expectedMessage := "must be a valid email address"
	if !strings.Contains(err.PublicMessage, expectedMessage) {
		t.Errorf("expected error message to contain '%s', but got: %s", expectedMessage, err.PublicMessage)
	}
}
