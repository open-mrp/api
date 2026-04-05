package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SamplePlanTypeIDFree = "pl_01gf7a8200eaj8fke1xvw4h50x"
const SamplePlanTypeIDStarter = "pl_01gf7a8200er3ar3pkfrb6kk29"
const SamplePlanTypeIDPro = "pl_01gf7a8200e9pvbd6bgyq395ae"

// PlanLimit represents a resource limit for a pricing plan.
type PlanLimit struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=plan_limit"`
	// The resource key this limit applies to (e.g., "sandboxes", "seats", "invoices").
	Key string `json:"key" validate:"required"`
	// The maximum allowed value, null means unlimited.
	Value *int `json:"value"`
}

// PricingPlan represents a pricing plan available for purchase.
type PricingPlan struct {
	// The unique ID of the plan.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pricing_plan"`
	// The display name of the plan.
	Name string `json:"name" validate:"required"`
	// The plan type code.
	PlanTypeCode constants.PublicPlanCode `json:"plan_type_code" validate:"required"`
	// The price per seat per month in dollars.
	PricePerSeat float64 `json:"price_per_seat"`
	// The flat monthly price in dollars, if applicable.
	PricePerMonth *float64 `json:"price_per_month"`
	// The minimum number of seats required for this plan.
	SeatMinimum *int `json:"seat_minimum"`
	// The resource limits for this plan.
	Limits []PlanLimit `json:"limits" validate:"required"`
	// The features to display on the pricing page.
	DisplayFeatures []string `json:"display_features" validate:"required"`
	// The display order for sorting plans on the pricing page.
	DisplayOrder int `json:"display_order"`
	// Whether this plan should be visually highlighted.
	IsHighlighted bool `json:"is_highlighted"`
	// The call-to-action button text for this plan.
	ButtonText string `json:"button_text" validate:"required"`
	// The name of the previous plan tier that this plan includes.
	IncludesPreviousPlan *string `json:"includes_previous_plan"`
}

var SamplePlanLimitSandboxes = &PlanLimit{
	Object: constants.ObjectTypePlanLimit,
	Key:    "sandboxes_maximum",
	Value:  new(1),
}

var SamplePlanLimitSeats = &PlanLimit{
	Object: constants.ObjectTypePlanLimit,
	Key:    "seats_maximum",
	Value:  new(5),
}

var SamplePlanLimitInvoices = &PlanLimit{
	Object: constants.ObjectTypePlanLimit,
	Key:    "invoices_maximum",
	Value:  new(10000),
}

var SamplePricingPlanFree = &PricingPlan{
	ID:           SamplePlanTypeIDFree,
	Object:       constants.ObjectTypePricingPlan,
	Name:         "Free",
	PlanTypeCode: constants.PublicPlanCodeFree,
	PricePerSeat: 0,
	PricePerMonth: func() *float64 {
		v := 0.0
		return &v
	}(),
	SeatMinimum: new(1),
	Limits: []PlanLimit{
		{Object: constants.ObjectTypePlanLimit, Key: "sandboxes_maximum", Value: new(1)},
		{Object: constants.ObjectTypePlanLimit, Key: "seats_maximum", Value: new(1)},
		{Object: constants.ObjectTypePlanLimit, Key: "invoices_maximum", Value: new(100)},
		{Object: constants.ObjectTypePlanLimit, Key: "batches_maximum", Value: new(10000)},
	},
	DisplayFeatures: []string{
		"1 sandbox environment",
		"Up to 100 invoices per month",
		"Basic support",
	},
	DisplayOrder:  1,
	IsHighlighted: false,
	ButtonText:    "Get Started",
	IncludesPreviousPlan: func() *string {
		v := ""
		return &v
	}(),
}

var SamplePricingPlanStarter = &PricingPlan{
	ID:           SamplePlanTypeIDStarter,
	Object:       constants.ObjectTypePricingPlan,
	Name:         "Starter",
	PlanTypeCode: constants.PublicPlanCodeStarter,
	PricePerSeat: 19,
	PricePerMonth: func() *float64 {
		v := 19.0
		return &v
	}(),
	SeatMinimum: new(1),
	Limits: []PlanLimit{
		{Object: constants.ObjectTypePlanLimit, Key: "sandboxes_maximum", Value: new(3)},
		{Object: constants.ObjectTypePlanLimit, Key: "seats_maximum", Value: new(5)},
		{Object: constants.ObjectTypePlanLimit, Key: "invoices_maximum", Value: new(10000)},
		{Object: constants.ObjectTypePlanLimit, Key: "batches_maximum", Value: new(10000)},
	},
	DisplayFeatures: []string{
		"3 sandbox environments",
		"Up to 5 team seats",
		"Unlimited invoices",
		"Priority email support",
	},
	DisplayOrder:  2,
	IsHighlighted: true,
	ButtonText:    "Get Started",
	IncludesPreviousPlan: func() *string {
		v := "Free"
		return &v
	}(),
}

var SamplePricingPlanPro = &PricingPlan{
	ID:           SamplePlanTypeIDPro,
	Name:         "Professional",
	Object:       constants.ObjectTypePricingPlan,
	PlanTypeCode: constants.PublicPlanCodePro,
	PricePerSeat: 49,
	PricePerMonth: func() *float64 {
		v := 147.0
		return &v
	}(),
	SeatMinimum: new(3),
	Limits: []PlanLimit{
		{Object: constants.ObjectTypePlanLimit, Key: "sandboxes_maximum", Value: new(999)},
		{Object: constants.ObjectTypePlanLimit, Key: "seats_maximum", Value: new(999)},
		{Object: constants.ObjectTypePlanLimit, Key: "invoices_maximum", Value: new(999)},
		{Object: constants.ObjectTypePlanLimit, Key: "batches_maximum", Value: new(10000)},
	},
	DisplayFeatures: []string{
		"Unlimited sandbox environments",
		"Unlimited team seats",
		"Unlimited invoices",
		"Dedicated support",
		"Custom integrations",
	},
	DisplayOrder:  3,
	IsHighlighted: false,
	ButtonText:    "Contact Sales",
	IncludesPreviousPlan: func() *string {
		v := "Starter"
		return &v
	}(),
}

func (*PlanLimit) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePlanLimitSandboxes)
}

func (*PricingPlan) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePricingPlanStarter)
}
