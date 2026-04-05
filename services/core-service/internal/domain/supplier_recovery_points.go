package domain

// Supplier-specific recovery points are not needed since CreateSupplier and
// UpdateSupplier only perform local atomic mutations within a single transaction.
// They reuse RecoveryPointStarted and RecoveryPointFinished from recovery_points.go.
