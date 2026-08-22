package grpc

import (
	"context"
	"time"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/rpc"
	"github.com/open-mrp/api/shared/tracing"

	grpclib "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const coreServiceName = "core-service"

var coreClientTracer = tracing.GetTracer("billing-service.core_client")

type BillingCoreClient struct {
	grpcConn      *contracts.GRPCClientConn
	accountClient pb.CoreAccountServiceClient
	salesClient   pb.CoreSalesServiceClient
}

func NewBillingCoreClient(url string) (*BillingCoreClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: coreServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &BillingCoreClient{
		grpcConn:      grpcConn,
		accountClient: pb.NewCoreAccountServiceClient(grpcConn.Conn()),
		salesClient:   pb.NewCoreSalesServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *BillingCoreClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *BillingCoreClient) Close() error {
	return c.grpcConn.Close()
}

func prepareCtx(ctx context.Context, idempotencyKey string) context.Context {
	if idempotencyKey != "" {
		return rpc.PrepareServiceCallCtx(ctx, rpc.WithIdempotencyKeyOverride(idempotencyKey))
	}
	return rpc.PrepareServiceCallCtx(ctx)
}

func (c *BillingCoreClient) RecordOrderPayment(ctx context.Context, idempotencyKey, salesOrderID, paymentIntentID string) *apierror.APIError {
	ctx = prepareCtx(ctx, idempotencyKey)

	_, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.record_order_payment", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*emptypb.Empty, error) {
			return c.salesClient.RecordOrderPayment(ctx, &pb.RecordOrderPaymentRequest{
				SalesOrderId:    salesOrderID,
				PaymentIntentId: paymentIntentID,
			}, opts...)
		})
	return apiErr
}

func (c *BillingCoreClient) GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError) {
	ctx = prepareCtx(ctx, "")

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_account_by_stripe_customer_id", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetAccountByStripeCustomerIDResponse, error) {
			return c.accountClient.GetAccountByStripeCustomerID(ctx, &pb.GetAccountByStripeCustomerIDRequest{
				StripeCustomerId: stripeCustomerID,
			}, opts...)
		})
	if apiErr != nil {
		return "", "", apiErr
	}

	return resp.AccountId, resp.PlanCode, nil
}

func (c *BillingCoreClient) UpdateAccountSubscription(ctx context.Context, idempotencyKey, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError {
	ctx = prepareCtx(ctx, idempotencyKey)

	var periodEndPb *timestamppb.Timestamp
	if periodEnd != nil {
		periodEndPb = timestamppb.New(*periodEnd)
	}

	_, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.update_account_subscription", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*emptypb.Empty, error) {
			return c.accountClient.UpdateAccountSubscription(ctx, &pb.UpdateAccountSubscriptionRequest{
				AccountId:                 accountID,
				SubscriptionStatus:        status,
				PlanCode:                  planCode,
				StripeSubscriptionId:      stripeSubID,
				CurrentPeriodEnd:          periodEndPb,
				StripeCustomerId:          stripeCustomerID,
				BillingProfileId:          billingProfileID,
				BillingCadenceId:          billingCadenceID,
				PricingPlanSubscriptionId: pricingPlanSubscriptionID,
				ServicingStatus:           servicingStatus,
				CollectionStatus:          collectionStatus,
			}, opts...)
		})

	return apiErr
}

func (c *BillingCoreClient) ClearAccountStripeCustomer(ctx context.Context, idempotencyKey, accountID string) *apierror.APIError {
	ctx = prepareCtx(ctx, idempotencyKey)

	_, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.clear_account_stripe_customer", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*emptypb.Empty, error) {
			return c.accountClient.ClearAccountStripeCustomer(ctx, &pb.ClearAccountStripeCustomerRequest{
				AccountId: accountID,
			}, opts...)
		})

	return apiErr
}
