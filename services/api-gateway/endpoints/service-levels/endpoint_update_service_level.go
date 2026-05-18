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

// Request to update a service level.
type UpdateServiceLevelRequest struct {
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Service level code.
	Code *string `json:"code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Whether this service level will be available for customers to select in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" nullable:"false"`
	// Default service levels are the default-selected service level for that carrier.
	IsDefault *bool `json:"is_default,omitempty" nullable:"false"`
}

var sampleUpdateServiceLevelName = "Express Shipping"
var sampleUpdateServiceLevelRequest = &UpdateServiceLevelRequest{
	Name: &sampleUpdateServiceLevelName,
}

func (*UpdateServiceLevelRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateServiceLevelRequest)
}

// Partially updates a service level.
type UpdateServiceLevelEndpoint struct{}

func (e *UpdateServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel] {
	return (&apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Update Service Level",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		Request:           &UpdateServiceLevelRequest{},
		Response:          &apiresource.ServiceLevel{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).UpdateServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	}).WithDocSource(e)
}
