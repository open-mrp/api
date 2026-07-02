package domain

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/augno/api/services/auth-service/pkg/types"
)

// GatewayRequest is a single HTTP call into the api-gateway's internal listener, made on behalf of an agent identity. The Path is already resolved (path params substituted); Query and Body carry the remaining inputs.
type GatewayRequest struct {
	Method   string
	Path     string
	Query    url.Values
	Body     json.RawMessage
	Identity *types.Identity
	// IdempotencyKey (optional) is forwarded as the Idempotency-Key header so the gateway dedupes a replayed mutating call. It is set for mutating endpoint-tool calls to a deterministic value derived from the agent run and tool-use IDs, making a re-delivered or re-issued tool call safe to retry without duplicating its side effect.
	IdempotencyKey string
}

// GatewayClient invokes api-gateway endpoints over the trusted internal listener, forwarding the agent identity. It is how generated endpoint-tools reach real API operations.
type GatewayClient interface {
	Do(ctx context.Context, req GatewayRequest) (string, error)
}

// NotificationClient invokes the notification-service email-bridge RPCs an agent uses to reply by email (send) or stage a draft for review on an email-bridged conversation.
type NotificationClient interface {
	SendInboxReply(ctx context.Context, in SendInboxReplyRequest) (messageID string, err error)
	PostReplyDraft(ctx context.Context, in PostReplyDraftRequest) (messageID string, err error)
}

// SendInboxReplyRequest sends an agent's outbound email through the conversation's bound inbox.
type SendInboxReplyRequest struct {
	ConversationID string
	Subject        string
	Body           string
	Cc             []string
	AgentConfigID  string
	AgentRunID     string
	Identity       *types.Identity
}

// PostReplyDraftRequest proposes a customer reply as a real status=draft message held for human
// approval on a customer case (channel resolved server-side). The conversation is fixed by the run;
// the agent supplies only the content, so it never needs a conversation id.
type PostReplyDraftRequest struct {
	ConversationID string
	Body           string
	Subject        string // used only on an email-bridged case (outbound subject); ignored otherwise
	AgentConfigID  string
	AgentRunID     string
	// SourceThreadMessageID records the internal note the draft was composed from (provenance).
	SourceThreadMessageID string
	Identity              *types.Identity
}

// CoreClient provides access to core-service via gRPC.
type CoreClient interface {
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, error)
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, error)
}

// AccountContext holds billing-relevant metadata for an account.
type AccountContext struct {
	IsSandbox                    bool
	OwnerAccountID               string
	PlanCode                     string
	AgentMonthlySpendingCapCents *int64
}

// BillingCustomerResolver resolves the Stripe customer ID for an account and the account's current agent spend.
type BillingCustomerResolver interface {
	GetStripeCustomerID(ctx context.Context, accountID string) (string, error)
	// GetAgentSpendCents returns the account's marked-up token spend for the current billing period, as Stripe will bill it.
	GetAgentSpendCents(ctx context.Context, accountID string) (int64, error)
}
