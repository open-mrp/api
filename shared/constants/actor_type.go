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
	// ActorTypeGroup indicates that the actor is a shared group identity (e.g. a "Customer Service" persona).
	ActorTypeGroup ActorType = "group"
	// ActorTypeDevice indicates that the actor is an enrolled device, such as a scanning station or kiosk.
	ActorTypeDevice ActorType = "device"
	// ActorTypeSystem indicates that OpenMRP itself acted, with no user, key, agent or device behind it. It has no ID.
	ActorTypeSystem ActorType = "system"
)

func (m ActorType) IsValid() bool {
	switch m {
	case ActorTypeUser, ActorTypeAPIKey, ActorTypeAgent, ActorTypeGroup, ActorTypeDevice, ActorTypeSystem:
		return true
	default:
		return false
	}
}

func (m ActorType) EnumValues() []string {
	return []string{string(ActorTypeUser), string(ActorTypeAPIKey), string(ActorTypeAgent), string(ActorTypeGroup),
		string(ActorTypeDevice), string(ActorTypeSystem)}
}

func (m *ActorType) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
