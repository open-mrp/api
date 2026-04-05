package constants

// TransactionType represents the type of a transaction.
type TransactionType string

const (
	// TransactionTypePayment indicates a payment transaction.
	TransactionTypePayment TransactionType = "payment"
	// TransactionTypeCreditMemo indicates a credit memo transaction.
	TransactionTypeCreditMemo TransactionType = "credit_memo"
	// TransactionTypeAdjustment indicates an adjustment transaction.
	TransactionTypeAdjustment TransactionType = "adjustment"
	// TransactionTypeRebate indicates a rebate transaction.
	TransactionTypeRebate TransactionType = "rebate"
)

func (m TransactionType) IsValid() bool {
	switch m {
	case TransactionTypePayment, TransactionTypeCreditMemo, TransactionTypeAdjustment, TransactionTypeRebate:
		return true
	default:
		return false
	}
}

func (m TransactionType) EnumValues() []string {
	return []string{
		string(TransactionTypePayment),
		string(TransactionTypeCreditMemo),
		string(TransactionTypeAdjustment),
		string(TransactionTypeRebate),
	}
}
