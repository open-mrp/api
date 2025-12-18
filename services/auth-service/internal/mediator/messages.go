package mediator

const (
	// Access token errors
	ErrAccessTokenInvalid = "Invalid access token."
	ErrAccessTokenExpired = "Access token has expired."
	// Refresh token errors
	ErrInvalidRefreshToken = "Invalid refresh token."
	ErrExpiredRefreshToken = "Refresh token has expired."
	ErrRefreshTokenRevoked = "Refresh token has been revoked."
)
