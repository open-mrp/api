package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Volume discount tier to create.
type CreateVolumeDiscountTierInput struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Percentage taken off the price once the threshold is met, as a decimal string (e.g. `5` for 5%).
	DiscountPercentage string `json:"discount_percentage" validate:"required" format:"decimal"`
	// Minimum ordered quantity at which this tier's discount begins to apply, as a decimal string.
	Threshold string `json:"threshold" validate:"required" format:"decimal"`
	// Parent tier ID for tier chaining.
	ParentTierID field.Optional[string] `json:"parent_tier_id,omitzero" validate:"omitempty"`
}

// Request to create a volume discount.
type CreateVolumeDiscountRequest struct {
	// Display name.
	//
	// Must be unique within the account.
	Name string `json:"name" validate:"required,max=255"`
	// Tiers for this volume discount.
	Tiers []CreateVolumeDiscountTierInput `json:"tiers" validate:"required"`
	// Account group IDs to scope the discount to specific customer groups.
	//
	// When empty, all customers qualify.
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
type CreateVolumeDiscountEndpoint struct{}

func (e *CreateVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return (&apiendpoint.APIEndpoint[*CreateVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Create Volume Discount",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/volume-discounts",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeVolumeDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).CreateVolumeDiscount
		},
		LocationFunc: func(resp *apiresource.VolumeDiscount) string {
			return "/v1/sales/volume-discounts/" + resp.ID
		},
	})
}
