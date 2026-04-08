package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateDCLocationRequest is the request to partially update a DC location.
type UpdateDCLocationRequest struct {
	// The ID of the DC location to update.
	DCLocationID string `path:"id" validate:"required"`
	// The ID of the customer account to associate with this DC location.
	CustomerID *string `json:"customer_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The location description.
	Location *string `json:"location,omitempty" validate:"omitempty,max=255"`
}

var sampleUpdateDCLocationLocation = "Warehouse B - Bay 1"
var sampleUpdateDCLocationRequest = &UpdateDCLocationRequest{
	Location: &sampleUpdateDCLocationLocation,
}

func (*UpdateDCLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDCLocationRequest)
}

type UpdateDCLocationEndpoint struct{}

func (e *UpdateDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDCLocationRequest, *apiresource.DCLocation] {
	return &apiendpoint.APIEndpoint[*UpdateDCLocationRequest, *apiresource.DCLocation]{
		Title:             "Update DC Location",
		Description:       "Partially updates a DC location.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/dc-locations/{id}",
		Request:           &UpdateDCLocationRequest{},
		Response:          &apiresource.DCLocation{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).UpdateDCLocation
		},
	}
}
