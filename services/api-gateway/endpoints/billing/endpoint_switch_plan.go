package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to switch pricing plans.
type SwitchPlanRequest struct {
	// Target pricing plan ID.
	PlanID string `path:"id" validate:"required"`
}

// Switches the account to a different pricing plan.
//
// Handles free-to-paid, paid-to-free, and paid-to-paid changes. Switches that owe a prorated amount are charged immediately; use Preview Plan Change to see the cost first.
type SwitchPlanEndpoint struct{}

func (e *SwitchPlanEndpoint) Materialize() *apiendpoint.APIEndpoint[*SwitchPlanRequest, *apiresource.SwitchPlanResponse] {
	return (&apiendpoint.APIEndpoint[*SwitchPlanRequest, *apiresource.SwitchPlanResponse]{
		Title:             "Switch Plan",
		Method:            http.MethodPost,
		Route:             "/v1/billing/plans/{id}/switch",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSwitchPlanResponse,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SwitchPlanRequest) (*apiresource.SwitchPlanResponse, *apierror.APIError) {
			return svc.(BillingSvc).SwitchPlan
		},
	})
}
