package constants

// Names a row in `account_plan_limit`, the per-plan caps an account is billed under. The value is
// the cap; a missing row or a NULL value means unlimited.
type AccountPlanLimitKey string

const (
	// Caps how many invoices an account may create per billing period.
	AccountPlanLimitInvoicesMaximum AccountPlanLimitKey = "invoices_maximum"
	// Caps how many batches an account may create per billing period.
	AccountPlanLimitBatchesMaximum AccountPlanLimitKey = "batches_maximum"
	// Caps how many users may hold a seat on the account.
	AccountPlanLimitSeatsMaximum AccountPlanLimitKey = "seats_maximum"
	// Caps how many sandboxes the account may hold at once.
	AccountPlanLimitSandboxesMaximum AccountPlanLimitKey = "sandboxes_maximum"
)

func (k AccountPlanLimitKey) IsValid() bool {
	switch k {
	case AccountPlanLimitInvoicesMaximum, AccountPlanLimitBatchesMaximum, AccountPlanLimitSeatsMaximum, AccountPlanLimitSandboxesMaximum:
		return true
	default:
		return false
	}
}

func (k *AccountPlanLimitKey) StringPtr() *string {
	if k == nil {
		return nil
	}
	s := string(*k)
	return &s
}

func (k AccountPlanLimitKey) EnumValues() []string {
	return []string{
		string(AccountPlanLimitInvoicesMaximum),
		string(AccountPlanLimitBatchesMaximum),
		string(AccountPlanLimitSeatsMaximum),
		string(AccountPlanLimitSandboxesMaximum),
	}
}
