package domain

type RecoveryPoint string

// Flows that only perform local atomic mutations within a single transaction (for example CreateSupplier and UpdateSupplier) reuse these generic points; multi-phase flows declare their own in domain-specific *_recovery_points.go files.
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
