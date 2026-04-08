package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateCarrierRequest is the request to update a carrier.
type UpdateCarrierRequest struct {
	// The ID of the carrier to update.
	CarrierID string `path:"id" validate:"required"`
	// The new display name for the carrier.
	Name *string `json:"name" validate:"omitempty,max=255"`
	// Whether this carrier is visible in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility"`
}

var sampleUpdateCarrierName = "FedEx Express"
var sampleUpdateCarrierRequest = &UpdateCarrierRequest{
	Name: &sampleUpdateCarrierName,
}

func (*UpdateCarrierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCarrierRequest)
}

type UpdateCarrierEndpoint struct{}

func (e *UpdateCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCarrierRequest, *apiresource.Carrier] {
	return &apiendpoint.APIEndpoint[*UpdateCarrierRequest, *apiresource.Carrier]{
		Title:             "Update Carrier",
		Description:       "Partially updates a carrier's name and portal visibility.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/carriers/{id}",
		Request:           &UpdateCarrierRequest{},
		Response:          &apiresource.Carrier{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).UpdateCarrier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "service_levels"},
		}),
	}
}
