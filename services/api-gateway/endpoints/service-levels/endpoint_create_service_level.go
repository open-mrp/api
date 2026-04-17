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

// Request to create a service level.
type CreateServiceLevelRequest struct {
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Service level code.
	Code string `json:"code" validate:"required,max=255"`
	// Customer portal visibility.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" default:"visible" nullable:"false"`
	// Default (system-synced) service level.
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
		Description:       "Creates a service level for a carrier.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels",
		Request:           &CreateServiceLevelRequest{},
		Response:          &apiresource.ServiceLevel{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).CreateServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
