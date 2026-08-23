package constants

// TransactionAllocationStatus filters a list of transactions by how much of the transaction has been applied to invoices.
type TransactionAllocationStatus string

const (
	// TransactionAllocationStatusAllocated returns transactions marked fully applied to invoices.
	TransactionAllocationStatusAllocated TransactionAllocationStatus = "allocated"
	// TransactionAllocationStatusUnallocated returns transactions still counted as an open credit.
	TransactionAllocationStatusUnallocated TransactionAllocationStatus = "unallocated"
)

func (s TransactionAllocationStatus) IsValid() bool {
	switch s {
	case TransactionAllocationStatusAllocated, TransactionAllocationStatusUnallocated:
		return true
	default:
		return false
	}
}

func (s TransactionAllocationStatus) EnumValues() []string {
	return []string{
		string(TransactionAllocationStatusAllocated),
		string(TransactionAllocationStatusUnallocated),
	}
}
