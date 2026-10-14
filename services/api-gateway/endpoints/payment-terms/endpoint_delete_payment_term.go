package paymenttermep

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

// Request to delete a payment term.
type DeletePaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
}

// Deletes a payment term.
//
// Only payment terms created by your account can be deleted; system-owned default terms cannot be.
type DeletePaymentTermEndpoint struct{}

func (e *DeletePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePaymentTermRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePaymentTermRequest, *object.Deleted]{
		Title:             "Delete Payment Term",
		Method:            http.MethodDelete,
		Route:             "/v1/finance/payment-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypePaymentTerm,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPaymentTerms, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePaymentTermRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PaymentTermSvc).DeletePaymentTerm, constants.ObjectTypePaymentTerm, "id")
		},
	})
}
