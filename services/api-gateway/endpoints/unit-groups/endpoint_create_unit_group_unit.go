package unitgroupep

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

// Request to add a unit to a unit group.
type CreateUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// ID of the unit to associate with the group.
	//
	// The unit's dimension must match the group's `type`.
	UnitID string `json:"unit_id" validate:"required"`
	// Share of the unit's price removed when an order is placed in this unit.
	//
	// Expressed as a decimal fraction rather than a whole number, so `0.1` is a 10% discount. Send `0` explicitly for no discount — omitting the field stores a discount of `1`, which removes the entire price.
	DiscountPercentage field.Optional[float64] `json:"discount_percentage,omitzero" default:"1"`
	// Flat amount subtracted from the unit's price when an order is placed in this unit.
	//
	// Subtracted before `discount_percentage` is applied.
	DiscountFixed field.Optional[float64] `json:"discount_fixed,omitzero" default:"0"`
	// Whether the unit is shown to customers in the customer portal.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero" default:"visible"`
}

var sampleCreateUnitGroupUnitDiscountPct = float64(1)
var sampleCreateUnitGroupUnitDiscountFixed = float64(0)
var sampleCreateUnitGroupUnitVisibility = constants.CustomerPortalVisibilityVisible
var sampleCreateUnitGroupUnitRequest = &CreateUnitGroupUnitRequest{
	UnitID:                   apiresource.SampleUnitID,
	DiscountPercentage:       field.Some(sampleCreateUnitGroupUnitDiscountPct),
	DiscountFixed:            field.Some(sampleCreateUnitGroupUnitDiscountFixed),
	CustomerPortalVisibility: field.Some(sampleCreateUnitGroupUnitVisibility),
}

func (*CreateUnitGroupUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitGroupUnitRequest)
}

// Adds a unit to a unit group so that products using the group can be ordered in it.
//
// A unit can appear in a group only once, so use the update endpoint to change the discount or visibility of a unit that is already associated. Units cannot be added to system unit groups.
type CreateUnitGroupUnitEndpoint struct{}

func (e *CreateUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return (&apiendpoint.APIEndpoint[*CreateUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Create Unit Group Associated Unit",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		// Adding/updating a unit within a group is a sub-resource mutation of the
		// parent group, so the downstream UpsertUnitGroupUnit checks unit_groups:update.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnitGroupUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).CreateUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	})
}
