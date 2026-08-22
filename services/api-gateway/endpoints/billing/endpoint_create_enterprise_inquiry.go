package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Asks the OpenMRP sales team to get in touch about an enterprise plan.
//
// The account, its current plan, and the requesting user's name and email are sent to sales for follow-up. Nothing about the account's plan, subscription, or billing changes.
type CreateEnterpriseInquiryEndpoint struct{}

func (e *CreateEnterpriseInquiryEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnterpriseInquiry] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnterpriseInquiry]{
		Title:             "Create Enterprise Inquiry",
		Method:            http.MethodPost,
		Route:             "/v1/billing/actions/request-enterprise",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeEnterpriseInquiry,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnterpriseInquiry, *apierror.APIError) {
			return svc.(BillingSvc).CreateEnterpriseInquiry
		},
	})
}
