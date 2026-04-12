package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetPlanProrationRequest is the request to preview the cost impact of a plan change.
type GetPlanProrationRequest struct {
	// The ID of the target pricing plan to preview switching to.
	PlanID string `path:"id" validate:"required"`
}

type GetPlanChangePreviewEndpoint struct{}

func (e *GetPlanChangePreviewEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPlanProrationRequest, *apiresource.PlanChangeProration] {
	return &apiendpoint.APIEndpoint[*GetPlanProrationRequest, *apiresource.PlanChangeProration]{
		Title:             "Preview Plan Change",
		Description:       "Returns a proration preview for switching the account to a different pricing plan.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/plans/{id}/proration",
		Request:           &GetPlanProrationRequest{},
		Response:          &apiresource.PlanChangeProration{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPlanProrationRequest) (*apiresource.PlanChangeProration, *apierror.APIError) {
			return svc.(BillingSvc).GetPlanChangePreview
		},
	}
}
