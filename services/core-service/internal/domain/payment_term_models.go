package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

type PaymentTerm struct {
	ID        string
	Name      string                      `audit:"name"`
	Status    constants.PaymentTermStatus `audit:"status"`
	AccountID *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListPaymentTermsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
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
}

type UpdatePaymentTermParams struct {
	AccountID     string
	PaymentTermID string
	Name          *string
}

type DeletePaymentTermParams struct {
	AccountID     string
	PaymentTermID string
}
