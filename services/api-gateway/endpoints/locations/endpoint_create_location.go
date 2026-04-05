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

// CreateLocationRequest is the request to create a new location.
type CreateLocationRequest struct {
	// The display name of the location.
	Name string `json:"name" validate:"required"`
	// The code of the location type.
	TypeCode constants.LocationTypeCode `json:"type_code" validate:"required"`
	// The ID of the parent location. Null for top-level locations.
	ParentID *string `json:"parent_id,omitempty"`
	// IDs of existing locations to attach as children of this location.
	ChildIDs []string `json:"child_ids,omitempty"`
}

var sampleCreateLocationRequest = &CreateLocationRequest{
	Name:     "Warehouse A",
	TypeCode: "building",
}

func (*CreateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateLocationRequest)
}

type CreateLocationEndpoint struct{}

func (e *CreateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location] {
	return &apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location]{
		Title:             "Create Location",
		Description:       "Creates a new location for the caller's account.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/locations",
		Request:           &CreateLocationRequest{},
		Response:          &apiresource.Location{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).CreateLocation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
	}
}
