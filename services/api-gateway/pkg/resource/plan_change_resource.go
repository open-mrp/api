package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Cost preview for a plan change.
type PlanChangeProration struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=plan_change_proration"`
	// Net amount in cents.
	NetAmount int64 `json:"net_amount"`
	// Formatted net amount for display (e.g., "$49.00").
	FormattedNetAmount string `json:"formatted_net_amount" validate:"required"`
	// Estimated monthly bill amount in cents after the change.
	MonthlyBillAmount int64 `json:"monthly_bill_amount"`
	// Formatted monthly bill amount for display.
	FormattedMonthlyBillAmount string `json:"formatted_monthly_bill_amount" validate:"required"`
	// Detailed line items from the cost preview.
	LineItems *List[PlanChangeLineItem] `json:"line_items"`
	// Whether the amounts are locally estimated rather than calculated by Stripe.
	IsEstimate bool `json:"is_estimate"`
}

// Line item in a plan change cost preview.
type PlanChangeLineItem struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=plan_change_line_item"`
	// Description of the line item.
	Description string `json:"description" validate:"required"`
	// Amount in cents (negative for credits).
	Amount int64 `json:"amount"`
}

var SamplePlanChangeProration = &PlanChangeProration{
	Object:                     constants.ObjectTypePlanChangeProration,
	NetAmount:                  4900,
	FormattedNetAmount:         "$49.00",
	MonthlyBillAmount:          4900,
	FormattedMonthlyBillAmount: "$49.00",
	LineItems: NewList([]PlanChangeLineItem{
		{Object: constants.ObjectTypePlanChangeLineItem, Description: "Professional plan \u2014 1 seat(s) \u00d7 $49.00/mo", Amount: 4900},
	}, PageInfo{}),
}

func (*PlanChangeProration) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanChangeProration)
}
