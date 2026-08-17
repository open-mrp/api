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

// Filters which of a sync's company reviews land in the exported file.
type ExportHubspotCompanyReviewsRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
	// Restrict the file to reviews in this resolution status.
	Status *constants.HubspotCompanyReviewStatus `json:"status"`
}

var sampleExportHubspotCompanyReviewsRequest = &ExportHubspotCompanyReviewsRequest{}

func (*ExportHubspotCompanyReviewsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportHubspotCompanyReviewsRequest)
}

// Starts an export of a sync's company-match review queue and returns the job that tracks it.
//
// The file carries each customer's name, email, and website alongside its candidate HubSpot companies, plus blank `Decision` and `HubSpot Company ID` columns. Filling those in and posting the rows back to the bulk-resolve endpoint applies them, so the queue can be worked outside the dashboard.
type ExportHubspotCompanyReviewsEndpoint struct{}

func (e *ExportHubspotCompanyReviewsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportHubspotCompanyReviewsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportHubspotCompanyReviewsRequest, *apiresource.Job]{
		Title:             "Export HubSpot Company Reviews",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations/hubspot/sync/{id}/company-reviews/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportHubspotCompanyReviewsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(HubspotSyncSvc).ExportCompanyReviews
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
