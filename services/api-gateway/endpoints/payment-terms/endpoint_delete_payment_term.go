package paymenttermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a payment term.
type DeletePaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
}

// Deletes a payment term.
//
// Only payment terms created by your account can be deleted; system-owned default terms cannot be.
type DeletePaymentTermEndpoint struct{}

func (e *DeletePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePaymentTermRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeletePaymentTermRequest, *apiresource.EmptyResource]{
		Title:               "Delete Payment Term",
		Method:              http.MethodDelete,
		Route:               "/v1/finance/payment-terms/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainPaymentTerms, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePaymentTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PaymentTermSvc).DeletePaymentTerm
		},
	})
}
