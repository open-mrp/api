package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/billing"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
)

const billingOperationTimeout = 25 * time.Second

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

func (c *AuthBillingClient) SetupBillingProfile(ctx context.Context, accountID string) (*domain.BillingProfileResult, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.setup_billing_profile", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SetupBillingProfileResponse, error) {
			return c.client.SetupBillingProfile(ctx, &pb.SetupBillingProfileRequest{
				AccountId: &accountID,
			}, opts...)
		}, rpc.WithTimeout(billingOperationTimeout))
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.BillingProfileResult{
		ProfileID: resp.GetBillingProfileId(),
		CadenceID: resp.GetBillingCadenceId(),
	}, nil
}

func (c *AuthBillingClient) SubscribeToPricingPlan(ctx context.Context, stripeCustomerID, planCode string) *apierror.APIError {
	ctx = prepareCtx(ctx)

	_, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.subscribe_to_pricing_plan", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SubscribeToPricingPlanResponse, error) {
			return c.client.SubscribeToPricingPlan(ctx, &pb.SubscribeToPricingPlanRequest{
				StripeCustomerId: stripeCustomerID,
				PlanCode:         planCode,
			}, opts...)
		}, rpc.WithTimeout(billingOperationTimeout))
	return apiErr
}

func (c *AuthBillingClient) CreateSetupIntent(ctx context.Context, customerID, idempotencyKey string) (*domain.SetupIntentResult, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.create_setup_intent", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSetupIntentResponse, error) {
			return c.client.CreateSetupIntent(ctx, &pb.CreateSetupIntentRequest{
				CustomerId:     customerID,
				IdempotencyKey: idempotencyKey,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.SetupIntentResult{
		SetupIntentID:  resp.SetupIntentId,
		ClientSecret:   resp.ClientSecret,
		PublishableKey: resp.PublishableKey,
	}, nil
}

func (c *AuthBillingClient) GetSetupIntentStatus(ctx context.Context, setupIntentID string) (*domain.SetupIntentResult, *apierror.APIError) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.get_setup_intent_status", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSetupIntentStatusResponse, error) {
			return c.client.GetSetupIntentStatus(ctx, &pb.GetSetupIntentStatusRequest{
				SetupIntentId: setupIntentID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.SetupIntentResult{
		Status:          resp.Status,
		PaymentMethodID: resp.PaymentMethodId,
	}, nil
}

func (c *AuthBillingClient) ValidateStripePricingPlan(ctx context.Context, planCode string) *apierror.APIError {
	ctx = prepareCtx(ctx)

	_, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.validate_stripe_pricing_plan", billingServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ValidateStripePricingPlanResponse, error) {
			return c.client.ValidateStripePricingPlan(ctx, &pb.ValidateStripePricingPlanRequest{
				PlanCode: planCode,
			}, opts...)
		})
	return apiErr
}
