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

// Request to create a location.
type CreateLocationRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Location type code.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// Parent location ID. Null for top-level locations.
	ParentID *string `json:"parent_id,omitempty" validate:"omitempty"`
	// IDs of child locations to attach.
	ChildIDs *[]string `json:"child_ids,omitempty"`
}

var sampleCreateLocationRequest = &CreateLocationRequest{
	Name:     "Warehouse A",
	TypeCode: "building",
}

func (*CreateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateLocationRequest)
}

// Creates a location for the caller's account.
type CreateLocationEndpoint struct{}

func (e *CreateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location]{
		Title:             "Create Location",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations",
		Request:           &CreateLocationRequest{},
		Response:          &apiresource.Location{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
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
	}).WithDocSource(e)
}
