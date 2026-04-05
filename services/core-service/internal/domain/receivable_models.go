package domain

import "time"

// ReceivableEntry represents a single receivable invoice entry with its remaining balance.
type ReceivableEntry struct {
	InvoiceID        string
	InvoiceNumber    string
	PONumber         *string
	InvoicedAt       time.Time
	CustomerID       string
	CustomerNumber   string
	CustomerName     string
	RemainingBalance string
	IsPaidInFull     bool
}

// OpenCredit represents an open credit memo or payment with remaining balance.
type OpenCredit struct {
	ID             string
	Number         string
	CreatedAt      time.Time
	OriginalAmount string
	LeftoverAmount string
}

// ListReceivablesParams holds parameters for listing receivables.
type ListReceivablesParams struct {
	AccountID  string
	CutoffDate *time.Time
	Cursor     *string
	Limit      int32
	Query      *string
}

// ListReceivablesResult holds the result of listing receivables.
type ListReceivablesResult struct {
	Items      []ReceivableEntry
	PageString *string
}

// ListReceivablesByCustomerParams holds parameters for listing receivables by customer.
type ListReceivablesByCustomerParams struct {
	AccountID         string
	CustomerAccountID string
	CutoffDate        *time.Time
	Cursor            *string
	Limit             int32
	Query             *string
}

// ListReceivablesByCustomerResult holds the result of listing receivables by customer.
type ListReceivablesByCustomerResult struct {
	Items      []ReceivableEntry
	PageString *string
}

// EmailReceivablesParams holds parameters for emailing receivables to a customer.
type EmailReceivablesParams struct {
	CustomerAccountID string
	RecipientEmails   []string
}
