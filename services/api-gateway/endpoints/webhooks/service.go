package webhooksep

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/billing"
	corepb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type WebhookSvc interface {
	ProcessWebhook(ctx context.Context, req *apiresource.StripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError)
	ProcessAccountWebhook(ctx context.Context, req *AccountStripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError)
}

type WebhookSvcConfig struct {
	// BillingClient (required) is the billing-service gRPC client.
	BillingClient pb.BillingServiceClient

	// SalesClient (required) is the core-service sales gRPC client, which verifies and records per-account Stripe webhook events.
	SalesClient corepb.CoreSalesServiceClient
}

type webhookSvcImpl struct {
	billingClient pb.BillingServiceClient
	salesClient   corepb.CoreSalesServiceClient
}

var webhookSvcTracer = tracing.GetTracer("api-gateway.endpoints.webhooks.service")

func (c *WebhookSvcConfig) validate() error {
	if c.BillingClient == nil {
		return fmt.Errorf("webhook endpoint service: billing client is required")
	}
	if c.SalesClient == nil {
		return fmt.Errorf("webhook endpoint service: sales client is required")
	}
	return nil
}

func NewWebhookSvc(config *WebhookSvcConfig) WebhookSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &webhookSvcImpl{
		billingClient: config.BillingClient,
		salesClient:   config.SalesClient,
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
		attrs := []any{
			"error_code", apiErr.Code,
			"error_message", apiErr.PublicMessage,
			"payload_size", len(req.RawBody),
		}
		if apiErr.InternalMessage != "" {
			attrs = append(attrs, "internal_message", apiErr.InternalMessage)
		}
		slog.ErrorContext(ctx, "webhook processing failed", attrs...)
		return nil, apiErr
	}

	slog.InfoContext(ctx, "webhook processed successfully")
	return &apiresource.WebhookResponse{Object: constants.ObjectTypeWebhookResponse, Received: true}, nil
}

func (m *webhookSvcImpl) ProcessAccountWebhook(ctx context.Context, req *AccountStripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError) {
	slog.InfoContext(ctx, "account webhook received at gateway",
		"account_id", req.AccountID,
		"payload_size", len(req.RawBody),
		"signature_present", req.Signature != "",
		"signature_length", len(req.Signature),
	)

	pbReq := &corepb.ProcessAccountStripeWebhookRequest{
		RawPayload:      req.RawBody,
		StripeSignature: req.Signature,
		AccountId:       req.AccountID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, webhookSvcTracer, "service.webhooks.process_account_webhook", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.salesClient.ProcessAccountStripeWebhook(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		attrs := []any{
			"error_code", apiErr.Code,
			"error_message", apiErr.PublicMessage,
			"account_id", req.AccountID,
			"payload_size", len(req.RawBody),
		}
		if apiErr.InternalMessage != "" {
			attrs = append(attrs, "internal_message", apiErr.InternalMessage)
		}
		slog.ErrorContext(ctx, "account webhook processing failed", attrs...)
		return nil, apiErr
	}

	slog.InfoContext(ctx, "account webhook processed successfully", "account_id", req.AccountID)
	return &apiresource.WebhookResponse{Object: constants.ObjectTypeWebhookResponse, Received: true}, nil
}
