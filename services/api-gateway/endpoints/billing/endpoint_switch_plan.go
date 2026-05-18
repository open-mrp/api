package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to switch pricing plans.
type SwitchPlanRequest struct {
	// Target pricing plan ID.
	PlanID string `path:"id" validate:"required"`
}

// Switches the account to a different pricing plan, handling free-to-paid, paid-to-free, and paid-to-paid scenarios.
type SwitchPlanEndpoint struct{}

func (e *SwitchPlanEndpoint) Materialize() *apiendpoint.APIEndpoint[*SwitchPlanRequest, *apiresource.SwitchPlanResponse] {
	return (&apiendpoint.APIEndpoint[*SwitchPlanRequest, *apiresource.SwitchPlanResponse]{
		Title:             "Switch Plan",
		Method:            http.MethodPost,
		Route:             "/v1/billing/plans/{id}/switch",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SwitchPlanRequest) (*apiresource.SwitchPlanResponse, *apierror.APIError) {
			return svc.(BillingSvc).SwitchPlan
		},
	})
}
