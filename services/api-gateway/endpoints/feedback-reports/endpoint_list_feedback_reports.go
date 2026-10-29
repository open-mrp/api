package feedbackreportep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
)

// Request to list feedback reports.
type ListFeedbackReportsRequest struct {
	// Opaque cursor token identifying where the page of results starts.
	//
	// Use the `cursor` value embedded in a previous response's `next_page_url` or `previous_page_url` to fetch the adjacent page. Omit to start from the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of results to return in a single page.
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
}

var _ contracts.DocumentedType = (*ListFeedbackReportsRequest)(nil)

func (*ListFeedbackReportsRequest) SchemaExample() any {
	return map[string]any{
		"cursor": pagination.EncodeDocumentationStringCursor(apiresource.SampleAnalyticsPeriodStart, apiresource.SampleFeedbackReportID),
		"limit":  int64(100),
	}
}

// Returns a paginated list of the feedback reports submitted on your account, most recently submitted first.
//
// Only account administrators can list feedback reports.
type ListFeedbackReportsEndpoint struct{}

func (e *ListFeedbackReportsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListFeedbackReportsRequest, *apiresource.List[apiresource.FeedbackReport]] {
	return (&apiendpoint.APIEndpoint[*ListFeedbackReportsRequest, *apiresource.List[apiresource.FeedbackReport]]{
		Title:             "List Feedback Reports",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/feedback-reports",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeFeedbackReport,
		RequiredRoleType:  constants.RoleTypeAdmin,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListFeedbackReportsRequest) (*apiresource.List[apiresource.FeedbackReport], *apierror.APIError) {
			return svc.(FeedbackReportSvc).ListFeedbackReports
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeFeedbackReport,
			Fields:     []string{"user"},
		}),
	})
}
