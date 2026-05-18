package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to submit user feedback.
type SubmitFeedbackRequest struct {
	// Question presented to the user.
	Question string `json:"question" validate:"required"`
	// Answer to the question.
	Answer string `json:"answer" validate:"required"`
	// URL of the page where feedback was submitted.
	PageURL *string `json:"page_url"`
}

var exampleSubmitFeedbackRequest = &SubmitFeedbackRequest{
	Question: "How would you rate this feature?",
	Answer:   "Very useful, but could use better documentation.",
}

func (*SubmitFeedbackRequest) SchemaExample() any {
	return exampleSubmitFeedbackRequest
}

// Submits user feedback for a given question and page.
type SubmitFeedbackEndpoint struct{}

func (e *SubmitFeedbackEndpoint) Materialize() *apiendpoint.APIEndpoint[*SubmitFeedbackRequest, *apiresource.MessageResource] {
	return (&apiendpoint.APIEndpoint[*SubmitFeedbackRequest, *apiresource.MessageResource]{
		Title:             "Submit Feedback",
		Method:            http.MethodPost,
		Route:             "/v1/core/actions/submit-feedback",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SubmitFeedbackRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(UtilsSvc).SubmitFeedback
		},
	})
}
