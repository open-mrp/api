package types

type IdentityType string

const (
	IdentityTypeUser            IdentityType = "user"
	IdentityTypeAPIKey          IdentityType = "api_key"
	IdentityTypeUnauthenticated IdentityType = "unauthenticated"
)
