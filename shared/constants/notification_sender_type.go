package constants

// NotificationSenderType identifies what kind of actor generated a notification (its polymorphic sender): user, group, system, agent, or api_key.
type NotificationSenderType string

const (
	// NotificationSenderTypeUser is an account user.
	NotificationSenderTypeUser NotificationSenderType = "user"
	// NotificationSenderTypeGroup is a shared group identity (e.g. "Customer Service").
	NotificationSenderTypeGroup NotificationSenderType = "group"
	// NotificationSenderTypeSystem is the platform itself.
	NotificationSenderTypeSystem NotificationSenderType = "system"
	// NotificationSenderTypeAgent is an AI agent.
	NotificationSenderTypeAgent NotificationSenderType = "agent"
	// NotificationSenderTypeAPIKey is an API key actor.
	NotificationSenderTypeAPIKey NotificationSenderType = "apikey"
)

func (t NotificationSenderType) IsValid() bool {
	switch t {
	case NotificationSenderTypeUser, NotificationSenderTypeGroup, NotificationSenderTypeSystem, NotificationSenderTypeAgent, NotificationSenderTypeAPIKey:
		return true
	default:
		return false
	}
}

func (t NotificationSenderType) EnumValues() []string {
	return []string{
		string(NotificationSenderTypeUser),
		string(NotificationSenderTypeGroup),
		string(NotificationSenderTypeSystem),
		string(NotificationSenderTypeAgent),
		string(NotificationSenderTypeAPIKey),
	}
}

func (t *NotificationSenderType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}

// ActorTypeFromSenderType maps a notification/chat sender type to its unified ActorType. System (and any unattributed/unknown) senders return the empty ActorType, signalling the caller to emit a null actor rather than fabricating one.
func ActorTypeFromSenderType(t NotificationSenderType) ActorType {
	switch t {
	case NotificationSenderTypeUser:
		return ActorTypeUser
	case NotificationSenderTypeAgent:
		return ActorTypeAgent
	case NotificationSenderTypeAPIKey:
		return ActorTypeAPIKey
	case NotificationSenderTypeGroup:
		return ActorTypeGroup
	default:
		return ""
	}
}
