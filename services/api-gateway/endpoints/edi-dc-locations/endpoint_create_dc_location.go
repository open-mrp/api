package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a DC location.
type CreateDCLocationRequest struct {
	// Customer account ID.
	CustomerID string `json:"customer_id" validate:"required"`
	// Location description.
	Location string `json:"location" validate:"required,max=255"`
}

var sampleCreateDCLocationRequest = &CreateDCLocationRequest{
	CustomerID: apiresource.SampleCustomerID,
	Location:   "Warehouse A - Bay 3",
}

func (*CreateDCLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateDCLocationRequest)
}

// Creates a DC location.
type CreateDCLocationEndpoint struct{}

func (e *CreateDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateDCLocationRequest, *apiresource.DCLocation] {
	return (&apiendpoint.APIEndpoint[*CreateDCLocationRequest, *apiresource.DCLocation]{
		Title:             "Create DC Location",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).CreateDCLocation
		},
		LocationFunc: func(resp *apiresource.DCLocation) string {
			return "/v1/operations/dc-locations/" + resp.ID
		},
	})
}
