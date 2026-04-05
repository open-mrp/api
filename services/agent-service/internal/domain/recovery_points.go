package domain

// RecoveryPoint tracks progress through idempotent operation execution.
type RecoveryPoint string

const (
	RecoveryPointStarted  RecoveryPoint = "agent:started"
	RecoveryPointFinished RecoveryPoint = "agent:finished"
)

func (r RecoveryPoint) IsValid() bool {
	switch r {
	case RecoveryPointStarted, RecoveryPointFinished:
		return true
	default:
		return false
	}
}
