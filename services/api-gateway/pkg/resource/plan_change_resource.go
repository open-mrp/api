package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/ptrutil"
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

type NextAction struct {
	Type        NextActionType `json:"type" validate:"required"`
	RedirectURL string         `json:"redirect_url" validate:"required"`
}

func (*NextAction) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNextAction)
}

type PlanChangeProration struct {
	CreditAmount       int64  `json:"credit_amount"`
	ChargeAmount       int64  `json:"charge_amount"`
	NetAmount          int64  `json:"net_amount"`
	FormattedNetAmount string `json:"formatted_net_amount" validate:"required"`
}

func (*PlanChangeProration) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanChangeProration)
}

type PlanChange struct {
	ID            string               `json:"id" validate:"required"`
	Object        constants.ObjectType `json:"object" validate:"required,enum=plan_change"`
	Status        PlanChangeStatus     `json:"status" validate:"required"`
	CurrentPlanID *string              `json:"current_plan_id"`
	TargetPlanID  string               `json:"target_plan_id" validate:"required"`
	NextAction    *NextAction          `json:"next_action"`
	Proration     *PlanChangeProration `json:"proration"`
	CreatedAt     time.Time            `json:"created_at" validate:"required"`
	CompletedAt   *time.Time           `json:"completed_at"`
}

func (*PlanChange) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanChange)
}

var SampleNextAction = &NextAction{
	Type:        NextActionTypeRedirectToCheckout,
	RedirectURL: "https://checkout.stripe.com/c/pay/cs_test_abc123xyz789",
}

var SamplePlanChangeProration = &PlanChangeProration{
	CreditAmount:       0,
	ChargeAmount:       4900,
	NetAmount:          4900,
	FormattedNetAmount: "$49.00",
}

var SamplePlanChange = &PlanChange{
	ID:            SamplePlanChangeID,
	Object:        constants.ObjectTypePlanChange,
	Status:        PlanChangeStatusRequiresPayment,
	CurrentPlanID: ptrutil.Ptr(SamplePlanTypeIDFree),
	TargetPlanID:  SamplePlanTypeIDPro,
	NextAction:    SampleNextAction,
	Proration:     SamplePlanChangeProration,
	CreatedAt:     time.Now(),
	CompletedAt:   nil,
}

var SamplePlanChangeSucceeded = &PlanChange{
	ID:            SamplePlanChangeID,
	Object:        constants.ObjectTypePlanChange,
	Status:        PlanChangeStatusSucceeded,
	CurrentPlanID: ptrutil.Ptr(SamplePlanTypeIDFree),
	TargetPlanID:  SamplePlanTypeIDPro,
	NextAction:    nil,
	Proration:     SamplePlanChangeProration,
	CreatedAt:     time.Now(),
	CompletedAt:   ptrutil.Ptr(time.Now()),
}

type PlanChangePreview struct {
	TargetPlanID string               `json:"target_plan_id" validate:"required"`
	Proration    *PlanChangeProration `json:"proration,omitempty"`
}

func (*PlanChangePreview) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanChangePreview)
}

var SamplePlanChangePreview = &PlanChangePreview{
	TargetPlanID: SamplePlanTypeIDPro,
	Proration:    SamplePlanChangeProration,
}
