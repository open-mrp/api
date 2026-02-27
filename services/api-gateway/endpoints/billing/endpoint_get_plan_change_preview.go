package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const getPlanChangePreviewDescription string = `Previews the cost impact of switching to a different pricing plan.
Returns proration details including credits, charges, and net amount.`

type GetPlanProrationRequest struct {
	PlanID string `path:"id"`
}

type GetPlanChangePreviewEndpoint struct{}

func (e *GetPlanChangePreviewEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPlanProrationRequest, *apiresource.PlanChangeProration] {
	return &apiendpoint.APIEndpoint[*GetPlanProrationRequest, *apiresource.PlanChangeProration]{
		Title:             "Preview Plan Change",
		Description:       getPlanChangePreviewDescription,
		Method:            http.MethodGet,
		Route:             "/v1/billing/plans/{id}/proration",
		Request:           &GetPlanProrationRequest{},
		Response:          apiresource.SamplePlanChangeProration,
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPlanProrationRequest) (*apiresource.PlanChangeProration, *apierror.APIError) {
			return svc.(BillingSvc).GetPlanChangePreview
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
