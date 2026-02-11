package idempotency

type RecoveryPoint string

const (
	RecoveryPointStarted  RecoveryPoint = "started"
	RecoveryPointFinished RecoveryPoint = "finished"
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
