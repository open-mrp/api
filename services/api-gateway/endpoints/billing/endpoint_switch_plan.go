package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const switchPlanDescription string = `Initiates a plan switch. Handles free-to-paid (checkout redirect),
paid-to-free (subscription cancellation), and paid-to-paid (subscription update) scenarios.`

type SwitchPlanRequest struct {
	PlanID string `path:"id"`
}

type SwitchPlanEndpoint struct{}

func (e *SwitchPlanEndpoint) Materialize() *apiendpoint.APIEndpoint[*SwitchPlanRequest, *apiresource.SwitchPlanResponse] {
	return &apiendpoint.APIEndpoint[*SwitchPlanRequest, *apiresource.SwitchPlanResponse]{
		Title:             "Switch Plan",
		Description:       switchPlanDescription,
		Method:            http.MethodPost,
		Route:             "/v1/billing/plans/{id}/switch",
		ContentType:       "application/json",
		Request:           &SwitchPlanRequest{},
		Response:          apiresource.SampleSwitchPlanResponse,
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SwitchPlanRequest) (*apiresource.SwitchPlanResponse, *apierror.APIError) {
			return svc.(BillingSvc).SwitchPlan
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
