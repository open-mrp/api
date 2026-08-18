package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Volume discount tier to create.
type CreateVolumeDiscountTierInput struct {
	// Display name of the tier.
	Name string `json:"name" validate:"required,max=255"`
	// Fraction of the price taken off once the threshold is met, as a decimal string.
	//
	// This is a multiplier, not a whole percent: `0.05` takes 5% off. When an order meets several tiers of the same discount, their reductions compound.
	DiscountPercentage string `json:"discount_percentage" validate:"required" format:"decimal"`
	// Minimum ordered quantity at which this tier's discount begins to apply, as a decimal string.
	//
	// The quantity compared against the threshold is the total across every line on the order that falls within the discount's scope, converted into one of the discount's units.
	Threshold string `json:"threshold" validate:"required" format:"decimal"`
	// ID of another tier that this tier follows.
	//
	// Tier IDs are assigned when the discount is created, so a tier created in this same request cannot be referenced here. The link is stored with the tier but does not affect pricing: every tier whose threshold is met applies, regardless of any parent.
	ParentTierID field.Optional[string] `json:"parent_tier_id,omitzero" validate:"omitempty"`
}

// Request to create a volume discount.
type CreateVolumeDiscountRequest struct {
	// Display name of the volume discount.
	//
	// Must be unique within the account.
	Name string `json:"name" validate:"required,max=255"`
	// Tiers for this volume discount.
	Tiers []CreateVolumeDiscountTierInput `json:"tiers" validate:"required,dive"`
	// Account group IDs to scope the discount to specific customer groups.
	//
	// When empty, all customers qualify. A discount scoped to a group the buyer belongs to is preferred over an unscoped one when both could apply to the same order line.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Product line IDs to scope the discount to.
	//
	// When empty, all product lines qualify.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// Item category IDs to scope the discount to.
	//
	// When empty, all categories qualify.
	CategoryIDs []string `json:"category_ids,omitzero"`
	// Attribute IDs to scope the discount to.
	//
	// When set, an item qualifies only if it has every listed attribute.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
	// IDs of the units that ordered quantities are measured in when evaluating tier thresholds.
	//
	// Quantities ordered in other units are converted into one of these before being compared against a threshold. Leaving this empty makes the discount inert: the quantity always evaluates to zero, so no threshold above zero is ever reached.
	UnitIDs []string `json:"unit_ids,omitzero"`
}

var sampleCreateVolumeDiscountRequest = &CreateVolumeDiscountRequest{
	Name: "Bulk Order Discount",
	Tiers: []CreateVolumeDiscountTierInput{
		{
			Name:               "100+ Units",
			DiscountPercentage: "5.000000000000000000000000000000",
			Threshold:          "100.000000000000000000000000000000",
		},
	},
}

func (*CreateVolumeDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateVolumeDiscountRequest)
}

// Creates a volume discount with its tiers and scoping associations.
//
// The discount name must be unique within the account; creating a discount with an existing name returns a conflict error.
//
// Each scoping list narrows the order lines the discount applies to, and an empty list places no restriction on that dimension. Because tier thresholds are compared against quantities converted into `unit_ids`, a discount created without any units never reaches a threshold above zero.
type CreateVolumeDiscountEndpoint struct{}

func (e *CreateVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return (&apiendpoint.APIEndpoint[*CreateVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Create Volume Discount",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/volume-discounts",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionCreate},
		},
		ObjectType: constants.ObjectTypeVolumeDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).CreateVolumeDiscount
		},
		LocationFunc: func(resp *apiresource.VolumeDiscount) string {
			return "/v1/sales/volume-discounts/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeVolumeDiscount,
			Fields:     []string{"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"},
		}),
	})
}
