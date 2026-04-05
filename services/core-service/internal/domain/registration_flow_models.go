package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type RegistrationFlow struct {
	ID                   string
	Name                 string `audit:"name"`
	AccountID            string
	CustomerGroupOptions []*RegistrationFlowOption `audit:"customer_group_options"`
	PaymentTermOptions   []*RegistrationFlowOption `audit:"payment_term_options"`
	ShippingTermOptions  []*RegistrationFlowOption `audit:"shipping_term_options"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RegistrationFlowOption struct {
	ID   string
	Name string
}

type ListRegistrationFlowsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListRegistrationFlowsResult struct {
	RegistrationFlows []*RegistrationFlow
	PageInfo          pagination.PageInfo
}

type CreateRegistrationFlowParams struct {
	AccountID        string
	Name             string
	CustomerGroupIDs []string
	PaymentTermIDs   []string
	ShippingTermIDs  []string
}

type UpdateRegistrationFlowParams struct {
	AccountID           string
	RegistrationFlowID  string
	Name                *string
	CustomerGroupIDs    []string
	PaymentTermIDs      []string
	ShippingTermIDs     []string
	HasCustomerGroupIDs bool
	HasPaymentTermIDs   bool
	HasShippingTermIDs  bool
}

type DeleteRegistrationFlowParams struct {
	AccountID          string
	RegistrationFlowID string
}
