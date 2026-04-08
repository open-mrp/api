package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateDCLocationRequest is the request to create a new DC location.
type CreateDCLocationRequest struct {
	// The ID of the customer account to associate with this DC location.
	CustomerID string `json:"customer_id" validate:"required,max=191"`
	// The location description.
	Location string `json:"location" validate:"required,max=255"`
}

var sampleCreateDCLocationRequest = &CreateDCLocationRequest{
	CustomerID: apiresource.SampleCustomerID,
	Location:   "Warehouse A - Bay 3",
}

func (*CreateDCLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateDCLocationRequest)
}

type CreateDCLocationEndpoint struct{}

func (e *CreateDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateDCLocationRequest, *apiresource.DCLocation] {
	return &apiendpoint.APIEndpoint[*CreateDCLocationRequest, *apiresource.DCLocation]{
		Title:             "Create DC Location",
		Description:       "Creates a new DC location.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/dc-locations",
		Request:           &CreateDCLocationRequest{},
		Response:          &apiresource.DCLocation{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).CreateDCLocation
		},
	}
}
