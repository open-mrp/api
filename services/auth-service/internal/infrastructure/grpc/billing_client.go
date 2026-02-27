package grpc

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/billing"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
)

const billingServiceName = "billing-service"

var billingClientTracer = tracing.GetTracer("auth-service.billing_client")

type AuthBillingClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.BillingServiceClient
}

func NewAuthBillingClient(url string) (*AuthBillingClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: billingServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &AuthBillingClient{
		grpcConn: grpcConn,
		client:   pb.NewBillingServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *AuthBillingClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AuthBillingClient) Close() error {
	return c.grpcConn.Close()
}

func (c *AuthBillingClient) GetPlanByCode(ctx context.Context, planCode string) (*domain.PlanInfo, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.get_plan_by_code", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPlanByCodeResponse, error) {
			return c.client.GetPlanByCode(ctx, &pb.GetPlanByCodeRequest{
				PlanCode: planCode,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	plan := resp.Plan
	if plan == nil {
		return nil, apierror.NewResourceNotFoundError("Pricing plan not found for code: " + planCode)
	}

	info := &domain.PlanInfo{
		TypeID:       plan.Id,
		Name:         plan.Name,
		PlanTypeCode: plan.PlanTypeCode,
		PricePerSeat: plan.PricePerSeat,
	}

	if plan.PricePerMonth != nil {
		info.PricePerMonth = plan.PricePerMonth
	}
	if plan.SeatMinimum != nil {
		v := int(*plan.SeatMinimum)
		info.SeatMinimum = &v
	}

	return info, nil
}

func (c *AuthBillingClient) CreateCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*domain.StripeCustomer, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.create_customer", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCustomerResponse, error) {
			return c.client.CreateCustomer(ctx, &pb.CreateCustomerRequest{
				Email:          email,
				Name:           name,
				IdempotencyKey: idempotencyKey,
				Metadata:       metadata,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.StripeCustomer{ID: resp.CustomerId}, nil
}

func (c *AuthBillingClient) CreateCheckoutSession(ctx context.Context, customerID, planCode, returnURL, idempotencyKey string) (*domain.StripeCheckoutSession, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.create_checkout_session", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCheckoutSessionResponse, error) {
			return c.client.CreateCheckoutSession(ctx, &pb.CreateCheckoutSessionRequest{
				CustomerId:     customerID,
				PlanCode:       planCode,
				ReturnUrl:      returnURL,
				IdempotencyKey: idempotencyKey,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.StripeCheckoutSession{
		ID:             resp.SessionId,
		ClientSecret:   resp.ClientSecret,
		PublishableKey: resp.PublishableKey,
	}, nil
}

func (c *AuthBillingClient) GetCheckoutSessionStatus(ctx context.Context, checkoutSessionID string) (*domain.StripeCheckoutSessionStatus, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.get_checkout_session_status", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCheckoutSessionStatusResponse, error) {
			return c.client.GetCheckoutSessionStatus(ctx, &pb.GetCheckoutSessionStatusRequest{
				CheckoutSessionId: checkoutSessionID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := &domain.StripeCheckoutSessionStatus{
		Status: resp.Status,
	}
	if resp.SubscriptionId != nil {
		result.SubscriptionID = *resp.SubscriptionId
	}
	if resp.CustomerId != nil {
		result.CustomerID = *resp.CustomerId
	}

	return result, nil
}
