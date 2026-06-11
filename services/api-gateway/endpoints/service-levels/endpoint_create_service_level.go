package servicelevelep

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

// Request to create a service level.
type CreateServiceLevelRequest struct {
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Human-readable name for the service level, shown to customers at checkout when the service level is visible.
	Name string `json:"name" validate:"required,max=255"`
	// Carrier-specific code identifying this service level (e.g. `fedex_ground`).
	//
	// Must be unique among the carrier's service levels.
	Code string `json:"code" validate:"required,max=255"`
	// Service level visibility in the customer portal.
	//
	// A `visible` service level can be selected by your customers at checkout; a `hidden` one is not offered there. New service levels are visible unless set to `hidden`.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero" default:"visible"`
	// Whether this becomes the carrier's default service level, pre-selected when the carrier is chosen.
	//
	// Each carrier has at most one default; setting this to `true` clears the carrier's existing default.
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

// Creates a service level for a carrier.
type CreateServiceLevelEndpoint struct{}

func (e *CreateServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateServiceLevelRequest, *apiresource.ServiceLevel] {
	return (&apiendpoint.APIEndpoint[*CreateServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Create Service Level",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).CreateServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
