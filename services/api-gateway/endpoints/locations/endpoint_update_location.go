package locationep

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

// Request to partially update a location.
type UpdateLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Location type code.
	TypeCode field.Optional[constants.LocationTypeCode] `json:"type,omitzero"`
	// Parent location ID. Send null to clear.
	ParentID field.Clearable[string] `json:"parent_id,omitzero" validate:"omitempty"`
	// Child location IDs. Replaces all current children when provided. Send null to clear.
	ChildIDs field.Clearable[[]string] `json:"child_ids,omitzero"`
}

var sampleUpdateLocationRequest = &UpdateLocationRequest{
	Name: field.Some("Warehouse B"),
}

func (*UpdateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateLocationRequest)
}

// Partially updates a location.
type UpdateLocationEndpoint struct{}

func (e *UpdateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location]{
		Title:             "Update Location",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
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
