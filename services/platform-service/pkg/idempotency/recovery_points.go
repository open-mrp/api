package idempotency

type RecoveryPoint string

const (
	RecoveryPointStarted  RecoveryPoint = "started"
	RecoveryPointFinished RecoveryPoint = "finished"
)

func (r RecoveryPoint) String() string {
	return string(r)
}
