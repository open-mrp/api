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

// Request to partially update a unit group.
type UpdateUnitGroupRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"id" validate:"required"`
	// Display name of the unit group.
	//
	// Must be unique within the account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Free-form notes about the unit group.
	//
	// Set to `null` to clear.
	Notes field.Clearable[string] `json:"notes,omitzero"`
	// ID of the unit to designate as the group's reference unit.
	//
	// Must be a unit of the group's dimension, which cannot itself be changed.
	BaseUnitID field.Optional[string] `json:"base_unit_id,omitzero" validate:"omitempty"`
	// Units to add to the group.
	//
	// Only units that are not already in the group can be listed here; use the associated-unit update and delete endpoints to change or remove an existing association. Associations left out of the list are untouched.
	AssociatedUnits field.Optional[[]CreateUnitGroupUnitParam] `json:"associated_units,omitzero"`
}

var sampleUpdateUnitGroupName = "Weight Units (Updated)"
var sampleUpdateUnitGroupBaseUnitID = apiresource.SampleUnitID
var sampleUpdateUnitGroupNotes = "Added kilogram association for metric orders."
var sampleUpdateUnitGroupUnitDiscountPct2 = float64(1)
var sampleUpdateUnitGroupUnitDiscountFixed2 = float64(0)
var sampleUpdateUnitGroupUnitVisibility2 = constants.CustomerPortalVisibilityVisible
var sampleUpdateUnitGroupRequest = &UpdateUnitGroupRequest{
	Name:       field.Some(sampleUpdateUnitGroupName),
	Notes:      field.Set(sampleUpdateUnitGroupNotes),
	BaseUnitID: field.Some(sampleUpdateUnitGroupBaseUnitID),
	AssociatedUnits: field.Some([]CreateUnitGroupUnitParam{
		{
			UnitID:                   apiresource.SampleUnitID,
			DiscountPercentage:       field.Some(sampleUpdateUnitGroupUnitDiscountPct2),
			DiscountFixed:            field.Some(sampleUpdateUnitGroupUnitDiscountFixed2),
			CustomerPortalVisibility: field.Some(sampleUpdateUnitGroupUnitVisibility2),
		},
	}),
}

func (*UpdateUnitGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitGroupRequest)
}

// Partially updates a unit group.
//
// System unit groups cannot be modified, and a group's dimension is fixed once it is created.
type UpdateUnitGroupEndpoint struct{}

func (e *UpdateUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitGroupRequest, *apiresource.UnitGroup] {
	return (&apiendpoint.APIEndpoint[*UpdateUnitGroupRequest, *apiresource.UnitGroup]{
		Title:               "Update Unit Group",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/unit-groups/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnitGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
			return svc.(UnitGroupSvc).UpdateUnitGroup
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroup,
			Fields:     []string{"owner", "owner.account", "base_unit", "associated_units"},
		}),
	})
}
