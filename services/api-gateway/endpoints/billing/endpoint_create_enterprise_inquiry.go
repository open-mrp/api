package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const createEnterpriseInquiryDescription string = `Sends a notification to the sales team requesting information about enterprise plans.
Returns a confirmation that the inquiry was submitted.`

type CreateEnterpriseInquiryEndpoint struct{}

func (e *CreateEnterpriseInquiryEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnterpriseInquiry] {
	return &apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnterpriseInquiry]{
		Title:             "Create Enterprise Inquiry",
		Description:       createEnterpriseInquiryDescription,
		Method:            http.MethodPost,
		Route:             "/v1/billing/actions/request-enterprise",
		ContentType:       "application/json",
		Request:           &apiresource.EmptyResource{},
		Response:          apiresource.SampleEnterpriseInquiry,
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnterpriseInquiry, *apierror.APIError) {
			return svc.(BillingSvc).CreateEnterpriseInquiry
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
