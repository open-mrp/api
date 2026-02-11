package domain

type RecoveryPoint string

const (
	RecoveryPointStarted  RecoveryPoint = "core:started"
	RecoveryPointFinished RecoveryPoint = "core:finished"
)
