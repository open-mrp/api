package carrierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a carrier.
type DeleteCarrierRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

// Deletes a carrier and its service levels.
//
// Only carriers your account created can be deleted, and only while no customer, order or shipment uses them. Archive a carrier that is in use by setting `status` to `archived`. A Shippo-connected carrier's Shippo account is deactivated.
type DeleteCarrierEndpoint struct{}

func (e *DeleteCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCarrierRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteCarrierRequest, *object.Deleted]{
		Title:             "Delete Carrier",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/carriers/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeCarrier,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCarrierRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(CarrierSvc).DeleteCarrier, constants.ObjectTypeCarrier, "id")
		},
	})
}
