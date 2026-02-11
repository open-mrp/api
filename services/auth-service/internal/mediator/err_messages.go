package mediator

const (
	// Access token errors
	ErrAccessTokenInvalid = "Invalid access token."
	ErrAccessTokenExpired = "Access token has expired."
	// Refresh token errors
	ErrInvalidRefreshToken = "Invalid refresh token."
	ErrExpiredRefreshToken = "Refresh token has expired."
	ErrRefreshTokenRevoked = "Refresh token has been revoked."
	// Account access errors
	ErrNoAccountAccess = "You do not have access to the specified account."
)
