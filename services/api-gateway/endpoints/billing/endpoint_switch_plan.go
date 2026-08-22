package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to switch pricing plans.
type SwitchPlanRequest struct {
	// ID of the plan to move the account onto.
	PlanID string `path:"id" validate:"required"`
}

// Switches the account to a different pricing plan, effective immediately.
//
// Free-to-paid, paid-to-free, and paid-to-paid changes are all handled: moving to the free plan cancels the current subscription, while moving to a paid plan subscribes the account at no fewer seats than that plan's seat minimum. A change that owes a prorated amount is charged straight away to the account's payment method on file, so use Preview Plan Change first to see the cost. Moving to a paid plan requires the account to already have a Stripe customer and billing profile.
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
