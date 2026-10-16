package constants

import "strings"

// SystemActor names the background process a system identity attributes a write to.
type SystemActor string

const (
	// SystemActorStripe is a Stripe webhook: a payment, payout or subscription change Stripe reported.
	SystemActorStripe SystemActor = "stripe"
	// SystemActorScheduler is a scheduled job, such as the production schedule generation cadence.
	SystemActorScheduler SystemActor = "scheduler"
	// SystemActorEmailIngest is inbound email threaded into a conversation.
	SystemActorEmailIngest SystemActor = "email_ingest"
)

// systemActorIDPrefix leads every system actor ID, which keeps them apart from every type-prefixed entity ID.
const systemActorIDPrefix = "system:"

func (a SystemActor) IsValid() bool {
	switch a {
	case SystemActorStripe, SystemActorScheduler, SystemActorEmailIngest:
		return true
	default:
		return false
	}
}

func (a SystemActor) EnumValues() []string {
	return []string{string(SystemActorStripe), string(SystemActorScheduler), string(SystemActorEmailIngest)}
}

func (a *SystemActor) StringPtr() *string {
	if a == nil {
		return nil
	}
	s := string(*a)
	return &s
}

// DisplayName is the process's name as an actor shows it, such as `Stripe`.
func (a SystemActor) DisplayName() string {
	switch a {
	case SystemActorStripe:
		return "Stripe"
	case SystemActorScheduler:
		return "Scheduler"
	case SystemActorEmailIngest:
		return "Inbound email"
	default:
		return "OpenMRP"
	}
}

// ActorID is the actor ID audit events record for the process, such as `system:stripe`.
func (a SystemActor) ActorID() string {
	return systemActorIDPrefix + string(a)
}

// ParseSystemActorID reads the process back out of a system actor ID, reporting false for anything that is not one.
func ParseSystemActorID(actorID string) (SystemActor, bool) {
	rest, ok := strings.CutPrefix(actorID, systemActorIDPrefix)
	if !ok {
		return "", false
	}
	a := SystemActor(rest)
	return a, a.IsValid()
}
