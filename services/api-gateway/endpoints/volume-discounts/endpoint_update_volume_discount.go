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

// Volume discount tier to upsert.
type UpdateVolumeDiscountTierInput struct {
	// Existing tier ID. Omit for new tiers.
	ID field.Optional[string] `json:"id,omitzero" validate:"omitempty"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Discount percentage as a decimal string.
	DiscountPercentage field.Optional[string] `json:"discount_percentage,omitzero" format:"decimal"`
	// Quantity threshold as a decimal string.
	Threshold field.Optional[string] `json:"threshold,omitzero" format:"decimal"`
	// Parent tier ID for tier chaining.
	ParentTierID field.Optional[string] `json:"parent_tier_id,omitzero" validate:"omitempty"`
}

// Request to partially update a volume discount.
type UpdateVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Tiers (upsert semantics).
	Tiers []UpdateVolumeDiscountTierInput `json:"tiers,omitzero"`
	// Account group IDs to set as customer groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Product line IDs to set.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// Item category IDs to set.
	CategoryIDs []string `json:"category_ids,omitzero"`
	// Attribute IDs to set.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
	// Unit IDs to set as acceptable units.
	UnitIDs []string `json:"unit_ids,omitzero"`
	// Whether to replace tiers.
	HasTiers bool `json:"has_tiers,omitzero"`
	// Whether to replace customer groups.
	HasCustomerGroups bool `json:"has_customer_groups,omitzero"`
	// Whether to replace product lines.
	HasProductLines bool `json:"has_product_lines,omitzero"`
	// Whether to replace categories.
	HasCategories bool `json:"has_categories,omitzero"`
	// Whether to replace attributes.
	HasAttributes bool `json:"has_attributes,omitzero"`
	// Whether to replace units.
	HasUnits bool `json:"has_units,omitzero"`
}

var sampleUpdateVolumeDiscountRequest = &UpdateVolumeDiscountRequest{
	Name:     field.Some("Updated Bulk Discount"),
	HasTiers: true,
	Tiers: []UpdateVolumeDiscountTierInput{
		{
			Name:               field.Some("50+ Units"),
			DiscountPercentage: field.Some("10.000000000000000000000000000000"),
			Threshold:          field.Some("50.000000000000000000000000000000"),
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
		ObjectType:        constants.ObjectTypeVolumeDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).UpdateVolumeDiscount
		},
	})
}
