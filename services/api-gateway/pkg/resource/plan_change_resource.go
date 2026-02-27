package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePlanChangeID = "plch_01gf7a8200eaj8fke1xvw4h50x"

type PlanChangeStatus string

const (
	PlanChangeStatusPending         PlanChangeStatus = "pending"
	PlanChangeStatusRequiresPayment PlanChangeStatus = "requires_payment"
	PlanChangeStatusProcessing      PlanChangeStatus = "processing"
	PlanChangeStatusSucceeded       PlanChangeStatus = "succeeded"
	PlanChangeStatusFailed          PlanChangeStatus = "failed"
)

type NextActionType string

const (
	NextActionTypeRedirectToCheckout NextActionType = "redirect_to_checkout"
)

// NextAction describes an action the client must take to continue a plan change.
type NextAction struct {
	// The type of action required (e.g., "redirect_to_checkout").
	Type NextActionType `json:"type" validate:"required"`
	// The URL to redirect the user to.
	RedirectURL string `json:"redirect_url" validate:"required"`
}

func (*NextAction) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNextAction)
}

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

// PlanChange represents a request to switch from one pricing plan to another.
type PlanChange struct {
	// The unique identifier for this plan change.
	ID string `json:"id" validate:"required"`
	// The object type.
	Object constants.ObjectType `json:"object" validate:"required,enum=plan_change"`
	// The current status of the plan change (pending, requires_payment, processing, succeeded, failed).
	Status PlanChangeStatus `json:"status" validate:"required"`
	// The ID of the current plan, null if on the free plan.
	CurrentPlanID *string `json:"current_plan_id"`
	// The ID of the target plan to switch to.
	TargetPlanID string `json:"target_plan_id" validate:"required"`
	// The next action required to complete the plan change, if any.
	NextAction *NextAction `json:"next_action"`
	// The proration details for this plan change.
	Proration *PlanChangeProration `json:"proration"`
	// When this plan change was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this plan change was completed, null if still in progress.
	CompletedAt *time.Time `json:"completed_at"`
}

func (*PlanChange) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanChange)
}

var SampleNextAction = &NextAction{
	Type:        NextActionTypeRedirectToCheckout,
	RedirectURL: SampleCheckoutURL,
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

var SamplePlanChange = &PlanChange{
	ID:            SamplePlanChangeID,
	Object:        constants.ObjectTypePlanChange,
	Status:        PlanChangeStatusRequiresPayment,
	CurrentPlanID: new(SamplePlanTypeIDFree),
	TargetPlanID:  SamplePlanTypeIDPro,
	NextAction:    SampleNextAction,
	Proration:     SamplePlanChangeProration,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CompletedAt:   nil,
}

var SamplePlanChangeSucceeded = &PlanChange{
	ID:            SamplePlanChangeID,
	Object:        constants.ObjectTypePlanChange,
	Status:        PlanChangeStatusSucceeded,
	CurrentPlanID: new(SamplePlanTypeIDFree),
	TargetPlanID:  SamplePlanTypeIDPro,
	NextAction:    nil,
	Proration:     SamplePlanChangeProration,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CompletedAt:   timeutil.TimestampToTimePtr(sampleCompletedAtTimestamp),
}

// PlanChangePreview represents a preview of a plan change before it is initiated.
type PlanChangePreview struct {
	// The ID of the target plan to switch to.
	TargetPlanID string `json:"target_plan_id" validate:"required"`
	// The estimated proration details for this plan change.
	Proration *PlanChangeProration `json:"proration,omitempty"`
}

func (*PlanChangePreview) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanChangePreview)
}

var SamplePlanChangePreview = &PlanChangePreview{
	TargetPlanID: SamplePlanTypeIDPro,
	Proration:    SamplePlanChangeProration,
}
