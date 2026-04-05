package domain

type RecoveryPoint string

const (
	RecoveryPointStarted            RecoveryPoint = "auth:started"
	RecoveryPointCustomerCreated    RecoveryPoint = "auth:customer_created"
	RecoveryPointCoreAccountCreated RecoveryPoint = "auth:core_account_created"
	RecoveryPointAccountsCreated    RecoveryPoint = "auth:accounts_created"
	RecoveryPointFinished           RecoveryPoint = "auth:finished"
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

func (r RecoveryPoint) IsValid() bool {
	switch r {
	case RecoveryPointStarted, RecoveryPointCustomerCreated, RecoveryPointCoreAccountCreated, RecoveryPointAccountsCreated, RecoveryPointFinished:
		return true
	default:
		return false
	}
}
