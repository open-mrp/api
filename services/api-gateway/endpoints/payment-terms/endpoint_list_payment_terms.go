package paymenttermep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list payment terms.
type ListPaymentTermsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return payment terms with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
}

// Returns a list of payment terms.
//
// Includes the terms your account created and the system defaults every account shares. Terms are sorted by creation date, newest first.
type ListPaymentTermsEndpoint struct{}

func (e *ListPaymentTermsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPaymentTermsRequest, *apiresource.List[apiresource.PaymentTerm]] {
	return (&apiendpoint.APIEndpoint[*ListPaymentTermsRequest, *apiresource.List[apiresource.PaymentTerm]]{
		Title:             "List Payment Terms",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/payment-terms",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPaymentTerms, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypePaymentTerm,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPaymentTermsRequest) (*apiresource.List[apiresource.PaymentTerm], *apierror.APIError) {
			return svc.(PaymentTermSvc).ListPaymentTerms
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
