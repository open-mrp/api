package signingpin

import (
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	MinLength = 4
	MaxLength = 8
)

// Validate enforces the PIN rules, naming param in the error: 4–8 digits, not one repeated digit, not a run of consecutive digits up or down.
func Validate(pin, param string) *apierror.APIError {
	if len(pin) < MinLength || len(pin) > MaxLength {
		return apierror.NewValidationErrorWithParam("A signing PIN must be 4 to 8 digits.", param)
	}
	for i := 0; i < len(pin); i++ {
		if pin[i] < '0' || pin[i] > '9' {
			return apierror.NewValidationErrorWithParam("A signing PIN may contain only digits.", param)
		}
	}
	if isRepeated(pin) {
		return apierror.NewValidationErrorWithParam("A signing PIN can't be one digit repeated.", param)
	}
	if isRun(pin, 1) || isRun(pin, -1) {
		return apierror.NewValidationErrorWithParam("A signing PIN can't be a run of consecutive digits.", param)
	}
	return nil
}

func isRepeated(pin string) bool {
	return isRun(pin, 0)
}

func isRun(pin string, step int) bool {
	for i := 1; i < len(pin); i++ {
		if int(pin[i])-int(pin[i-1]) != step {
			return false
		}
	}
	return true
}
