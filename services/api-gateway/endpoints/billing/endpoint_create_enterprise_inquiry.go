package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Submits an enterprise plan inquiry to the sales team.
type CreateEnterpriseInquiryEndpoint struct{}

func (e *CreateEnterpriseInquiryEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnterpriseInquiry] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnterpriseInquiry]{
		Title:             "Create Enterprise Inquiry",
		Method:            http.MethodPost,
		Route:             "/v1/billing/actions/request-enterprise",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnterpriseInquiry, *apierror.APIError) {
			return svc.(BillingSvc).CreateEnterpriseInquiry
		},
	})
}
