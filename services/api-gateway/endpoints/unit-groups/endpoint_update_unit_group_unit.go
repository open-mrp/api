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
	// Unit group unit ID.
	AssociatedUnitID string `path:"id" validate:"required"`
	// ID of the unit this association refers to.
	//
	// The unit's dimension must match the group's `type`.
	UnitID field.Optional[string] `json:"unit_id,omitzero" validate:"omitempty"`
	// Percentage discount applied to the unit's price when an order is placed in this unit (e.g. `10` is a 10% discount).
	DiscountPercentage field.Optional[float64] `json:"discount_percentage,omitzero"`
	// Flat amount subtracted from the unit's price when an order is placed in this unit.
	DiscountFixed field.Optional[float64] `json:"discount_fixed,omitzero"`
	// Whether the unit is shown to customers in the customer portal.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero"`
}

var sampleUpdateUnitGroupUnitDiscountPct = float64(0.9)
var sampleUpdateUnitGroupUnitUnitID = apiresource.SampleUnitID
var sampleUpdateUnitGroupUnitRequest = &UpdateUnitGroupUnitRequest{
	UnitID:             field.Some(sampleUpdateUnitGroupUnitUnitID),
	DiscountPercentage: field.Some(sampleUpdateUnitGroupUnitDiscountPct),
}

func (*UpdateUnitGroupUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitGroupUnitRequest)
}

// Partially updates an associated unit within a unit group.
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
