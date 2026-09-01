package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

// Bills a customer for goods shipped against a sales order; read, list and update all return it.
type Invoice struct {
	ID                       string
	Number                   string                 `audit:"number"`
	Note                     *string                `audit:"note"`
	OrderID                  string                 `audit:"order_id"`
	OrderNumber              string                 `audit:"order_number"`
	PriorityCode             constants.PriorityCode `audit:"priority_code"`
	CustomerID               string                 `audit:"customer_id"`
	CustomerName             string                 `audit:"customer_name"`
	CustomerNumber           string                 `audit:"customer_number"`
	CustomerStatusCode       *string                `audit:"customer_status_code"`
	CustomerCommissionPolicy *string                `audit:"customer_commission_policy"`
	CustomerIsEdiEnabled     bool                   `audit:"customer_is_edi_enabled"`
	PaymentTermID            *string                `audit:"payment_term_id"`
	PaymentTermName          *string                `audit:"payment_term_name"`
	PaymentTermIsActive      *bool                  `audit:"payment_term_is_active"`
	BillingAddressID         string                 `audit:"billing_address_id"`
	BillingAddressName       *string                `audit:"billing_address_name"`
	BillingAddressLine1      *string                `audit:"billing_address_line_1"`
	BillingAddressLine2      *string                `audit:"billing_address_line_2"`
	BillingAddressCity       *string                `audit:"billing_address_city"`
	BillingAddressState      *string                `audit:"billing_address_state"`
	BillingAddressZip        *string                `audit:"billing_address_zip"`
	BillingAddressCountry    string                 `audit:"billing_address_country"`
	ShipmentID               *string                `audit:"shipment_id"`
	ShipmentNumber           *string                `audit:"shipment_number"`
	LineCount                int32                  `audit:"line_count"`
	TotalInvoiced            string                 `audit:"total_invoiced"`
	IsPaidInFull             bool                   `audit:"is_paid_in_full"`
	IsOverPaid               bool                   `audit:"is_over_paid"`
	IsEdiSent                bool                   `audit:"is_edi_sent"`
	HasBeenSent              bool                   `audit:"has_been_sent"`
	AcceptsInvoiceEmails     bool                   `audit:"accepts_invoice_emails"`
	Lines                    []*InvoiceLine
	Allocations              []*InvoiceAllocation
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// InvoiceLine represents a line item in an invoice.
type InvoiceLine struct {
	ID               string
	QuantityID       string
	QuantityValue    string
	QuantityUnitID   string
	QuantityUnitAbbr string
	QuantityUnitName string
	UnitPriceID      string
	UnitPriceValue   string
	UnitPriceNumUnit string
	UnitPriceDenUnit string
	// UnitPriceDenUnitAbbr labels the price's pricing unit ("$8.50 / pr"). It is the rate's own denominator, which can differ from the line's quantity unit.
	UnitPriceDenUnitAbbr string
	OrderLineID          string
	OrderLineItemID      *string
	OrderLineItemNumber  *int32
	OrderLineProductID   *string
	OrderLineQtyOrdered  string
	OrderLineItemSKU     *string
	OrderLineDescription *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
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
	Includes         []string
}

// Holds one page of invoices plus its cursors.
type ListInvoicesResult struct {
	Invoices []*Invoice
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
	Note         field.Clearable[string]
	HasBeenSent  *bool
	IsEdiSent    *bool
	IsPaidInFull *bool
	Includes     []string
}

// ListCustomerInvoicesParams holds parameters for listing invoices by customer.
type ListCustomerInvoicesParams struct {
	AccountID         string
	CustomerAccountID string
	Cursor            *string
	Limit             int32
	Query             *string
	Includes          []string
}

// ListCustomerInvoicesResult holds the result of listing customer invoices.
type ListCustomerInvoicesResult struct {
	Invoices []*InvoiceForPayment
	PageInfo pagination.PageInfo
}

// InvoicePaymentFlags holds the recomputed payment flags for a single invoice, derived from its transaction allocations vs. its invoiced total.
type InvoicePaymentFlags struct {
	InvoiceID    string
	IsPaidInFull bool
	IsOverPaid   bool
}

// Carries everything CreateFromShipment needs, resolved by the service inside the ship transaction.
type CreateInvoiceFromShipmentParams struct {
	AccountID    string
	InvoiceID    string
	Number       string
	SalesOrderID string
	ShippedLines []InvoiceLineDraft
}

// Describes one line to write: the order line billed and the quantity billed for it.
type InvoiceLineDraft struct {
	SalesOrderLineID string
	QuantityValue    string
	QuantityUnitID   string
}
