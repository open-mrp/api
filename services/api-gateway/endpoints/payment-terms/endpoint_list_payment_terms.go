package paymenttermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListPaymentTermsRequest is the request to list payment terms with optional filters.
type ListPaymentTermsRequest struct {
	apiresource.PaginationRequest
}

type ListPaymentTermsEndpoint struct{}

func (e *ListPaymentTermsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPaymentTermsRequest, *apiresource.List[apiresource.PaymentTerm]] {
	return &apiendpoint.APIEndpoint[*ListPaymentTermsRequest, *apiresource.List[apiresource.PaymentTerm]]{
		Title:             "List Payment Terms",
		Description:       "Returns a paginated list of payment terms for the account, including both account-specific and default system payment terms.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/payment-terms",
		Request:           &ListPaymentTermsRequest{},
		Response:          &apiresource.List[apiresource.PaymentTerm]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPaymentTermsRequest) (*apiresource.List[apiresource.PaymentTerm], *apierror.APIError) {
			return svc.(PaymentTermSvc).ListPaymentTerms
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
