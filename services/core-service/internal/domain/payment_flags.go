package domain

import "github.com/shopspring/decimal"

// InvoicePaymentFlagsFor decides an invoice's paid-in-full and over-paid flags as the dashboard's
// InvoiceRepo.evaluatePaymentStatusAndUpdate did: the balance (invoiced total less every allocation
// against the invoice) is rounded to the cent, half away from zero like decimal.js toDP(2); a zero
// balance is paid in full, a negative one is also over paid. An invoice with nothing invoiced and
// nothing allocated therefore reads as paid.
func InvoicePaymentFlagsFor(invoicedTotal, allocatedTotal decimal.Decimal) (isPaidInFull, isOverPaid bool) {
	balance := invoicedTotal.Round(2).Sub(allocatedTotal).Round(2)
	switch balance.Sign() {
	case 0:
		return true, false
	case -1:
		return true, true
	default:
		return false, false
	}
}

// TransactionFullyAllocated decides a transaction's fully-allocated flag as the dashboard's
// TransactionRepo.evaluateStatusAndUpdate did: its amount less every allocation drawn from it,
// rounded to the cent, is at most zero.
func TransactionFullyAllocated(amount, allocatedTotal decimal.Decimal) bool {
	return amount.Sub(allocatedTotal).Round(2).Sign() <= 0
}

// PaymentTotals are the stored amounts a payment flag is derived from, as exact decimal strings.
type PaymentTotals struct {
	ID        string
	Total     string
	Allocated string
}

// InvoicePaymentTotals are an invoice's payment totals with its current paid-in-full flag and who, if
// anyone, set that flag by hand.
type InvoicePaymentTotals struct {
	PaymentTotals
	Number       string
	IsPaidInFull bool
	MarkedByID   *string
}
