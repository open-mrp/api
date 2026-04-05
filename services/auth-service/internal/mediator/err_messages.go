package mediator

import "fmt"

const (
	// Access token errors
	ErrAccessTokenInvalid = "Invalid access token."
	// Refresh token errors
	ErrInvalidRefreshToken = "Invalid refresh token."
	ErrExpiredRefreshToken = "Refresh token has expired."
	ErrRefreshTokenRevoked = "Refresh token has been revoked."
	// Actor account header: user must be an account member
	ErrActorAccountRequiresMember = "You do not have access to act on behalf of this account. The Augno-Actor-Account header requires you to be a member of the account."
)

func errNoAccountAccess(accountID string) string {
	return fmt.Sprintf("You do not have access to account '%s'.", accountID)
}
