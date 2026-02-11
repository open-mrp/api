package domain

type RecoveryPoint string

const (
	RecoveryPointStarted  RecoveryPoint = "auth:started"
	RecoveryPointFinished RecoveryPoint = "auth:finished"
)

func (r RecoveryPoint) String() string {
	return string(r)
}

func (r RecoveryPoint) IsFinished() bool {
	return r == RecoveryPointFinished
}

func (r RecoveryPoint) IsStarted() bool {
	return r == RecoveryPointStarted
}
