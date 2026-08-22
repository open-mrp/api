package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a carrier.
type UpdateCarrierRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
	// Human-readable name for the carrier.
	//
	// Must not match another carrier already visible to your account, including the system-provided ones.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Whether customers can see and select this carrier at checkout in the customer portal.
	//
	// Each of the carrier's service levels carries its own customer portal visibility, which this does not change.
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

// Updates a carrier's name and customer portal visibility.
//
// Only these two attributes can change: a carrier's code and account number are fixed at creation, and system-owned carriers cannot be updated at all.
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
