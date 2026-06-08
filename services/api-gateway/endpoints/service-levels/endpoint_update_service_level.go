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

// Request to update a service level.
type UpdateServiceLevelRequest struct {
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Service level code.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
	// Whether this service level will be available for customers to select in the customer portal.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero"`
	// Default service levels are the default-selected service level for that carrier.
	IsDefault field.Optional[bool] `json:"is_default,omitzero"`
}

var sampleUpdateServiceLevelName = "Express Shipping"
var sampleUpdateServiceLevelRequest = &UpdateServiceLevelRequest{
	Name: field.Some(sampleUpdateServiceLevelName),
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
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).UpdateServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
