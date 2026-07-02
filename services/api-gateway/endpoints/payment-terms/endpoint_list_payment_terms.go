package paymenttermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list payment terms.
type ListPaymentTermsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of payment terms.
//
// The list includes both payment terms created by your account and Augno-provided system defaults.
type ListPaymentTermsEndpoint struct{}

func (e *ListPaymentTermsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPaymentTermsRequest, *apiresource.List[apiresource.PaymentTerm]] {
	return (&apiendpoint.APIEndpoint[*ListPaymentTermsRequest, *apiresource.List[apiresource.PaymentTerm]]{
		Title:               "List Payment Terms",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/finance/payment-terms",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainPaymentTerms, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypePaymentTerm,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPaymentTermsRequest) (*apiresource.List[apiresource.PaymentTerm], *apierror.APIError) {
			return svc.(PaymentTermSvc).ListPaymentTerms
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
