package types

type IdentityActorType string

const (
	IdentityActorTypeUser            IdentityActorType = "user"
	IdentityActorTypeAPIKey          IdentityActorType = "api_key"
	IdentityActorTypeAgent           IdentityActorType = "agent"
	IdentityActorTypeUnauthenticated IdentityActorType = "unauthenticated"
)
