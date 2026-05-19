package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Volume discount tier to upsert.
type UpdateVolumeDiscountTierInput struct {
	// Existing tier ID. Omit for new tiers.
	ID *string `json:"id,omitempty" validate:"omitempty"`
	// Display name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Discount percentage as a decimal string.
	DiscountPercentage *string `json:"discount_percentage,omitempty" format:"decimal"`
	// Quantity threshold as a decimal string.
	Threshold *string `json:"threshold,omitempty" format:"decimal"`
	// Parent tier ID for tier chaining.
	ParentTierID *string `json:"parent_tier_id,omitempty" validate:"omitempty"`
}

// Request to partially update a volume discount.
type UpdateVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Tiers (upsert semantics).
	Tiers []UpdateVolumeDiscountTierInput `json:"tiers,omitempty"`
	// Account group IDs to set as customer groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Product line IDs to set.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Item category IDs to set.
	CategoryIDs []string `json:"category_ids,omitempty"`
	// Attribute IDs to set.
	AttributeIDs []string `json:"attribute_ids,omitempty"`
	// Unit IDs to set as acceptable units.
	UnitIDs []string `json:"unit_ids,omitempty"`
	// Whether to replace tiers.
	HasTiers bool `json:"has_tiers,omitempty"`
	// Whether to replace customer groups.
	HasCustomerGroups bool `json:"has_customer_groups,omitempty"`
	// Whether to replace product lines.
	HasProductLines bool `json:"has_product_lines,omitempty"`
	// Whether to replace categories.
	HasCategories bool `json:"has_categories,omitempty"`
	// Whether to replace attributes.
	HasAttributes bool `json:"has_attributes,omitempty"`
	// Whether to replace units.
	HasUnits bool `json:"has_units,omitempty"`
}

var sampleUpdateVolumeDiscountRequest = &UpdateVolumeDiscountRequest{
	Name:     new("Updated Bulk Discount"),
	HasTiers: true,
	Tiers: []UpdateVolumeDiscountTierInput{
		{
			Name:               new("50+ Units"),
			DiscountPercentage: new("10.000000000000000000000000000000"),
			Threshold:          new("50.000000000000000000000000000000"),
		},
	},
}

func (*UpdateVolumeDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateVolumeDiscountRequest)
}

// Partially updates a volume discount. Tiers use upsert semantics and relations are replaced when the corresponding has_* flag is true.
type UpdateVolumeDiscountEndpoint struct{}

func (e *UpdateVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return (&apiendpoint.APIEndpoint[*UpdateVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Update Volume Discount",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).UpdateVolumeDiscount
		},
	})
}
