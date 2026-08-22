package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to preview the cost of a plan change.
type GetPlanProrationRequest struct {
	// ID of the plan the account would switch to.
	PlanID string `path:"id" validate:"required"`
}

// Returns what it would cost to switch the account to a different pricing plan.
//
// The preview covers the prorated amount due now and the estimated recurring monthly bill afterwards. Nothing is charged and the subscription is left unchanged. Amounts are quoted by Stripe where possible; when Stripe cannot quote the change, OpenMRP estimates them and flags the result with `is_estimate`. A switch to the free plan always previews as zero.
type GetPlanChangePreviewEndpoint struct{}

func (e *GetPlanChangePreviewEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPlanProrationRequest, *apiresource.PlanChangeProration] {
	return (&apiendpoint.APIEndpoint[*GetPlanProrationRequest, *apiresource.PlanChangeProration]{
		Title:             "Preview Plan Change",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/plans/{id}/proration",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypePlanChangeProration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPlanProrationRequest) (*apiresource.PlanChangeProration, *apierror.APIError) {
			return svc.(BillingSvc).GetPlanChangePreview
		},
	})
}
