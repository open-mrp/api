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
	// Discount percentage as a decimal string.
	DiscountPercentage string `json:"discount_percentage" validate:"required" format:"decimal"`
	// Quantity threshold as a decimal string.
	Threshold string `json:"threshold" validate:"required" format:"decimal"`
	// Parent tier ID for tier chaining.
	ParentTierID field.Optional[string] `json:"parent_tier_id,omitzero" validate:"omitempty"`
}

// Request to create a volume discount.
type CreateVolumeDiscountRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Tiers for this volume discount.
	Tiers []CreateVolumeDiscountTierInput `json:"tiers" validate:"required"`
	// Account group IDs to associate as customer groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Product line IDs to associate.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// Item category IDs to associate.
	CategoryIDs []string `json:"category_ids,omitzero"`
	// Attribute IDs to associate.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
	// Unit IDs to associate as acceptable units.
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

// Creates a volume discount for the target account.
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
