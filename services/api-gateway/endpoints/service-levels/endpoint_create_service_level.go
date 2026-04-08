package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CreateServiceLevelRequest is the request to create a new service level.
type CreateServiceLevelRequest struct {
	// The ID of the carrier.
	CarrierID string `path:"carrier_id" validate:"required"`
	// The display name of the service level.
	Name string `json:"name" validate:"required,max=255"`
	// The service level code.
	Code string `json:"code" validate:"required,max=255"`
	// Whether this service level is visible in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" default:"visible" nullable:"false"`
	// Whether this is a default (system-synced) service level.
	IsDefault bool `json:"is_default"`
}

var sampleCreateServiceLevelRequest = &CreateServiceLevelRequest{
	Name:      "Ground Shipping",
	Code:      "ground",
	IsDefault: false,
}

func (*CreateServiceLevelRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateServiceLevelRequest)
}

type CreateServiceLevelEndpoint struct{}

func (e *CreateServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateServiceLevelRequest, *apiresource.ServiceLevel] {
	return &apiendpoint.APIEndpoint[*CreateServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Create Service Level",
		Description:       "Creates a new service level (shipping service level) for a carrier.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels",
		Request:           &CreateServiceLevelRequest{},
		Response:          &apiresource.ServiceLevel{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).CreateServiceLevel
		},
	}
}
