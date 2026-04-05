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
