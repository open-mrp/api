package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a carrier.
type UpdateCarrierRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
	// Human-readable name for the carrier, unique among your account's carriers.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Carrier visibility in the customer portal.
	//
	// A `visible` carrier can be selected by your customers at checkout; a `hidden` carrier is not offered there.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero"`
}

var sampleUpdateCarrierName = "FedEx Express"
var sampleUpdateCarrierRequest = &UpdateCarrierRequest{
	Name:                     field.Some(sampleUpdateCarrierName),
	CustomerPortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
}

func (*UpdateCarrierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCarrierRequest)
}

// Partially updates a carrier's name and portal visibility.
type UpdateCarrierEndpoint struct{}

func (e *UpdateCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCarrierRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*UpdateCarrierRequest, *apiresource.Carrier]{
		Title:               "Update Carrier",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/operations/carriers/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).UpdateCarrier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "owner.account", "service_levels"},
		}),
	})
}
