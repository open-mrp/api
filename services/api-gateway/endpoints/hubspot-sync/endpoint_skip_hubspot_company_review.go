package hubspotsyncep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to skip a company review.
type SkipHubspotCompanyReviewRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
	// Company review ID.
	ReviewID string `path:"review_id" validate:"required"`
}

// Skips a reviewed customer, excluding it (and its orders) from the sync entirely.
type SkipHubspotCompanyReviewEndpoint struct{}

func (e *SkipHubspotCompanyReviewEndpoint) Materialize() *apiendpoint.APIEndpoint[*SkipHubspotCompanyReviewRequest, *apiresource.HubspotCompanyReview] {
	return (&apiendpoint.APIEndpoint[*SkipHubspotCompanyReviewRequest, *apiresource.HubspotCompanyReview]{
		Title:               "Skip HubSpot Company Review",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/{id}/company-reviews/{review_id}/actions/skip",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotCompanyReview,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SkipHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError) {
			return svc.(HubspotSyncSvc).SkipCompanyReview
		},
	})
}
