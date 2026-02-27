package domain

type RecoveryPoint string

const (
	RecoveryPointStarted  RecoveryPoint = "core:started"
	RecoveryPointFinished RecoveryPoint = "core:finished"
)

func (r RecoveryPoint) String() string {
	return string(r)
}

func (r RecoveryPoint) IsValid() bool {
	switch r {
	case RecoveryPointStarted, RecoveryPointFinished:
		return true
	default:
		return false
	}
}
