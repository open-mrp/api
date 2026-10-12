package types

type IdentityActorType string

const (
	IdentityActorTypeUser            IdentityActorType = "user"
	IdentityActorTypeAPIKey          IdentityActorType = "api_key"
	IdentityActorTypeAgent           IdentityActorType = "agent"
	IdentityActorTypeUnauthenticated IdentityActorType = "unauthenticated"
	// IdentityActorTypeSystem attributes a background write (a webhook, the scheduler, inbound mail) to the process that made it. It authenticates nothing: see NewSystemIdentity.
	IdentityActorTypeSystem IdentityActorType = "system"
)
