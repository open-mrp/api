package validate

import "regexp"

var (
	hasNumber      = regexp.MustCompile(`[0-9]`)
	hasSpecialChar = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
)

func ValidatePasswordPlaintext(v *Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 characters long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 characters long")
	v.Check(hasNumber.MatchString(password), "password", "must contain at least one number")
	v.Check(hasSpecialChar.MatchString(password), "password", "must contain at least one special character")
}
