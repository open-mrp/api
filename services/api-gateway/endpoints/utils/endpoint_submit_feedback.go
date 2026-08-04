package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to submit user feedback.
type SubmitFeedbackRequest struct {
	// The question the user was prompted with.
	Question string `json:"question" validate:"required"`
	// The user's response to the question.
	Answer string `json:"answer" validate:"required"`
	// URL of the page the user was on when they answered, recorded so the feedback can be read in context.
	PageURL field.Optional[string] `json:"page_url,omitzero"`
}

var sampleSubmitFeedbackRequest = &SubmitFeedbackRequest{
	Question: "How would you rate this feature?",
	Answer:   "Very useful, but could use better documentation.",
}

func (*SubmitFeedbackRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSubmitFeedbackRequest)
}

// Submits an answer to an in-product feedback prompt for the Augno team to review.
//
// The submission creates no resource and cannot be read back through the API. The response carries a confirmation message suitable for display.
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
			HideFromRequestLog: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SubmitFeedbackRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(UtilsSvc).SubmitFeedback
		},
	})
}
