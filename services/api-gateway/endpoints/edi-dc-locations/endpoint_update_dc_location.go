package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a DC location.
type UpdateDCLocationRequest struct {
	// DC location ID.
	DCLocationID string `path:"id" validate:"required"`
	// Customer account ID.
	CustomerID *string `json:"customer_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// Location description.
	Location *string `json:"location,omitempty" nullable:"false" validate:"omitempty,max=255"`
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
		ContentType:       "application/json",
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
