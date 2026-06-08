package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// RecoveryPoint constants for invoice operations.
const (
	InvoiceRecoveryPointStarted  RecoveryPoint = "started"
	InvoiceRecoveryPointFinished RecoveryPoint = "finished"
)

// InvoiceSummary represents a lightweight invoice for list views.
type InvoiceSummary struct {
	ID                       string
	Number                   string                 `audit:"number"`
	Note                     *string                `audit:"note"`
	CustomerID               string                 `audit:"customer_id"`
	CustomerName             string                 `audit:"customer_name"`
	CustomerNumber           string                 `audit:"customer_number"`
	CustomerStatusCode       *string                `audit:"customer_status_code"`
	CustomerCommissionPolicy *string                `audit:"customer_commission_policy"`
	CustomerIsEdiEnabled     bool                   `audit:"customer_is_edi_enabled"`
	OrderID                  string                 `audit:"order_id"`
	OrderNumber              string                 `audit:"order_number"`
	ShipmentID               *string                `audit:"shipment_id"`
	LineCount                int32                  `audit:"line_count"`
	BillingAddressID         string                 `audit:"billing_address_id"`
	BillingAddressName       *string                `audit:"billing_address_name"`
	BillingAddressLine1      *string                `audit:"billing_address_line_1"`
	BillingAddressLine2      *string                `audit:"billing_address_line_2"`
	BillingAddressCity       *string                `audit:"billing_address_city"`
	BillingAddressState      *string                `audit:"billing_address_state"`
	BillingAddressZip        *string                `audit:"billing_address_zip"`
	BillingAddressCountry    string                 `audit:"billing_address_country"`
	PriorityCode             constants.PriorityCode `audit:"priority_code"`
	PaymentTermID            *string                `audit:"payment_term_id"`
	PaymentTermName          *string                `audit:"payment_term_name"`
	PaymentTermIsActive      *bool                  `audit:"payment_term_is_active"`
	PaymentTermDays          *int32                 `audit:"payment_term_days"`
	IsPaidInFull             bool                   `audit:"is_paid_in_full"`
	IsEdiSent                bool                   `audit:"is_edi_sent"`
	HasBeenSent              bool                   `audit:"has_been_sent"`
	TotalInvoiced            string                 `audit:"total_invoiced"`
	AcceptsInvoiceEmails     bool                   `audit:"accepts_invoice_emails"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// Invoice represents a full invoice with expandable lines and allocations.
type Invoice struct {
	ID                    string
	Number                string  `audit:"number"`
	Note                  *string `audit:"note"`
	OrderID               string  `audit:"order_id"`
	OrderNumber           string  `audit:"order_number"`
	CustomerID            string  `audit:"customer_id"`
	PaymentTermID         *string `audit:"payment_term_id"`
	BillingAddressID      string  `audit:"billing_address_id"`
	BillingAddressName    *string `audit:"billing_address_name"`
	BillingAddressLine1   *string `audit:"billing_address_line_1"`
	BillingAddressLine2   *string `audit:"billing_address_line_2"`
	BillingAddressCity    *string `audit:"billing_address_city"`
	BillingAddressState   *string `audit:"billing_address_state"`
	BillingAddressZip     *string `audit:"billing_address_zip"`
	BillingAddressCountry string  `audit:"billing_address_country"`
	ShipmentID            *string `audit:"shipment_id"`
	ShipmentNumber        *string `audit:"shipment_number"`
	IsPaidInFull          bool    `audit:"is_paid_in_full"`
	IsOverPaid            bool    `audit:"is_over_paid"`
	IsEdiSent             bool    `audit:"is_edi_sent"`
	HasBeenSent           bool    `audit:"has_been_sent"`
	AcceptsInvoiceEmails  bool    `audit:"accepts_invoice_emails"`
	Lines                 []*InvoiceLine
	Allocations           []*InvoiceAllocation
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// InvoiceLine represents a line item in an invoice.
type InvoiceLine struct {
	ID               string
	QuantityID       string
	QuantityValue    string
	QuantityUnitID   string
	QuantityUnitAbbr string
	UnitPriceID      string
	UnitPriceValue   string
	UnitPriceNumUnit string
	UnitPriceDenUnit string
	OrderLineID      string
	OrderLineItemID  *string
	OrderLineItemSKU *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// InvoiceAllocation represents a transaction allocation against an invoice.
type InvoiceAllocation struct {
	ID             string
	TransactionID  string
	AmountID       string
	AmountValue    string
	AmountUnitID   string
	AmountUnitAbbr string
	Note           *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// InvoiceForPayment represents an invoice in the customer payment context.
type InvoiceForPayment struct {
	ID                 string
	Number             string
	CustomerPO         *string
	CustomerID         string
	CustomerName       string
	CustomerNumber     string
	IsParentAccount    bool
	ParentAccountID    *string
	IsPrepaid          bool
	BillingAddressID   *string
	BillingAddressName *string
	InvoiceTotal       string
	IsPaidInFull       bool
	Allocations        []*InvoiceAllocation
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ListInvoicesParams holds parameters for listing invoices.
type ListInvoicesParams struct {
	AccountID        string
	Cursor           *string
	Limit            int32
	Query            *string
	Status           *string
	ItemIDs          []string
	CustomerIDs      []string
	ProductLineIDs   []string
	CustomerGroupIDs []string
	SalesRepIDs      []string
	StartDate        *time.Time
	EndDate          *time.Time
}

// ListInvoicesResult holds the result of listing invoices.
type ListInvoicesResult struct {
	Invoices []*InvoiceSummary
	PageInfo pagination.PageInfo
}

// GetInvoiceParams holds parameters for getting a single invoice.
type GetInvoiceParams struct {
	AccountID string
	InvoiceID string
	Includes  []string
}

// UpdateInvoiceParams holds parameters for updating an invoice.
type UpdateInvoiceParams struct {
	AccountID    string
	InvoiceID    string
	Note         *string
	HasBeenSent  *bool
	IsEdiSent    *bool
	IsPaidInFull *bool
}

// ListCustomerInvoicesParams holds parameters for listing invoices by customer.
type ListCustomerInvoicesParams struct {
	AccountID            string
	CustomerAccountID    string
	Cursor               *string
	Limit                int32
	Query                *string
	IncludeChildAccounts bool
}

// ListCustomerInvoicesResult holds the result of listing customer invoices.
type ListCustomerInvoicesResult struct {
	Invoices []*InvoiceForPayment
	PageInfo pagination.PageInfo
}
