package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

const SamplePlanTypeIDFree = "pl_1nz4huuc8n5n"
const SamplePlanTypeIDStarter = "pl_ktxa0uvfgxe9"
const SamplePlanTypeIDPro = "pl_ahxp2c58ykmk"

// Resource limit for a pricing plan.
type PlanLimit struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=plan_limit"`
	// Resource this limit applies to.
	//
	// - `seats_maximum`: users that can belong to the account.
	// - `sandboxes_maximum`: sandbox environments the account can have.
	// - `invoices_maximum`: invoices the account can issue per billing period.
	// - `batches_maximum`: production batches the account can create per billing period.
	Key constants.AccountPlanLimitKey `json:"key" validate:"required"`
	// Maximum allowed value.
	//
	// Null means the plan places no limit on this resource.
	Value *int `json:"value"`
}

// A subscription plan an account can be billed on.
type PricingPlan struct {
	// Plan ID.
	//
	// Pass this value when previewing or performing a plan switch.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pricing_plan"`
	// Display name of the plan.
	Name string `json:"name" validate:"required"`
	// Tier of this pricing plan.
	PlanTypeCode constants.PublicPlanCode `json:"plan_type" validate:"required"`
	// Price per seat per month in dollars.
	PricePerSeat float64 `json:"price_per_seat"`
	// Per-seat price override in dollars used in place of `price_per_seat` when set.
	//
	// The monthly bill multiplies this by the number of seats (at least `seat_minimum`). `null` or `0` falls back to `price_per_seat`.
	PricePerMonth *float64 `json:"price_per_month"`
	// Minimum number of seats billed on this plan.
	//
	// When the account has fewer users than this minimum, the monthly bill is still calculated using this seat count. Null means no minimum.
	SeatMinimum *int `json:"seat_minimum"`
	// Resource limits for this plan.
	Limits *List[PlanLimit] `json:"limits" validate:"required"`
	// Features to display on the pricing page.
	DisplayFeatures []string `json:"display_features" validate:"required"`
	// Display order for sorting on the pricing page.
	DisplayOrder int `json:"display_order"`
	// Whether this plan should be visually highlighted.
	IsHighlighted bool `json:"is_highlighted"`
	// Call-to-action button text.
	ButtonText string `json:"button_text" validate:"required"`
	// Name of the lower plan tier whose features this plan also includes, for an "everything in X, plus..." callout on the pricing page.
	//
	// Null for the entry tier, which builds on no prior plan.
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
	Limits: NewList([]PlanLimit{
		{Object: constants.ObjectTypePlanLimit, Key: "sandboxes_maximum", Value: new(1)},
		{Object: constants.ObjectTypePlanLimit, Key: "seats_maximum", Value: new(1)},
		{Object: constants.ObjectTypePlanLimit, Key: "invoices_maximum", Value: new(100)},
		{Object: constants.ObjectTypePlanLimit, Key: "batches_maximum", Value: new(10000)},
	}, PageInfo{}),
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
	Limits: NewList([]PlanLimit{
		{Object: constants.ObjectTypePlanLimit, Key: "sandboxes_maximum", Value: new(3)},
		{Object: constants.ObjectTypePlanLimit, Key: "seats_maximum", Value: new(5)},
		{Object: constants.ObjectTypePlanLimit, Key: "invoices_maximum", Value: new(10000)},
		{Object: constants.ObjectTypePlanLimit, Key: "batches_maximum", Value: new(10000)},
	}, PageInfo{}),
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
	Limits: NewList([]PlanLimit{
		{Object: constants.ObjectTypePlanLimit, Key: "sandboxes_maximum", Value: new(999)},
		{Object: constants.ObjectTypePlanLimit, Key: "seats_maximum", Value: new(999)},
		{Object: constants.ObjectTypePlanLimit, Key: "invoices_maximum", Value: new(999)},
		{Object: constants.ObjectTypePlanLimit, Key: "batches_maximum", Value: new(10000)},
	}, PageInfo{}),
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
