package webhooksep

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/billing"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type WebhookSvc interface {
	ProcessWebhook(ctx context.Context, req *apiresource.StripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError)
}

type WebhookSvcConfig struct {
	BillingClient pb.BillingServiceClient
}

type webhookSvcImpl struct {
	billingClient pb.BillingServiceClient
}

var webhookSvcTracer = tracing.GetTracer("api-gateway.endpoints.webhooks.service")

func (c *WebhookSvcConfig) validate() error {
	if c.BillingClient == nil {
		return fmt.Errorf("webhook endpoint service: billing client is required")
	}
	return nil
}

func NewWebhookSvc(config *WebhookSvcConfig) WebhookSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &webhookSvcImpl{
		billingClient: config.BillingClient,
	}
}

func (m *webhookSvcImpl) ProcessWebhook(ctx context.Context, req *apiresource.StripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError) {
	slog.InfoContext(ctx, "webhook received at gateway",
		"payload_size", len(req.RawBody),
		"signature_present", req.Signature != "",
		"signature_length", len(req.Signature),
	)

	pbReq := &pb.ProcessWebhookEventRequest{
		RawPayload:      req.RawBody,
		StripeSignature: req.Signature,
	}

	_, apiErr := grpcutil.CallRPC(ctx, webhookSvcTracer, "service.webhooks.process_webhook", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ProcessWebhookEventResponse, error) {
			return m.billingClient.ProcessWebhookEvent(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		slog.ErrorContext(ctx, "webhook processing failed",
			"error_code", apiErr.Code,
			"error_message", apiErr.PublicMessage,
			"payload_size", len(req.RawBody),
		)
		return nil, apiErr
	}

	slog.InfoContext(ctx, "webhook processed successfully")
	return &apiresource.WebhookResponse{Received: true}, nil
}
