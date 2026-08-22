package grpc

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/id"
	pb "github.com/open-mrp/api/shared/proto/billing"
	"github.com/open-mrp/api/shared/rpc"
	"github.com/open-mrp/api/shared/tracing"

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
	// When called from the runner (internal context), there is no idempotency key on the context. Generate one so the billing service's idempotency tracking doesn't reject the call.
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

// GetAgentSpend returns the account's marked-up token spend for the current billing period (as Stripe will bill it) plus the plan's marked-up per-token rates. Used to enforce the spending cap against the same figures the dashboard shows and Stripe bills.
func (c *AgentBillingClient) GetAgentSpend(ctx context.Context, accountID string) (int64, []domain.AgentTokenRate, error) {
	ctx = rpc.PrepareServiceCallCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, billingClientTracer, "billing_client.get_agent_spend", billingServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetAgentSpendResponse, error) {
			return c.client.GetAgentSpend(ctx, &pb.GetAgentSpendRequest{
				AccountId: &accountID,
			}, opts...)
		})
	if apiErr != nil {
		return 0, nil, apiErr
	}

	rates := make([]domain.AgentTokenRate, 0, len(resp.TokenRates))
	for _, r := range resp.TokenRates {
		rates = append(rates, domain.AgentTokenRate{
			Model:           r.Model,
			TokenType:       r.TokenType,
			UnitAmountCents: r.UnitAmountCents,
		})
	}

	return resp.EstimatedSpendCents, rates, nil
}
