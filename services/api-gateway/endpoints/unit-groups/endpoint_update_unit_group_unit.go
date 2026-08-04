package unitgroupep

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

// Request to partially update an associated unit within a unit group.
type UpdateUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// ID of the unit's association with the group, not the ID of the unit itself.
	AssociatedUnitID string `path:"id" validate:"required"`
	// ID of the unit this association refers to.
	//
	// Sending a different unit does not repoint the association; remove the association and add a new one instead. A unit sent here must still match the group's `type`.
	UnitID field.Optional[string] `json:"unit_id,omitzero" validate:"omitempty"`
	// Share of the unit's price removed when an order is placed in this unit.
	//
	// Expressed as a decimal fraction rather than a whole number, so `0.1` is a 10% discount and `0` is no discount.
	DiscountPercentage field.Optional[float64] `json:"discount_percentage,omitzero"`
	// Flat amount subtracted from the unit's price when an order is placed in this unit.
	//
	// Subtracted before `discount_percentage` is applied.
	DiscountFixed field.Optional[float64] `json:"discount_fixed,omitzero"`
	// Whether the unit is shown to customers in the customer portal.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero"`
}

var sampleUpdateUnitGroupUnitDiscountPct = float64(0.9)
var sampleUpdateUnitGroupUnitDiscountFixed = float64(2.5)
var sampleUpdateUnitGroupUnitUnitID = apiresource.SampleUnitID
var sampleUpdateUnitGroupUnitVisibility = constants.CustomerPortalVisibilityVisible
var sampleUpdateUnitGroupUnitRequest = &UpdateUnitGroupUnitRequest{
	UnitID:                   field.Some(sampleUpdateUnitGroupUnitUnitID),
	DiscountPercentage:       field.Some(sampleUpdateUnitGroupUnitDiscountPct),
	DiscountFixed:            field.Some(sampleUpdateUnitGroupUnitDiscountFixed),
	CustomerPortalVisibility: field.Some(sampleUpdateUnitGroupUnitVisibility),
}

func (*UpdateUnitGroupUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitGroupUnitRequest)
}

// Partially updates a unit's association with a unit group, changing the discount or customer portal visibility applied when ordering in that unit.
//
// Associations within system unit groups cannot be modified.
type UpdateUnitGroupUnitEndpoint struct{}

func (e *UpdateUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return (&apiendpoint.APIEndpoint[*UpdateUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:               "Update Unit Group Associated Unit",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/unit-groups/{unit_group_id}/units/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnitGroupUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).UpdateUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	})
}
