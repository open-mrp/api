package feedbackreportep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a feedback report.
type CreateFeedbackReportRequest struct {
	// What the user is reporting, in their own words.
	Message string `json:"message" validate:"required,max=10000"`
	// Dashboard page the user was on when they submitted the report, recorded so the report can be read in context.
	Page field.Optional[string] `json:"page,omitzero" validate:"omitempty,max=2083"`
}

var sampleCreateFeedbackReportRequest = &CreateFeedbackReportRequest{
	Message: "The item list loses its filters when I open an item and come back.",
	Page:    field.Some("/dashboard/items"),
}

func (*CreateFeedbackReportRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateFeedbackReportRequest)
}

// Creates a feedback report from the signed-in user.
//
// The report is attributed to the user who submitted it and the account they were acting on. API keys and agents cannot submit feedback.
type CreateFeedbackReportEndpoint struct{}

func (e *CreateFeedbackReportEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateFeedbackReportRequest, *apiresource.FeedbackReport] {
	return (&apiendpoint.APIEndpoint[*CreateFeedbackReportRequest, *apiresource.FeedbackReport]{
		Title:             "Create Feedback Report",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/core/feedback-reports",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeFeedbackReport,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateFeedbackReportRequest) (*apiresource.FeedbackReport, *apierror.APIError) {
			return svc.(FeedbackReportSvc).CreateFeedbackReport
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeFeedbackReport,
			Fields:     []string{"user"},
		}),
	})
}
