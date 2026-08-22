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

// Request to resolve a company review by creating a new HubSpot company.
type CreateNewHubspotCompanyReviewRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
	// Company review ID.
	ReviewID string `path:"review_id" validate:"required"`
}

// Resolves a reviewed customer by creating a new HubSpot company for it during the sync (rather than linking to an existing one).
type CreateNewHubspotCompanyReviewEndpoint struct{}

func (e *CreateNewHubspotCompanyReviewEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateNewHubspotCompanyReviewRequest, *apiresource.HubspotCompanyReview] {
	return (&apiendpoint.APIEndpoint[*CreateNewHubspotCompanyReviewRequest, *apiresource.HubspotCompanyReview]{
		Title:               "Create New HubSpot Company For Review",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/{id}/company-reviews/{review_id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotCompanyReview,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateNewHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError) {
			return svc.(HubspotSyncSvc).CreateNewCompanyReview
		},
	})
}
