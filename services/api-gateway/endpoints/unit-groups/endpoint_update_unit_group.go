package unitgroupep

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

// UpdateUnitGroupRequest is a request to partially update a unit group.
type UpdateUnitGroupRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Notes. Set to null to clear.
	Notes field.Clearable[string] `json:"notes,omitzero"`
	// Base unit ID.
	BaseUnitID field.Optional[string] `json:"base_unit_id,omitzero" validate:"omitempty"`
	// Upserts associated units when provided. Existing units not in the list are preserved.
	AssociatedUnits field.Optional[[]CreateUnitGroupUnitParam] `json:"associated_units,omitzero"`
}

var sampleUpdateUnitGroupName = "Weight Units (Updated)"
var sampleUpdateUnitGroupBaseUnitID = apiresource.SampleUnitID
var sampleUpdateUnitGroupRequest = &UpdateUnitGroupRequest{
	Name:       field.Some(sampleUpdateUnitGroupName),
	BaseUnitID: field.Some(sampleUpdateUnitGroupBaseUnitID),
}

func (*UpdateUnitGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitGroupRequest)
}

// Partially updates a unit group. System unit groups cannot be updated.
type UpdateUnitGroupEndpoint struct{}

func (e *UpdateUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitGroupRequest, *apiresource.UnitGroup] {
	return (&apiendpoint.APIEndpoint[*UpdateUnitGroupRequest, *apiresource.UnitGroup]{
		Title:             "Update Unit Group",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/unit-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUnitGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
			return svc.(UnitGroupSvc).UpdateUnitGroup
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroup,
			Fields:     []string{"owner", "owner.account", "base_unit", "associated_units"},
		}),
	})
}
