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
	ID *string `json:"id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Discount percentage as a decimal string.
	DiscountPercentage *string `json:"discount_percentage,omitempty" nullable:"false" format:"decimal"`
	// Quantity threshold as a decimal string.
	Threshold *string `json:"threshold,omitempty" nullable:"false" format:"decimal"`
	// Parent tier ID for tier chaining.
	ParentTierID *string `json:"parent_tier_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
}

// Request to partially update a volume discount.
type UpdateVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
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
	Name:     ptrString("Updated Bulk Discount"),
	HasTiers: true,
	Tiers: []UpdateVolumeDiscountTierInput{
		{
			Name:               ptrString("50+ Units"),
			DiscountPercentage: ptrString("10.000000000000000000000000000000"),
			Threshold:          ptrString("50.000000000000000000000000000000"),
		},
	},
}

func ptrString(s string) *string { return &s }

func (*UpdateVolumeDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateVolumeDiscountRequest)
}

type UpdateVolumeDiscountEndpoint struct{}

func (e *UpdateVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return &apiendpoint.APIEndpoint[*UpdateVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Update Volume Discount",
		Description:       "Partially updates a volume discount. Tiers use upsert semantics and relations are replaced when the corresponding has_* flag is true.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		Request:           &UpdateVolumeDiscountRequest{},
		Response:          &apiresource.VolumeDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).UpdateVolumeDiscount
		},
	}
}
