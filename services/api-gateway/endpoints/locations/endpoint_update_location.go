package locationep

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

// Request to partially update a location.
type UpdateLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
	// Display name of the location.
	//
	// Maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// This location's level in the storage hierarchy.
	//
	// The levels run from largest to smallest: `building`, `section`, `aisle`, `rack`, `shelf`, `bin`. They are descriptive labels rather than a rule — the parent is not required to be the next level up.
	TypeCode field.Optional[constants.LocationTypeCode] `json:"type,omitzero"`
	// The location this one sits under in the storage hierarchy.
	//
	// Must be an existing location in your account, and cannot be the location being updated. Send `null` to detach it from its parent and make it a top-level location.
	ParentID field.Clearable[string] `json:"parent_id,omitzero" validate:"omitempty"`
	// The locations that sit directly beneath this one.
	//
	// This replaces the full set of children: current children that are not listed are detached and become top-level locations, and listed locations are reparented onto this location. Send `null` to detach every child. Omit the field to leave the existing children untouched.
	ChildIDs field.Clearable[[]string] `json:"child_ids,omitzero"`
}

var sampleUpdateLocationRequest = &UpdateLocationRequest{
	Name:     field.Some("Warehouse B"),
	TypeCode: field.Some(constants.LocationTypeCodeSection),
	ParentID: field.Set(apiresource.SampleLocationID),
	ChildIDs: field.Set([]string{apiresource.SampleLocationID}),
}

func (*UpdateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateLocationRequest)
}

// Partially updates a location.
type UpdateLocationEndpoint struct{}

func (e *UpdateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location]{
		Title:               "Update Location",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/operations/locations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).UpdateLocation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
