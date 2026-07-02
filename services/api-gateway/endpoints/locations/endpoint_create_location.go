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

// Request to create a location.
type CreateLocationRequest struct {
	// Display name of the location.
	//
	// Maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// Location type code, identifying this location's level in the storage hierarchy.
	//
	// - `building`: a building-level location.
	// - `section`: a section within a building.
	// - `aisle`: an aisle within a section.
	// - `rack`: a rack within an aisle.
	// - `shelf`: a shelf within a rack.
	// - `bin`: a bin within a shelf.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// ID of the parent location.
	//
	// Omit for top-level locations.
	ParentID field.Optional[string] `json:"parent_id,omitzero"`
	// IDs of existing locations to attach as children of the new location.
	//
	// Listed locations are moved from their current parent, if any.
	ChildIDs field.Optional[[]string] `json:"child_ids,omitzero"`
}

var sampleCreateLocationRequest = &CreateLocationRequest{
	Name:     "Warehouse A",
	TypeCode: "building",
	ParentID: field.Some(apiresource.SampleLocationID),
	ChildIDs: field.Some([]string{apiresource.SampleLocationID}),
}

func (*CreateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateLocationRequest)
}

// Creates a storage location, optionally placing it in the location hierarchy.
type CreateLocationEndpoint struct{}

func (e *CreateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location]{
		Title:               "Create Location",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/locations",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionCreate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).CreateLocation
		},
		LocationFunc: func(resp *apiresource.Location) string {
			return "/v1/operations/locations/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
