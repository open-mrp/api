package httpgroup

import (
	"fmt"

	feedbackreportep "github.com/open-mrp/api/services/api-gateway/endpoints/feedback-reports"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type FeedbackReportsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type FeedbackReportsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *FeedbackReportsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("feedback reports endpoint group: core client is required")
	}
	return nil
}

func (*FeedbackReportsEndpointGroup) Materialize(config *FeedbackReportsEndpointGroupConfig) *FeedbackReportsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	feedbackReportSvc := feedbackreportep.NewFeedbackReportSvc(&feedbackreportep.FeedbackReportSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Feedback Report",
		Description:  "Submit and review the feedback and bug reports users send from the dashboard.",
		ResourceType: &apiresource.FeedbackReport{},
	}

	createFeedbackReportEndpoint := apiendpoint.From(&feedbackreportep.CreateFeedbackReportEndpoint{}).WithService(inner, feedbackReportSvc)
	listFeedbackReportsEndpoint := apiendpoint.From(&feedbackreportep.ListFeedbackReportsEndpoint{}).WithService(inner, feedbackReportSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listFeedbackReportsEndpoint,
		createFeedbackReportEndpoint,
	}

	return &FeedbackReportsEndpointGroup{inner}
}
