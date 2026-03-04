package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
)

// PlanChangeProration represents the proration calculation for a plan change.
type PlanChangeProration struct {
	// The credit amount in cents from unused time on the current plan.
	CreditAmount int64 `json:"credit_amount"`
	// The charge amount in cents for the new plan.
	ChargeAmount int64 `json:"charge_amount"`
	// The net amount in cents (charge minus credit).
	NetAmount int64 `json:"net_amount"`
	// The formatted net amount for display (e.g., "$49.00").
	FormattedNetAmount string `json:"formatted_net_amount" validate:"required"`
	// Whether the net amount is a credit (negative).
	IsCredit bool `json:"is_credit"`
	// The total invoice amount in cents that would be charged immediately.
	TotalInvoiceAmount int64 `json:"total_invoice_amount"`
	// The formatted total invoice amount for display.
	FormattedTotalInvoiceAmount string `json:"formatted_total_invoice_amount" validate:"required"`
	// The estimated monthly bill amount in cents after the change.
	MonthlyBillAmount int64 `json:"monthly_bill_amount"`
	// The formatted monthly bill amount for display.
	FormattedMonthlyBillAmount string `json:"formatted_monthly_bill_amount" validate:"required"`
	// Detailed line items from the proration calculation.
	LineItems []PlanChangeLineItem `json:"line_items"`
}

// PlanChangeLineItem represents a single line item in a proration preview.
type PlanChangeLineItem struct {
	// Description of the line item.
	Description string `json:"description" validate:"required"`
	// Amount in cents (negative for credits).
	Amount int64 `json:"amount"`
	// Whether this line item is a proration adjustment.
	IsProration bool `json:"is_proration"`
}

func (*PlanChangeProration) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanChangeProration)
}

var SamplePlanChangeProration = &PlanChangeProration{
	CreditAmount:                0,
	ChargeAmount:                4900,
	NetAmount:                   4900,
	FormattedNetAmount:          "$49.00",
	IsCredit:                    false,
	TotalInvoiceAmount:          4900,
	FormattedTotalInvoiceAmount: "$49.00",
	MonthlyBillAmount:           4900,
	FormattedMonthlyBillAmount:  "$49.00",
	LineItems: []PlanChangeLineItem{
		{Description: "Professional plan — 1 seat(s) × $49.00/mo", Amount: 4900, IsProration: false},
	},
}
