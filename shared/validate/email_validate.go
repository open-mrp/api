package validate

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	EmailRX = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)
)

func ValidateEmail(v *Validator, email string) {
	v.Check(email != "", "email", "must be provided")

	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		localPart := parts[0]
		domain := parts[1]

		v.Check(utf8.RuneCountInString(localPart) <= 64, "email", "local part must not exceed 64 characters")
		v.Check(utf8.RuneCountInString(localPart) > 0, "email", "local part cannot be empty")
		v.Check(utf8.RuneCountInString(domain) <= 253, "email", "domain must not exceed 253 characters")
		v.Check(!strings.Contains(localPart, ".."), "email", "local part cannot contain consecutive dots")
		v.Check(!strings.HasPrefix(localPart, "."), "email", "local part cannot start with a dot")
		v.Check(!strings.HasSuffix(localPart, "."), "email", "local part cannot end with a dot")
		domainParts := strings.Split(domain, ".")
		if len(domainParts) >= 2 {
			tld := domainParts[len(domainParts)-1]
			v.Check(len(tld) >= 2, "email", "top-level domain must be at least 2 characters")
			v.Check(regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(tld), "email", "top-level domain must contain only letters")
		}
	}

	if email != "" {
		v.Check(utf8.RuneCountInString(email) <= 254, "email", "must not exceed 254 characters")
	}

	if !Matches(email, EmailRX) {
		v.AddError("email", "must be a valid email address")
	}
}

func ValidateUsernameOrEmail(v *Validator, usernameOrEmail string) {
	v.Check(usernameOrEmail != "", "username", "must be provided")

	if usernameOrEmail == "" {
		return
	}

	if strings.Contains(usernameOrEmail, "@") {
		ValidateEmail(v, usernameOrEmail)
		if emailErr, exists := v.Errors["email"]; exists {
			delete(v.Errors, "email")
			v.Errors["username"] = emailErr
		}
	} else {
		usernameLen := utf8.RuneCountInString(usernameOrEmail)
		v.Check(usernameLen >= 3, "username", "must be at least 3 characters long")
		v.Check(usernameLen <= 50, "username", "must not exceed 50 characters")
		usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
		v.Check(usernameRegex.MatchString(usernameOrEmail), "username", "must contain only letters, numbers, and underscores")
	}
}
