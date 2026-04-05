package constants

// TransactionMethod represents the method of a transaction.
type TransactionMethod string

const (
	// TransactionMethodCash indicates a cash transaction.
	TransactionMethodCash TransactionMethod = "cash"
	// TransactionMethodCheck indicates a check transaction.
	TransactionMethodCheck TransactionMethod = "check"
	// TransactionMethodCreditCard indicates a credit card transaction.
	TransactionMethodCreditCard TransactionMethod = "credit_card"
	// TransactionMethodGiftCard indicates a gift card transaction.
	TransactionMethodGiftCard TransactionMethod = "gift_card"
	// TransactionMethodACH indicates an ACH transaction.
	TransactionMethodACH TransactionMethod = "ach"
)

func (m TransactionMethod) IsValid() bool {
	switch m {
	case TransactionMethodCash, TransactionMethodCheck, TransactionMethodCreditCard,
		TransactionMethodGiftCard, TransactionMethodACH:
		return true
	default:
		return false
	}
}

func (m TransactionMethod) EnumValues() []string {
	return []string{
		string(TransactionMethodCash),
		string(TransactionMethodCheck),
		string(TransactionMethodCreditCard),
		string(TransactionMethodGiftCard),
		string(TransactionMethodACH),
	}
}
