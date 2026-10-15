package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

type PaymentTerm struct {
	ID        string
	Name      string                  `audit:"name"`
	Status    constants.ArchiveStatus `audit:"status"`
	AccountID *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListPaymentTermsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	// IsActive narrows to active (true) or archived (false) terms; nil returns both.
	IsActive *bool
}

type ListPaymentTermsResult struct {
	PaymentTerms []*PaymentTerm
	PageInfo     pagination.PageInfo
}

type GetPaymentTermParams struct {
	AccountID     string
	PaymentTermID string
}

type CreatePaymentTermParams struct {
	AccountID string
	Name      string
	Status    constants.ArchiveStatus
}

type UpdatePaymentTermParams struct {
	AccountID     string
	PaymentTermID string
	Name          *string
	Status        *constants.ArchiveStatus
}

type DeletePaymentTermParams struct {
	AccountID     string
	PaymentTermID string
}

// PaymentTermReferences blocks deleting a payment term a customer or order still uses.
var PaymentTermReferences = RegisterReferences(References{
	Noun:       "payment term",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "customers", Table: "account_relation", Column: "payment_term_id"},
		{Label: "sales orders", Table: "sales_order", Column: "payment_term_id"},
	},
})
