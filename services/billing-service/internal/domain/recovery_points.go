package domain

type RecoveryPoint string

const (
	RecoveryPointStarted         RecoveryPoint = "billing:started"
	RecoveryPointProfileCreated  RecoveryPoint = "billing:profile_created"
	RecoveryPointIntentCommitted RecoveryPoint = "billing:intent_committed"
	RecoveryPointFinished        RecoveryPoint = "billing:finished"
)

func (r RecoveryPoint) String() string {
	return string(r)
}

func (r RecoveryPoint) IsValid() bool {
	switch r {
	case RecoveryPointStarted, RecoveryPointProfileCreated, RecoveryPointIntentCommitted, RecoveryPointFinished:
		return true
	default:
		return false
	}
}
