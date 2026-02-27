package mediator

import "fmt"

const (
	// Access token errors
	ErrAccessTokenInvalid = "Invalid access token."
	// Refresh token errors
	ErrInvalidRefreshToken = "Invalid refresh token."
	ErrExpiredRefreshToken = "Refresh token has expired."
	ErrRefreshTokenRevoked = "Refresh token has been revoked."
)

func errNoAccountAccess(accountID string) string {
	return fmt.Sprintf("You do not have access to account '%s'.", accountID)
}
