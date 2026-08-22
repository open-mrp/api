package paymenttermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a payment term.
type RetrievePaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
}

// Returns a payment term by ID.
//
// Both payment terms created by your account and OpenMRP-provided system defaults can be retrieved.
type RetrievePaymentTermEndpoint struct{}

func (e *RetrievePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePaymentTermRequest, *apiresource.PaymentTerm] {
	return (&apiendpoint.APIEndpoint[*RetrievePaymentTermRequest, *apiresource.PaymentTerm]{
		Title:               "Retrieve Payment Term",
		Method:              http.MethodGet,
		Route:               "/v1/finance/payment-terms/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainPaymentTerms, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypePaymentTerm,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).GetPaymentTerm
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
