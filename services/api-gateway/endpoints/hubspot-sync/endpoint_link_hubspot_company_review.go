package hubspotsyncep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to link a company review to an existing HubSpot company.
type LinkHubspotCompanyReviewRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
	// Company review ID.
	ReviewID string `path:"review_id" validate:"required"`
	// The HubSpot company id to link this customer to.
	ResolvedHubspotID string `json:"resolved_hubspot_id" validate:"required"`
}

var sampleLinkHubspotCompanyReviewRequest = &LinkHubspotCompanyReviewRequest{
	ResolvedHubspotID: "12345",
}

func (*LinkHubspotCompanyReviewRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleLinkHubspotCompanyReviewRequest)
}

// Links a reviewed customer to an existing HubSpot company, resolving the review so the sync can proceed.
type LinkHubspotCompanyReviewEndpoint struct{}

func (e *LinkHubspotCompanyReviewEndpoint) Materialize() *apiendpoint.APIEndpoint[*LinkHubspotCompanyReviewRequest, *apiresource.HubspotCompanyReview] {
	return (&apiendpoint.APIEndpoint[*LinkHubspotCompanyReviewRequest, *apiresource.HubspotCompanyReview]{
		Title:               "Link HubSpot Company Review",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/{id}/company-reviews/{review_id}/actions/link",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotCompanyReview,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *LinkHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError) {
			return svc.(HubspotSyncSvc).LinkCompanyReview
		},
	})
}
