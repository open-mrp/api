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
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of payment terms.
//
// The list includes both payment terms created by your account and OpenMRP-provided system defaults.
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
		Preview:           true,
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
