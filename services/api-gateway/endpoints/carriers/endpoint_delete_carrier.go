package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a carrier.
type DeleteCarrierRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

// Deletes a carrier and all of its service levels.
//
// If the carrier is connected through Shippo, its Shippo carrier account is deactivated. System-owned carriers cannot be deleted.
type DeleteCarrierEndpoint struct{}

func (e *DeleteCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCarrierRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteCarrierRequest, *apiresource.EmptyResource]{
		Title:               "Delete Carrier",
		Method:              http.MethodDelete,
		Route:               "/v1/operations/carriers/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCarrierRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CarrierSvc).DeleteCarrier
		},
	})
}
