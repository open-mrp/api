package hubspotsyncep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// One company-match decision.
type HubspotCompanyReviewResolutionInput struct {
	// The review being decided.
	ReviewID string `json:"review_id" validate:"required"`
	// What to do with the customer.
	//
	// - `link`: match it to the existing HubSpot company named by `resolved_hubspot_id`.
	// - `create_new`: create a new HubSpot company for it.
	// - `skip`: leave the customer and its orders out of the sync.
	Action constants.HubspotCompanyReviewAction `json:"action" validate:"required"`
	// The HubSpot company id to link to. Required when `action` is `link`.
	ResolvedHubspotID *string `json:"resolved_hubspot_id"`
}

// Request to resolve many company reviews at once.
type BulkResolveHubspotCompanyReviewsRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
	// The decisions to apply. Every review must belong to this sync.
	Reviews []HubspotCompanyReviewResolutionInput `json:"reviews" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkResolveHubspotCompanyReviewsRequest = &BulkResolveHubspotCompanyReviewsRequest{
	Reviews: []HubspotCompanyReviewResolutionInput{
		{ReviewID: apiresource.SampleHubspotCompanyReviewID, Action: constants.HubspotCompanyReviewActionLink, ResolvedHubspotID: &sampleResolvedHubspotID},
	},
}

var sampleResolvedHubspotID = "12345"

func (*BulkResolveHubspotCompanyReviewsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkResolveHubspotCompanyReviewsRequest)
}

// Applies many company-match decisions to a sync at once, the way a reviewed spreadsheet comes back in.
//
// The decisions are validated against the sync synchronously — an unknown review, or one belonging to another sync, fails the whole request — and then applied by a background job. Poll the returned job to follow it: each decision that could not be applied lands in the job's `errors`, keyed by its index in `reviews`, while the rest still take effect.
type BulkResolveHubspotCompanyReviewsEndpoint struct{}

func (e *BulkResolveHubspotCompanyReviewsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkResolveHubspotCompanyReviewsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkResolveHubspotCompanyReviewsRequest, *apiresource.Job]{
		Title:             "Bulk Resolve HubSpot Company Reviews",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations/hubspot/sync/{id}/company-reviews/actions/bulk-resolve",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkResolveHubspotCompanyReviewsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(HubspotSyncSvc).BulkResolveCompanyReviews
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
