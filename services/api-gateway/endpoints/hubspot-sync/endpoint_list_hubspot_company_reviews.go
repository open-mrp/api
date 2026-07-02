package hubspotsyncep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list a sync job's company-match reviews.
type ListHubspotCompanyReviewsRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
	// Restrict the results to reviews in this resolution status.
	Status *constants.HubspotCompanyReviewStatus `query:"status"`
}

// Lists the company-match review queue for a sync job — the customers that could not be confidently matched to a HubSpot company and need a human decision before the sync executes.
type ListHubspotCompanyReviewsEndpoint struct{}

func (e *ListHubspotCompanyReviewsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListHubspotCompanyReviewsRequest, *apiresource.List[apiresource.HubspotCompanyReview]] {
	return (&apiendpoint.APIEndpoint[*ListHubspotCompanyReviewsRequest, *apiresource.List[apiresource.HubspotCompanyReview]]{
		Title:               "List HubSpot Company Reviews",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/{id}/company-reviews",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotCompanyReview,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListHubspotCompanyReviewsRequest) (*apiresource.List[apiresource.HubspotCompanyReview], *apierror.APIError) {
			return svc.(HubspotSyncSvc).ListCompanyReviews
		},
	})
}
