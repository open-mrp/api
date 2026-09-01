package domain

// DuplicateCheckType represents the type of duplicate check to perform.
type DuplicateCheckType string

const (
	DuplicateCheckTypeInvoiceNumber DuplicateCheckType = "invoice_number"
	DuplicateCheckTypeOrderNumber   DuplicateCheckType = "order_number"
	DuplicateCheckTypeCustomerPO    DuplicateCheckType = "customer_po_number"
)

func (t DuplicateCheckType) IsValid() bool {
	switch t {
	case DuplicateCheckTypeInvoiceNumber,
		DuplicateCheckTypeOrderNumber,
		DuplicateCheckTypeCustomerPO:
		return true
	}
	return false
}

func (t DuplicateCheckType) EnumValues() []string {
	return []string{
		string(DuplicateCheckTypeInvoiceNumber),
		string(DuplicateCheckTypeOrderNumber),
		string(DuplicateCheckTypeCustomerPO),
	}
}

// CheckDuplicateParams holds parameters for a duplicate check.
type CheckDuplicateParams struct {
	Type         DuplicateCheckType
	RecordNumber string
	CustomerID   *string
}

// CheckDuplicateResult holds the result of a duplicate check.
type CheckDuplicateResult struct {
	IsDuplicate bool
	Message     *string
}

// EmailRecordType represents the type of record to email.
type EmailRecordType string

const (
	EmailRecordTypeInvoice       EmailRecordType = "invoice"
	EmailRecordTypeSalesOrder    EmailRecordType = "sales_order"
	EmailRecordTypePurchaseOrder EmailRecordType = "purchase_order"
)

func (t EmailRecordType) IsValid() bool {
	switch t {
	case EmailRecordTypeInvoice,
		EmailRecordTypeSalesOrder,
		EmailRecordTypePurchaseOrder:
		return true
	}
	return false
}

func (t EmailRecordType) EnumValues() []string {
	return []string{
		string(EmailRecordTypeInvoice),
		string(EmailRecordTypeSalesOrder),
		string(EmailRecordTypePurchaseOrder),
	}
}

// EmailRecordParams holds parameters for emailing a record.
type EmailRecordParams struct {
	ID   string
	Type EmailRecordType
}

// RequestDemoParams holds parameters for submitting a demo request.
type RequestDemoParams struct {
	Name        string
	Email       string
	Company     string
	PhoneNumber *string
	Message     *string
}

// SubmitFeedbackParams holds parameters for submitting user feedback.
type SubmitFeedbackParams struct {
	Question string
	Answer   string
	PageURL  *string
}

// InvoiceIssuedEvent is the payload for CoreEventInvoiceIssued. The account rides on the envelope
// identity, as it does for every message.
//
// Which copies are owed is carried rather than derived: a ship can be told not to mail the customer,
// and a manual resend never mails the rep. Neither is recoverable from the invoice row once the
// action that raised it has returned.
type InvoiceIssuedEvent struct {
	InvoiceID string `json:"invoice_id"`
	// EmailCustomer mails the invoice's recipients and flags the invoice sent.
	EmailCustomer bool `json:"email_customer"`
	// EmailSalesRep copies the order's sales rep. Never flags the invoice sent — that records whether
	// the customer received it, and the rep copy is an internal notification.
	EmailSalesRep bool `json:"email_sales_rep"`
}

// SalesOrderAcknowledgedEvent is the payload for CoreEventSalesOrderAcknowledged.
type SalesOrderAcknowledgedEvent struct {
	SalesOrderID string `json:"sales_order_id"`
}

// PurchaseOrderSubmittedEvent is the payload for CoreEventPurchaseOrderSubmitted.
type PurchaseOrderSubmittedEvent struct {
	PurchaseOrderID string `json:"purchase_order_id"`
}

// SendInvoiceEmailParams holds parameters for mailing an invoice outside a request.
type SendInvoiceEmailParams struct {
	AccountID     string
	InvoiceID     string
	EmailCustomer bool
	EmailSalesRep bool
}

// SendSalesOrderAcknowledgementParams holds parameters for mailing an order acknowledgement outside a request.
type SendSalesOrderAcknowledgementParams struct {
	AccountID    string
	SalesOrderID string
}

// SendPurchaseOrderSubmissionParams holds parameters for mailing a purchase order submission outside a request.
type SendPurchaseOrderSubmissionParams struct {
	AccountID       string
	PurchaseOrderID string
}
