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

// UpdateServiceLevelRequest is the request to update a service level.
type UpdateServiceLevelRequest struct {
	// The ID of the carrier.
	CarrierID string `path:"carrier_id" validate:"required"`
	// The ID of the service level to update.
	ServiceLevelID string `path:"id" validate:"required"`
	// The new display name for the service level.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The new service level code.
	Code *string `json:"code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Whether this service level is visible in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" nullable:"false"`
}

var sampleUpdateServiceLevelName = "Express Shipping"
var sampleUpdateServiceLevelRequest = &UpdateServiceLevelRequest{
	Name: &sampleUpdateServiceLevelName,
}

func (*UpdateServiceLevelRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateServiceLevelRequest)
}

type UpdateServiceLevelEndpoint struct{}

func (e *UpdateServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel] {
	return &apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Update Service Level",
		Description:       "Partially updates a service level's name, code, and portal visibility.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		Request:           &UpdateServiceLevelRequest{},
		Response:          &apiresource.ServiceLevel{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).UpdateServiceLevel
		},
	}
}
