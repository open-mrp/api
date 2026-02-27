package grpc

import (
	"context"
	"time"

	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	grpclib "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const coreServiceName = "core-service"

var coreClientTracer = tracing.GetTracer("billing-service.core_client")

type BillingCoreClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.CoreServiceClient
}

func NewBillingCoreClient(url string) (*BillingCoreClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: coreServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &BillingCoreClient{
		grpcConn: grpcConn,
		client:   pb.NewCoreServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *BillingCoreClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *BillingCoreClient) Close() error {
	return c.grpcConn.Close()
}

func prepareCtx(ctx context.Context, idempotencyKey string) context.Context {
	opts := []rpc.MetadataOption{rpc.WithIdentity(ctx)}
	if idempotencyKey != "" {
		opts = append(opts, rpc.WithMetadata(contracts.IdempotencyKeyHeader, idempotencyKey))
	}
	return rpc.PrepareRPCCtx(ctx, opts...)
}

func (c *BillingCoreClient) GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError) {
	ctx = rpc.PrepareRPCCtx(ctx, rpc.WithIdentity(ctx))

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_account_by_stripe_customer_id", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetAccountByStripeCustomerIDResponse, error) {
			return c.client.GetAccountByStripeCustomerID(ctx, &pb.GetAccountByStripeCustomerIDRequest{
				StripeCustomerId: stripeCustomerID,
			}, opts...)
		})
	if apiErr != nil {
		return "", "", apiErr
	}

	return resp.AccountId, resp.PlanCode, nil
}

func (c *BillingCoreClient) UpdateAccountSubscription(ctx context.Context, idempotencyKey, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string) *apierror.APIError {
	ctx = prepareCtx(ctx, idempotencyKey)

	var periodEndPb *timestamppb.Timestamp
	if periodEnd != nil {
		periodEndPb = timestamppb.New(*periodEnd)
	}

	_, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.update_account_subscription", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*emptypb.Empty, error) {
			return c.client.UpdateAccountSubscription(ctx, &pb.UpdateAccountSubscriptionRequest{
				AccountId:            accountID,
				SubscriptionStatus:   status,
				PlanCode:             planCode,
				StripeSubscriptionId: stripeSubID,
				CurrentPeriodEnd:     periodEndPb,
				StripeCustomerId:     stripeCustomerID,
			}, opts...)
		})

	return apiErr
}

func (c *BillingCoreClient) ClearAccountStripeCustomer(ctx context.Context, idempotencyKey, accountID string) *apierror.APIError {
	ctx = prepareCtx(ctx, idempotencyKey)

	_, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.clear_account_stripe_customer", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*emptypb.Empty, error) {
			return c.client.ClearAccountStripeCustomer(ctx, &pb.ClearAccountStripeCustomerRequest{
				AccountId: accountID,
			}, opts...)
		})

	return apiErr
}
