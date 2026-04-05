package grpc

import (
	"context"
	"fmt"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/id"
	pb "github.com/augno/api/shared/proto/billing"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	grpclib "google.golang.org/grpc"
)

const billingServiceName = "billing-service"

var billingClientTracer = tracing.GetTracer("agent-service.billing_client")

type AgentBillingClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.BillingServiceClient
}

func NewAgentBillingClient(url string) (*AgentBillingClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: billingServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &AgentBillingClient{
		grpcConn: grpcConn,
		client:   pb.NewBillingServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *AgentBillingClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AgentBillingClient) Close() error {
	return c.grpcConn.Close()
}

func (c *AgentBillingClient) GetStripeCustomerID(ctx context.Context, accountID string) (string, error) {
	// When called from the runner (internal context), there is no idempotency
	// key on the context. Generate one so the billing service's idempotency
	// tracking doesn't reject the call.
	var opts []rpc.ServiceCallOption
	if key, ok := appctx.GetIdempotencyKey(ctx); !ok || key == "" {
		generated, genErr := id.GenID(id.IdempotencyKeyIDPrefix, nil)
		if genErr != nil {
			return "", fmt.Errorf("failed to generate idempotency key: %w", genErr)
		}
		opts = append(opts, rpc.WithIdempotencyKeyOverride(generated))
	}

	ctx = rpc.PrepareServiceCallCtx(ctx, opts...)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.ensure_billing_customer", billingServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.EnsureBillingCustomerResponse, error) {
			return c.client.EnsureBillingCustomer(ctx, &pb.EnsureBillingCustomerRequest{
				AccountId: &accountID,
			}, opts...)
		})
	if apiErr != nil {
		return "", apiErr
	}

	return resp.StripeCustomerId, nil
}
