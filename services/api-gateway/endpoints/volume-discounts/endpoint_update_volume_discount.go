package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Volume discount tier to upsert.
//
// Each entry is written as a whole: send every value you want the tier to keep, since values left out are not carried over from the existing tier.
type UpdateVolumeDiscountTierInput struct {
	// ID of an existing tier to update.
	//
	// Omit to create a new tier.
	ID field.Optional[string] `json:"id,omitzero" validate:"omitempty"`
	// Display name of the tier.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Fraction of the price taken off once the threshold is met, as a decimal string.
	//
	// This is a multiplier, not a whole percent: `0.05` takes 5% off. When an order meets several tiers of the same discount, their reductions compound.
	DiscountPercentage field.Optional[string] `json:"discount_percentage,omitzero" format:"decimal"`
	// Minimum ordered quantity at which this tier's discount begins to apply, as a decimal string.
	//
	// The quantity compared against the threshold is the total across every line on the order that falls within the discount's scope, converted into one of the discount's units.
	Threshold field.Optional[string] `json:"threshold,omitzero" format:"decimal"`
	// ID of another tier in this discount that this tier follows.
	//
	// The link is stored with the tier but does not affect pricing. Omitting it when updating an existing tier clears the link.
	ParentTierID field.Optional[string] `json:"parent_tier_id,omitzero" validate:"omitempty"`
}

// Request to partially update a volume discount.
type UpdateVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
	// Display name of the volume discount.
	//
	// Must be unique within the account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// The full set of tiers for this discount.
	//
	// Only applied when `has_tiers` is `true`. Tiers with an `id` are updated, tiers without an `id` are created, and existing tiers not present in the list are deleted.
	Tiers []UpdateVolumeDiscountTierInput `json:"tiers,omitzero" validate:"omitempty,dive"`
	// Account group IDs to set as customer groups.
	//
	// Only applied when `has_customer_groups` is `true`, in which case they replace the existing set entirely.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Product line IDs to set.
	//
	// Only applied when `has_product_lines` is `true`, in which case they replace the existing set entirely.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// Item category IDs to set.
	//
	// Only applied when `has_categories` is `true`, in which case they replace the existing set entirely.
	CategoryIDs []string `json:"category_ids,omitzero"`
	// Attribute IDs to set.
	//
	// Only applied when `has_attributes` is `true`, in which case they replace the existing set entirely.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
	// IDs of the units to set as acceptable units.
	//
	// Only applied when `has_units` is `true`, in which case they replace the existing set entirely. Clearing every unit makes the discount inert, since ordered quantity then always evaluates to zero.
	UnitIDs []string `json:"unit_ids,omitzero"`
	// Whether to apply the `tiers` field.
	//
	// When `true`, the discount's tiers are replaced with the contents of `tiers` (an empty list deletes all tiers). When `false`, `tiers` is ignored.
	HasTiers bool `json:"has_tiers,omitzero"`
	// Whether to apply the `customer_group_ids` field; when `false`, it is ignored.
	HasCustomerGroups bool `json:"has_customer_groups,omitzero"`
	// Whether to apply the `product_line_ids` field; when `false`, it is ignored.
	HasProductLines bool `json:"has_product_lines,omitzero"`
	// Whether to apply the `category_ids` field; when `false`, it is ignored.
	HasCategories bool `json:"has_categories,omitzero"`
	// Whether to apply the `attribute_ids` field; when `false`, it is ignored.
	HasAttributes bool `json:"has_attributes,omitzero"`
	// Whether to apply the `unit_ids` field; when `false`, it is ignored.
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

// Partially updates a volume discount.
//
// The tier and association lists are only applied when their corresponding `has_*` flag is `true`, in which case they replace the existing set entirely. Tiers use upsert semantics: tiers with an `id` are updated, tiers without one are created, and existing tiers omitted from the list are deleted.
//
// The name must remain unique within the account; reusing another discount's name returns a conflict error. Order lines that have already been priced keep the unit price they were given; the revised discount applies to lines priced after the change.
type UpdateVolumeDiscountEndpoint struct{}

func (e *UpdateVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return (&apiendpoint.APIEndpoint[*UpdateVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Update Volume Discount",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeVolumeDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).UpdateVolumeDiscount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeVolumeDiscount,
			Fields:     []string{"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"},
		}),
	})
}
