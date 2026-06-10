package domain

// Recovery points for multi-phase shipping operations.
const (
	RecoveryPointShipLabelsCreated  RecoveryPoint = "core:ship_labels_created"
	RecoveryPointVoidLabelsRefunded RecoveryPoint = "core:void_labels_refunded"
)
