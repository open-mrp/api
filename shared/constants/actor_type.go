package constants

// ActorType represents the type of actor.
type ActorType string

const (
	// ActorTypeUser indicates that the actor is a user.
	ActorTypeUser ActorType = "user"
	// ActorTypeAPIKey indicates that the actor is an API key.
	ActorTypeAPIKey ActorType = "api_key"
	// ActorTypeAgent indicates that the actor is an agent.
	ActorTypeAgent ActorType = "agent"
)

func (m ActorType) IsValid() bool {
	switch m {
	case ActorTypeUser, ActorTypeAPIKey, ActorTypeAgent:
		return true
	default:
		return false
	}
}

func (m ActorType) EnumValues() []string {
	return []string{string(ActorTypeUser), string(ActorTypeAPIKey), string(ActorTypeAgent)}
}

func (m *ActorType) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
