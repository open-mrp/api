package domain

type RecoveryPoint string

const (
	RecoveryPointStarted  RecoveryPoint = "notification:started"
	RecoveryPointFinished RecoveryPoint = "notification:finished"
)
