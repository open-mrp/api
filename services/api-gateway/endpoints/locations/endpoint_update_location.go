package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateLocationRequest is the request to partially update a location.
type UpdateLocationRequest struct {
	// The ID of the location to update.
	LocationID string `path:"id" validate:"required"`
	// The display name of the location.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The code of the location type.
	TypeCode *constants.LocationTypeCode `json:"type,omitempty" nullable:"false"`
	// The ID of the parent location. Send null to clear.
	ParentID *string `json:"parent_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The IDs of child locations. When provided, replaces all current children.
	ChildIDs *[]string `json:"child_ids,omitempty" nullable:"false"`
}

var sampleUpdateName = "Warehouse B"

var sampleUpdateLocationRequest = &UpdateLocationRequest{
	Name: &sampleUpdateName,
}

func (*UpdateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateLocationRequest)
}

type UpdateLocationEndpoint struct{}

func (e *UpdateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location] {
	return &apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location]{
		Title:             "Update Location",
		Description:       "Partially updates a location.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/{id}",
		Request:           &UpdateLocationRequest{},
		Response:          &apiresource.Location{},
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
	}
}
