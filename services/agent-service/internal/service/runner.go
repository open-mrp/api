package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/augno/api/services/agent-service/internal/agents"
	"github.com/augno/api/services/agent-service/internal/domain"
	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/agent-service/internal/llm"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/retry"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

const maxToolLoopIterations = 20

// llmCallTimeout bounds a single LLM turn (streaming or not) as a backstop against a connection that stalls without a clean close — e.g. a streaming response silently severed by an egress NAT/firewall, where the read would otherwise block forever and pin the run in "running". The gateway streaming path has a tighter idle watchdog (streamIdleTimeout); this is the absolute ceiling for the whole attempt. One turn caps at 4096 output tokens, so minutes is generous.
const llmCallTimeout = 5 * time.Minute

var runnerTracer = tracing.GetTracer("agent-service.runner")

// agentConfig is the subset of AgentDefinitionConfig relevant to execution.
type agentConfig struct {
	SystemPrompt string   `json:"system_prompt"`
	Model        string   `json:"model"`
	Provider     string   `json:"provider"`
	Temperature  *float64 `json:"temperature"`
	// Tier selects the model tier (frontier/high/balanced/cheap/legacy); empty = DefaultModelTier. Internal — not user-settable.
	Tier string `json:"tier"`
	// EndpointToolSlugs is the explicit per-agent allow-list of api-gateway endpoint-tools the agent may discover and use. Entries are endpoint-tool slugs; the single entry "*" grants the whole catalog. Empty (default) means the agent gets no endpoint-tools at all. Stored in the agent_definition.config JSON, so granting endpoints needs no migration.
	EndpointToolSlugs []string `json:"endpoint_tool_slugs"`
	// EndpointToolReview is the per-agent override of which granted endpoint-tools require human approval before they execute, keyed by endpoint-tool slug. A true value gates the tool (the run pauses in awaiting_approval when the agent calls it); absent or false means no review, matching the default-off behaviour of linked built-in tools. Stored alongside EndpointToolSlugs in the agent_definition.config JSON, so it needs no migration.
	EndpointToolReview map[string]bool `json:"endpoint_tool_review"`
}

type RunnerConfig struct {
	// Repos (required) is the repository factory for agent persistence.
	Repos domain.RepoFactory

	// ToolRegistry (required) resolves the tool handlers available to agent runs.
	ToolRegistry *agents.ToolHandlerRegistry

	// LLMProviders (required) maps provider names to LLM provider implementations.
	LLMProviders map[string]llm.LLMProvider

	// OutboxRepo (required) is the outbox repository used to enqueue messages.
	OutboxRepo messaging.OutboxRepo

	// CoreClient (required) is the core-service client used to resolve account context and role permissions.
	CoreClient domain.CoreClient

	// GatewayClient (optional; default: nil) invokes api-gateway endpoints for generated endpoint-tools. When nil, those tools return an error if called.
	GatewayClient domain.GatewayClient

	// NotificationClient (optional; default: nil) invokes notification-service RPCs for the draft_reply / send_email tools. When nil, those tools return an error if called.
	NotificationClient domain.NotificationClient

	// Broker (optional; default: nil) is the message broker used to stream run step events. When nil, step events are not published.
	Broker messaging.MessageBroker

	// BillingClient (required) resolves billing customers for spend tracking.
	BillingClient domain.BillingCustomerResolver

	// OutboxNotifier (optional; default: nil) wakes the outbox enqueuer the instant the runner commits a latency-sensitive durable row (the streaming reply bubble, its finalize, run completion), so those post without waiting out the enqueuer's idle poll backoff. When nil, they are still picked up on the next poll.
	OutboxNotifier messaging.OutboxNotifier
}

func (c *RunnerConfig) WithDefaults() *RunnerConfig {
	if c == nil {
		c = &RunnerConfig{}
	}
	return c
}

func (c *RunnerConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("runner service: repos is required")
	}
	if c.ToolRegistry == nil {
		return fmt.Errorf("runner service: tool registry is required")
	}
	if c.LLMProviders == nil {
		return fmt.Errorf("runner service: llm providers is required")
	}
	if c.OutboxRepo == nil {
		return fmt.Errorf("runner service: outbox repo is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("runner service: core client is required")
	}
	if c.BillingClient == nil {
		return fmt.Errorf("runner service: billing client is required")
	}
	return nil
}

type runnerSvc struct {
	repos              domain.RepoFactory
	toolRegistry       *agents.ToolHandlerRegistry
	llmProviders       map[string]llm.LLMProvider
	outboxRepo         messaging.OutboxRepo
	coreClient         domain.CoreClient
	gatewayClient      domain.GatewayClient
	notificationClient domain.NotificationClient
	broker             messaging.MessageBroker
	billingClient      domain.BillingCustomerResolver
	outboxNotifier     messaging.OutboxNotifier
}

func NewRunnerSvc(config *RunnerConfig) domain.RunnerSvc {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &runnerSvc{
		repos:              config.Repos,
		toolRegistry:       config.ToolRegistry,
		llmProviders:       config.LLMProviders,
		outboxRepo:         config.OutboxRepo,
		coreClient:         config.CoreClient,
		gatewayClient:      config.GatewayClient,
		notificationClient: config.NotificationClient,
		broker:             config.Broker,
		billingClient:      config.BillingClient,
		outboxNotifier:     config.OutboxNotifier,
	}
}

// kickOutbox wakes the outbox enqueuer so a just-committed durable row (a streaming reply, its finalize, or run completion) is published immediately rather than on the enqueuer's next idle poll, which can be up to MaxPollInterval away. No-op when no notifier was injected. Call only after the writing Create has returned — kicking before the row is committed would race the poll against an as-yet-invisible row.
func (s *runnerSvc) kickOutbox() {
	if s.outboxNotifier != nil {
		s.outboxNotifier.Notify()
	}
}

// billingContext holds resolved billing state for an agent run.
type billingContext struct {
	billingAccountID  string
	stripeCustomerID  string
	spendingCapCents  *int64 // nil = no cap
	currentSpendCents int64  // estimated spend so far this month
	model             string // for cost estimation during loop
}

func (s *runnerSvc) resolveBillingContext(ctx context.Context, accountID, model string) (*billingContext, error) {
	acctCtx, err := s.coreClient.GetAccountContext(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account context: %w", err)
	}

	billingAccountID := accountID
	if acctCtx.IsSandbox && acctCtx.OwnerAccountID != "" {
		billingAccountID = acctCtx.OwnerAccountID
	}

	if acctCtx.PlanCode == string(constants.PlanCodeFree) {
		return nil, fmt.Errorf("agent usage is not available on the free plan")
	}

	customerID, err := s.billingClient.GetStripeCustomerID(ctx, billingAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Stripe customer: %w", err)
	}
	if customerID == "" {
		return nil, fmt.Errorf("no Stripe customer ID available for billing — agent runs require an active billing customer")
	}

	bc := &billingContext{
		billingAccountID: billingAccountID,
		stripeCustomerID: customerID,
		spendingCapCents: acctCtx.AgentMonthlySpendingCapCents,
		model:            model,
	}

	// If a cap is set, read the current period spend as Stripe will bill it — the same marked-up figure the dashboard shows — so the cap and the displayed number agree. The in-loop gate adds this run's per-turn estimate on top of this baseline to avoid a Stripe round trip per turn.
	if bc.spendingCapCents != nil {
		spendCents, spendErr := s.billingClient.GetAgentSpendCents(ctx, billingAccountID)
		if spendErr != nil {
			return nil, fmt.Errorf("failed to resolve current agent spend: %w", spendErr)
		}

		bc.currentSpendCents = spendCents

		if bc.currentSpendCents >= *bc.spendingCapCents {
			capDollars := float64(*bc.spendingCapCents) / 100.0
			return nil, fmt.Errorf("monthly agent spending cap of $%.2f has been reached", capDollars)
		}
	}

	return bc, nil
}

// chatTurnOriginKey marks whether the current run turn was triggered from its conversation. Only conversation-originated turns post their reply/failure back into the conversation; a free-text continuation typed in the agent-run console is a private fork of the run and stays on the run (it must not leak into the thread, nor into the context of subsequent conversation messages).
type chatTurnOriginKey struct{}

func withChatTurnFromConversation(ctx context.Context, fromConversation bool) context.Context {
	return context.WithValue(ctx, chatTurnOriginKey{}, fromConversation)
}

func chatTurnFromConversation(ctx context.Context) bool {
	v, _ := ctx.Value(chatTurnOriginKey{}).(bool)
	return v
}

// chatStreamPatch* throttle the partial-body updates streamed into a chat reply message. Coarser than the per-token live console deltas (20 chars / 50ms): each patch persists to the DB and pushes to every live subscriber, so ~2-3 writes/sec/run is the right cadence — fine-grained enough to read as live, cheap enough not to hammer the row.
const (
	chatStreamPatchInterval = 400 * time.Millisecond
	chatStreamPatchMinChars = 200
)

// chatStreamKey carries the in-flight streaming reply for a conversation turn so the LLM stream loop can patch partial answer text into the already-posted (empty) agent message — the "true single record" model where the reply is one bubble that fills in, not a live bubble swapped for a persisted one at run end. nil for non-conversation turns (console forks) and non-chat runs.
type chatStreamKey struct{}

type chatStreamState struct {
	messageID       string
	accountID       string
	conversationID  string
	clientMessageID string
	agentName       string

	// mu guards the throttle bookkeeping: completeWithRetry runs the stream callback on one goroutine, but it is invoked once per agent-loop iteration, so the state outlives any single callback.
	mu        sync.Mutex
	lastFlush time.Time
	lastLen   int
}

func withChatStream(ctx context.Context, css *chatStreamState) context.Context {
	return context.WithValue(ctx, chatStreamKey{}, css)
}

func chatStreamFromContext(ctx context.Context) *chatStreamState {
	css, _ := ctx.Value(chatStreamKey{}).(*chatStreamState)
	return css
}

// chatAgentName resolves the agent definition display name for a chat reply. The streaming turn carries it on chatStreamState; otherwise it is loaded from the run's definition id.
func (s *runnerSvc) chatAgentName(ctx context.Context, run *sqlc.AgentRun) string {
	if css := chatStreamFromContext(ctx); css != nil && css.agentName != "" {
		return css.agentName
	}
	def, err := s.repos.NewAgentDefinitionRepo().GetByID(ctx, run.AgentDefinitionID)
	if err != nil {
		return ""
	}
	return def.Name
}

func (s *runnerSvc) ExecuteRun(ctx context.Context, runID, configID, accountID, triggerType string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runnerTracer, "service.runner.execute_run")
	defer span.End()

	// A fresh chat run is, by definition, triggered from its conversation — its reply (and any failure notice) posts back into the thread. Non-chat runs have no conversation and never post.
	ctx = withChatTurnFromConversation(ctx, triggerType == string(constants.AgentTriggerTypeChat))

	startTime := time.Now()
	runRepo := s.repos.NewAgentRunRepo()
	configRepo := s.repos.NewAgentConfigRepo()
	defRepo := s.repos.NewAgentDefinitionRepo()

	run, runErr := runRepo.GetByID(ctx, runID)
	if runErr != nil {
		return apierror.NewInternalError(runErr, fmt.Sprintf("failed to load run %s", runID))
	}

	config, cfgErr := configRepo.GetByID(ctx, configID)
	if cfgErr != nil {
		return apierror.NewInternalError(cfgErr, fmt.Sprintf("failed to load config %s", configID))
	}

	def, defErr := defRepo.GetByID(ctx, config.AgentDefinitionID)
	if defErr != nil {
		return apierror.NewInternalError(defErr, "failed to load definition")
	}

	// Atomically claim the run (pending → running). The claim guards on status='pending', so a
	// redelivered execute_run command — RabbitMQ is at-least-once, and a long LLM turn easily outlives a
	// redelivery window — claims 0 rows and is dropped here, before any trigger event or streaming reply
	// bubble is emitted. Without this a duplicate delivery ran the whole turn again, leaving an orphaned
	// "Thinking" placeholder in the thread. (The inbox also dedups, but this is the authoritative guard.)
	claimed, startErr := runRepo.UpdateStarted(ctx, runID)
	if startErr != nil {
		return apierror.NewInternalError(startErr, "failed to mark run as started")
	}
	if claimed == 0 {
		slog.InfoContext(ctx, "skipping execute_run: run already claimed (duplicate delivery)", "run_id", runID)
		return nil
	}

	// If this run is chat-linked, tell its conversation the run has begun so subscribers can stream the agent's interim steps inline in the thread (best-effort; the final reply still posts).
	s.emitChatRunStarted(ctx, accountID, run)

	// Post the reply message now, empty and in a streaming state, so the thread renders one bubble that fills in token-by-token (via message.updated patches below) and is finalized at run end — rather than a transient live bubble swapped for the persisted reply. A fresh chat run threads its reply under the message that summoned it. No-op for non-conversation turns. The returned context carries the stream handle so the LLM loop and the terminal reply/failure paths address the same record.
	var chatReplyTo string
	if tid := agentdb.StringFromPgText(run.TriggerMessageID); tid != nil {
		chatReplyTo = *tid
	}
	ctx = s.beginChatStream(ctx, run, runID, accountID, chatReplyTo, def.Name)

	var agentCfg agentConfig
	if def.Config != nil {
		if err := json.Unmarshal(def.Config, &agentCfg); err != nil {
			return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("invalid agent config: %s", err.Error()))
		}
	}

	// Resolve the model chain from the agent's tier (or an explicit model override). The runner tries the primary first and fails over to the next, cross-provider model on a provider outage.
	var modelChain []string
	if agentCfg.Model != "" {
		if !domain.AllowedModels[agentCfg.Model] {
			return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("model %q is not allowed", agentCfg.Model))
		}
		modelChain = []string{agentCfg.Model}
	} else {
		// Tier precedence: an explicit per-agent tier in the config wins; otherwise auto-assign by trigger purpose (chat/manual → high, scheduled/event → balanced background work).
		tier := constants.ModelTier(agentCfg.Tier)
		if !tier.IsValid() {
			tier = tierForTrigger(triggerType)
		}
		modelChain = tier.ModelChain()
	}

	// Fail-closed: agent must have a role_id
	roleID := agentdb.StringFromPgText(def.RoleID)
	if roleID == nil || *roleID == "" {
		return s.failRun(ctx, runRepo, runID, startTime, "agent definition has no role_id; cannot execute without permissions")
	}

	// Resolve agent's role permissions
	permissions, err := s.coreClient.GetRolePermissions(ctx, *roleID)
	if err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to resolve agent permissions: %s", err.Error()))
	}

	// Build agent identity
	roleTypeCode := string(constants.RoleTypeAgent)
	agentIdentity := &types.Identity{
		Type:        types.IdentityActorTypeAgent,
		Target:      &types.IdentityTarget{AccountID: accountID},
		AccountMode: constants.AccountModeProduction,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           def.ID,
			Name:         &def.Name,
			AccountID:    &accountID,
			RoleID:       roleID,
			RoleType:     &roleTypeCode,
			Permissions:  permissions,
		},
	}

	// Store identity in context so outgoing gRPC calls propagate it
	ctx = appctx.WithIdentity(ctx, agentIdentity)

	// Resolve billing context and inject Stripe customer ID. This must run after the identity is in context: the billing service resolves the Stripe customer from the caller's identity, so resolving billing earlier would send the gRPC call without identity metadata and fail.
	bc, billingErr := s.resolveBillingContext(ctx, accountID, modelChain[0])
	if billingErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, billingErr.Error())
	}
	ctx = llm.WithStripeCustomerID(ctx, bc.stripeCustomerID)

	// Load linked tools from DB
	toolRepo := s.repos.NewAgentDefinitionToolRepo()
	linkedTools, toolErr := toolRepo.ListByAgentDefinitionID(ctx, def.ID)
	if toolErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to load tools: %s", toolErr.Error()))
	}

	// Resolve the agent's endpoint-tool grant and build the up-front tool list (linked + built-in + search meta-tool) and review map.
	allowedEndpointTools := resolveAllowedEndpointTools(agentCfg.EndpointToolSlugs)
	toolDefs, requireReviewBySlug := s.buildAgentToolDefs(linkedTools, allowedEndpointTools, agentCfg.EndpointToolReview)

	// Resolve temperature
	temperature := 0.0
	if agentCfg.Temperature != nil {
		temperature = *agentCfg.Temperature
	}

	// Execute the agent
	result, err := s.executeAgent(ctx, run, config, def, accountID, agentIdentity, agentCfg.SystemPrompt, modelChain, toolDefs, temperature, requireReviewBySlug, allowedEndpointTools, bc)
	if err != nil {
		// Transient, side-effect-free failures are re-enqueued with backoff instead of surfaced as a terminal failure.
		if s.maybeAutoRetry(ctx, runRepo, run, err) {
			return nil
		}
		return s.failRun(ctx, runRepo, runID, startTime, err.Error())
	}

	// Persist outputs
	if err := s.persistOutputs(ctx, runID, accountID, bc.billingAccountID, result); err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to persist outputs: %s", err.Error()))
	}

	// Finalize run
	durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
	statusCode := domain.RunStatusCompleted
	switch {
	case result.Cancelled:
		// The run was stopped mid-flight; keep it cancelled so the stop request isn't overwritten by the normal completed/awaiting_input transition. The partial output is still persisted below. Cancellation is not a completion — leave completed_at/duration_ms null.
		if cancelErr := runRepo.UpdateCancelled(ctx, sqlc.UpdateAgentRunCancelledParams{
			Output:            result.Output,
			TotalInputTokens:  int64(result.InputTokens),
			TotalOutputTokens: int64(result.OutputTokens),
			ID:                runID,
		}); cancelErr != nil {
			return apierror.NewInternalError(cancelErr, "failed to finalize cancelled run")
		}
		statusCode = domain.RunStatusCancelled
	case result.AwaitingApproval:
		statusCode = domain.RunStatusAwaitingApproval
	case triggerType == string(constants.AgentTriggerTypeManual) || triggerType == string(constants.AgentTriggerTypeChat):
		// Manual and chat runs stay open for follow-ups (a chat run is continued when the user replies to its message) rather than completing after a single turn.
		statusCode = domain.RunStatusAwaitingInput
	}
	if statusCode != domain.RunStatusCancelled {
		if completeErr := runRepo.UpdateCompleted(ctx, sqlc.UpdateAgentRunCompletedParams{
			StatusCode:        statusCode,
			Output:            result.Output,
			DurationMs:        agentdb.PgInt4(durationMs),
			TotalInputTokens:  int64(result.InputTokens),
			TotalOutputTokens: int64(result.OutputTokens),
			ID:                runID,
		}); completeErr != nil {
			return apierror.NewInternalError(completeErr, "failed to finalize run")
		}
	}

	// Write outbox message for run completion
	s.writeRunCompletedEvent(ctx, runID, accountID, bc.billingAccountID, result)

	// Notify the triggering user that their run finished (only for fully completed runs).
	if statusCode == domain.RunStatusCompleted {
		s.writeRunCompletedNotification(ctx, run, def, runID, accountID)
	}
	// A chat-triggered run posts its reply back into the conversation each turn (it pauses as awaiting_input rather than completing). A run that pauses on tool approval posts too: the reply body is the approval request ("I need your approval to run: X"), and it carries the run id so the chat UI can render approve/deny controls against the awaiting_approval run.
	if triggerType == string(constants.AgentTriggerTypeChat) {
		// Finalize the streaming reply: set the completed body and flip it to complete (threaded under the message that summoned the agent — only set for directed mention/keyword triggers, so "always" runs finalize un-threaded). If the stream never started (early infra failure), this falls back to creating the reply outright.
		s.writeChatComplete(ctx, run, runID, accountID, result, chatReplyTo)
	}

	slog.Info("Agent run completed",
		"run_id", runID,
		"agent", def.Slug,
		"duration_ms", durationMs,
		"input_tokens", result.InputTokens,
		"output_tokens", result.OutputTokens,
		"actions", len(result.Actions),
	)

	return nil
}

func (s *runnerSvc) emitDelta(ctx context.Context, runID, accountID string, deltaSeq int, content string) {
	if s.broker == nil {
		return
	}
	eventID := fmt.Sprintf("delta-%s-%d", runID, deltaSeq)
	stepData := messaging.AgentRunStepData{
		AgentRunID: runID,
		AccountID:  accountID,
		EventID:    eventID,
		StepType:   "content_delta",
		Title:      "Content delta",
		Content:    &content,
		Sequence:   deltaSeq,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	dataBytes, _ := json.Marshal(stepData)
	_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange,
		string(contracts.AgentEventRunStep), contracts.AmqpMessage{Data: dataBytes})
}

// emitReasoningDelta streams a chunk of the model's native reasoning to the run's WebSocket as an ephemeral reasoning_delta step. Mirrors emitDelta but on the reasoning channel — operators see it in the live thinking panel; it is never persisted (the consolidated "thinking" step is).
func (s *runnerSvc) emitReasoningDelta(ctx context.Context, runID, accountID string, deltaSeq int, content string) {
	if s.broker == nil {
		return
	}
	eventID := fmt.Sprintf("reasoning-%s-%d", runID, deltaSeq)
	stepData := messaging.AgentRunStepData{
		AgentRunID: runID,
		AccountID:  accountID,
		EventID:    eventID,
		StepType:   "reasoning_delta",
		Title:      "Reasoning delta",
		Content:    &content,
		Sequence:   deltaSeq,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	dataBytes, _ := json.Marshal(stepData)
	_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange,
		string(contracts.AgentEventRunStep), contracts.AmqpMessage{Data: dataBytes})
}

// emitToolCallStart streams an ephemeral "tool_call_delta" marker to the run's WebSocket the moment the model begins a tool call, so the live chat indicator can show "calling <tool>" while the arguments are still streaming. Like the content/reasoning delta channels it is never persisted — the durable tool_call step (carrying the assembled input) is recorded by the agent loop once the turn completes.
func (s *runnerSvc) emitToolCallStart(ctx context.Context, runID, accountID, toolUseID, toolName string) {
	if s.broker == nil {
		return
	}
	meta, _ := json.Marshal(map[string]any{"tool_use_id": toolUseID, "tool_name": toolName})
	stepData := messaging.AgentRunStepData{
		AgentRunID: runID,
		AccountID:  accountID,
		EventID:    "toolstart-" + runID + "-" + toolUseID,
		StepType:   "tool_call_delta",
		Title:      toolName,
		Metadata:   meta,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	dataBytes, _ := json.Marshal(stepData)
	_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange,
		string(contracts.AgentEventRunStep), contracts.AmqpMessage{Data: dataBytes})
}

// emitThinkingStep records a turn's reasoning as a durable "thinking" timeline step. The reasoning already streamed live token-by-token (reasoning_delta); this is the persisted record. Content is the native thinking blocks' text (Anthropic) or the accumulated OpenAI-compat reasoning. No-op when the turn produced no reasoning.
func (s *runnerSvc) emitThinkingStep(ctx context.Context, runID, accountID string, seq *int, resp *llm.ToolResponse, iterStart time.Time) {
	reasoning := concatThinking(resp.Thinking)
	if reasoning == "" {
		return
	}
	iterDurationMs := safeconv.Int64ToInt32(time.Since(iterStart).Milliseconds())
	thinkingMeta, _ := json.Marshal(map[string]any{
		"input_tokens":  resp.InputTokens,
		"output_tokens": resp.OutputTokens,
	})
	s.emitEvent(ctx, runID, accountID, seq, "thinking", "Reasoning", &reasoning, &iterDurationMs, nil, thinkingMeta)
}

// concatThinking joins a response's reasoning blocks into a single string.
func concatThinking(blocks []llm.ThinkingBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		sb.WriteString(b.Text)
	}
	return strings.TrimSpace(sb.String())
}

// actorFromContext pulls the acting identity (id, type, name) out of the request context, defaulting to empty strings when no identity is present.
func actorFromContext(ctx context.Context) (actorID, actorType, actorName string) {
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil {
		actorType = string(identity.Type)
		if identity.Actor != nil {
			actorID = identity.Actor.ID
			if identity.Actor.Name != nil {
				actorName = *identity.Actor.Name
			}
		}
	}
	return actorID, actorType, actorName
}

func (s *runnerSvc) emitEvent(ctx context.Context, runID, accountID string, seq *int, stepType, title string, content *string, durationMs *int32, actionID *string, metadata json.RawMessage) {
	actorID, actorType, actorName := actorFromContext(ctx)
	s.emitEventAs(ctx, runID, accountID, seq, stepType, title, content, durationMs, actionID, metadata, actorID, actorType, actorName, false)
}

// emitTerminalEvent emits a final run step (e.g. the "Run failed" marker) and flags it terminal so the WS gateway also sends a run_complete frame, letting the frontend leave its loading state on a failed run.
func (s *runnerSvc) emitTerminalEvent(ctx context.Context, runID, accountID string, seq *int, stepType, title string, content *string, durationMs *int32, actionID *string, metadata json.RawMessage) {
	actorID, actorType, actorName := actorFromContext(ctx)
	s.emitEventAs(ctx, runID, accountID, seq, stepType, title, content, durationMs, actionID, metadata, actorID, actorType, actorName, true)
}

func (s *runnerSvc) emitEventAs(ctx context.Context, runID, accountID string, seq *int, stepType, title string, content *string, durationMs *int32, actionID *string, metadata json.RawMessage, actorID, actorType, actorName string, terminal bool) {
	eventID, err := id.GenID(id.AgentRunEventIDPrefix, nil)
	if err != nil {
		slog.Error("Failed to generate event ID", "error", err, "run_id", runID)
		return
	}
	if metadata == nil {
		metadata = json.RawMessage(`{}`)
	}

	eventRepo := s.repos.NewAgentRunEventRepo()
	params := sqlc.InsertAgentRunEventParams{
		ID:         eventID,
		AgentRunID: runID,
		AccountID:  accountID,
		StepType:   stepType,
		Title:      title,
		Sequence:   safeconv.IntToInt32(*seq),
		Metadata:   metadata,
	}
	if content != nil {
		params.Content = agentdb.PgText(*content)
	}
	if durationMs != nil {
		params.DurationMs = agentdb.PgInt4(*durationMs)
	}
	if actionID != nil {
		params.AgentActionID = agentdb.PgText(*actionID)
	}
	if actorID != "" {
		params.ActorID = agentdb.PgText(actorID)
	}
	if actorType != "" {
		params.ActorType = agentdb.PgText(actorType)
	}
	if actorName != "" {
		params.ActorName = agentdb.PgText(actorName)
	}
	if insertErr := eventRepo.Insert(ctx, params); insertErr != nil {
		slog.Error("Failed to insert run event", "error", insertErr, "run_id", runID, "step_type", stepType)
		return
	}
	*seq++

	// Publish to RabbitMQ for real-time WebSocket streaming (fire-and-forget).
	if s.broker != nil {
		stepData := messaging.AgentRunStepData{
			AgentRunID: runID,
			AccountID:  accountID,
			EventID:    eventID,
			StepType:   stepType,
			Title:      title,
			Content:    content,
			Sequence:   int(params.Sequence),
			DurationMs: durationMs,
			ActionID:   actionID,
			Metadata:   metadata,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			ActorID:    actorID,
			ActorType:  actorType,
			ActorName:  actorName,
			Terminal:   terminal,
		}
		dataBytes, _ := json.Marshal(stepData)
		_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange,
			string(contracts.AgentEventRunStep), contracts.AmqpMessage{Data: dataBytes})
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// chatResourceLinkPreamble instructs a chat-triggered agent to link the records it looks up so they render as clickable in-app links in the conversation. The `augno:<object>/<id>` form is resolved client-side to the correct shell-aware route, so the agent never has to know real URLs — it just copies the `object` and `id` it already sees in tool results.
const chatResourceLinkPreamble = `When you reference a specific record you looked up (sales order, purchase order, invoice, customer, or product), link to it so the user can open it. Write the link in markdown as [<label>](augno:<object>/<id>), taking <object> and <id> verbatim from that record's "object" and "id" fields in the tool result and using its human-readable number or name as <label>. For example, a sales order {"id":"so_abc","object":"sales_order","number":"SO-1042"} becomes [SO-1042](augno:sales_order/so_abc). Link each record the first time you mention it. Only link records you actually retrieved — never guess or invent an id.`

// chatReplyDeliveryPreamble tells a chat-triggered agent how its output reaches the conversation, so it
// stops trying to "post" or "send" messages through generic API tools and stalling on a conversation id
// it was never handed (the "provide the conversation ID so I can send the reply" failure). Its written
// answer is delivered automatically as an internal team note; an outbound reply to the case's external
// party is proposed via the draft_reply tool (when granted) and held for human approval — never sent by
// the agent.
const chatReplyDeliveryPreamble = `Your written response is delivered into this conversation automatically the moment you finish — it posts as an internal note to the team. Never call a generic messaging or "post message" API tool to deliver it, and never ask for or wait on a conversation ID: you do not have one, do not need one, and will not be given one. To reply to the person the case is with — a customer, supplier, or other external contact — call the draft_reply tool with the exact message to send them; that proposes a draft a teammate approves and sends. This is also how you reply by EMAIL: on an email-bridged case the approved draft goes out as an email (set the optional subject), so use draft_reply for email cases too rather than any other tool. You never send outbound messages yourself. If you don't have that tool, just state your proposed reply in your response for a teammate to send. Write outbound text as the message to the recipient, not as a note to the team.`

// isChatRun reports whether a run is linked to a conversation (so it streams native reasoning into the thread and links the records it references).
func isChatRun(run *sqlc.AgentRun) bool {
	cid := agentdb.StringFromPgText(run.ConversationID)
	return cid != nil && *cid != ""
}

// toolDiscoveryPreamble tells the agent what API areas it can act on and that specific operations are found via search_api_tools (they aren't all listed up front), so it stops guessing or giving up when it doesn't see a matching tool. Empty when the agent has no endpoint-tool grant.
func toolDiscoveryPreamble(allowedEndpointTools map[string]bool) string {
	groups := agents.AllowedToolGroups(allowedEndpointTools)
	if len(groups) == 0 {
		return ""
	}
	return "You can look up data and take actions through tools. Beyond the tools already listed for you, you have access to API operations across these areas: " + strings.Join(groups, ", ") + ". Those specific operations are NOT all shown to you up front — call the search_api_tools tool with a plain-language description of what you need (for example \"list open sales orders\" or \"create a customer\") to find the matching operation and make it callable, then call it. Whenever a task needs data or an action you don't already have a tool for, search for it before telling the user you can't do it. When a record's related objects (like parent_account, owner, addresses, or type) come back as null, they simply weren't expanded — call the same tool again with its `include` parameter set to the field keys you need to get those full objects. Always expand to get authoritative related data rather than guessing relationships from names or numbers. Don't use emojis in your responses."
}

// augmentSystemPrompt prepends cross-cutting guidance to the agent's configured prompt — tool discovery for every run, plus the resource-link convention for chat runs. Applied on each turn (including continuations), since the prompt is rebuilt and re-sent on every model call.
func (s *runnerSvc) augmentSystemPrompt(systemPrompt string, run *sqlc.AgentRun, allowedEndpointTools map[string]bool) string {
	if tp := toolDiscoveryPreamble(allowedEndpointTools); tp != "" {
		systemPrompt = tp + "\n\n" + systemPrompt
	}
	if isChatRun(run) {
		// Resource-link convention, chat runs only. Reasoning vs. answer is now split by the provider's native reasoning channel, not a respond_to_user tool.
		systemPrompt = chatResourceLinkPreamble + "\n\n" + systemPrompt
		// The agent's answer is auto-posted as the reply — keep it from trying to deliver the message
		// itself (and stalling on a conversation id it has no need for). Prepended last so it leads.
		systemPrompt = chatReplyDeliveryPreamble + "\n\n" + systemPrompt
	}
	return systemPrompt
}

func (s *runnerSvc) executeAgent(
	ctx context.Context,
	run *sqlc.AgentRun,
	config *sqlc.AgentConfig,
	def *sqlc.AgentDefinition,
	accountID string,
	identity *types.Identity,
	systemPrompt string,
	modelChain []string,
	toolDefs []llm.ToolDefinition,
	temperature float64,
	requireReviewBySlug map[string]bool,
	allowedEndpointTools map[string]bool,
	bc *billingContext,
) (*domain.RunResult, error) {
	seq := 0

	// Load memories
	memoryRepo := s.repos.NewAgentMemoryRepo()
	memories, memErr := memoryRepo.ListAccountMemories(ctx, accountID, accountID, 50)
	if memErr != nil {
		memories = nil // non-fatal
	}

	// Append memories to system prompt
	if len(memories) > 0 {
		var sb strings.Builder
		sb.WriteString(systemPrompt)
		sb.WriteString("\n\nRelevant memories from previous interactions:\n")
		for _, m := range memories {
			fmt.Fprintf(&sb, "- [%s] %s\n", m.Category, m.Content)
		}
		systemPrompt = sb.String()
	}

	// Prepend cross-cutting guidance (tool discovery; resource links for chat runs).
	systemPrompt = s.augmentSystemPrompt(systemPrompt, run, allowedEndpointTools)

	// Build initial messages from run input
	var inputText string
	hasUserMessage := false
	var history []domain.ChatHistoryMessage
	if run.Input != nil {
		var inputData struct {
			Message string                      `json:"message"`
			History []domain.ChatHistoryMessage `json:"history"`
		}
		if err := json.Unmarshal(run.Input, &inputData); err == nil && (inputData.Message != "" || len(inputData.History) > 0) {
			inputText = inputData.Message
			history = inputData.History
			hasUserMessage = inputData.Message != ""
		} else if raw := strings.TrimSpace(string(run.Input)); raw != "" && raw != "{}" {
			inputText = raw
			hasUserMessage = true
		}
	}
	if inputText == "" {
		inputText = "Process the input and take appropriate action."
	}

	// Seed prior thread turns (chat-triggered runs) ahead of the triggering message so the agent has conversation context, and emit each as an event so the run view shows the full context the agent received — not just the trigger. The agent's own past replies are "assistant"; a person's turn is "user" and prefixed with their name when known so multi-party threads stay attributable. These events also let a continued run rebuild the thread from its own log (see reconstructMessages).
	messages := make([]llm.Message, 0, len(history)+1)
	for _, h := range history {
		if h.Role == "assistant" {
			messages = append(messages, llm.Message{Role: "assistant", Content: h.Body})
			body := h.Body
			s.emitEvent(ctx, run.ID, accountID, &seq, "assistant_message", "Earlier reply", &body, nil, nil, nil)
			continue
		}
		content := h.Body
		if h.Name != "" {
			content = h.Name + ": " + h.Body
		}
		messages = append(messages, llm.Message{Role: "user", Content: content})
		s.emitEvent(ctx, run.ID, accountID, &seq, "user_message", "Earlier message", &content, nil, nil, nil)
	}

	// Emit trigger_received event
	s.emitEvent(ctx, run.ID, accountID, &seq, "trigger_received", "Run triggered", &inputText, nil, nil, nil)

	// Only emit user_message event if the client provided an actual message
	if hasUserMessage {
		s.emitEvent(ctx, run.ID, accountID, &seq, "user_message", "User message", &inputText, nil, nil, nil)
	}

	messages = append(messages, llm.Message{Role: "user", Content: inputText})

	// Create run context for handler
	runCtx := &domain.HandlerRunContext{
		AccountID:                accountID,
		RunID:                    run.ID,
		Definition:               def,
		Config:                   config,
		Repos:                    s.repos,
		CoreClient:               s.coreClient,
		GatewayClient:            s.gatewayClient,
		NotificationClient:       s.notificationClient,
		ConversationID:           ptrutil.Deref(agentdb.StringFromPgText(run.ConversationID)),
		Identity:                 identity,
		RequireReviewBySlug:      requireReviewBySlug,
		AlwaysAllowedSlugs:       make(map[string]bool),
		OneTimeApprovedSlugs:     make(map[string]bool),
		AllowedEndpointToolSlugs: allowedEndpointTools,
		RevealedToolSlugs:        make(map[string]bool),
	}

	return s.runAgentLoop(ctx, run, accountID, identity, systemPrompt, modelChain, toolDefs, temperature, messages, &seq, runCtx, bc.spendingCapCents, bc.currentSpendCents)

}

// completeWithRetry performs one LLM call (streaming-aware) and retries on a retryable gateway error (429/529/5xx) with backoff, honoring Retry-After. Returns the response or the last error; the caller decides whether to fail over to another model.
func (s *runnerSvc) completeWithRetry(ctx context.Context, runID, accountID string, seq *int, provider llm.LLMProvider, llmReq *llm.ToolRequest, retryCfg *retry.Config) (*llm.ToolResponse, error) {
	callLLM := func() (*llm.ToolResponse, error) {
		// Bound each attempt so a stalled connection surfaces as an error instead of hanging the run forever. A fresh deadline per attempt means retries/failover aren't starved by an earlier slow call.
		callCtx, cancel := context.WithTimeout(ctx, llmCallTimeout)
		defer cancel()
		if sp, ok := provider.(llm.StreamingLLMProvider); ok {
			// Two independent live streams flushed on the same cadence (20 chars or 50ms): the answer (content_delta) and the reasoning (reasoning_delta). Reasoning is also accumulated so a persisted "thinking" step can be recorded for providers that don't return signed thinking blocks (the OpenAI-compat path).
			var deltaSeq int
			var deltaBuf strings.Builder
			var lastFlush time.Time
			flushDelta := func() {
				if deltaBuf.Len() == 0 {
					return
				}
				s.emitDelta(ctx, runID, accountID, deltaSeq, deltaBuf.String())
				deltaSeq++
				deltaBuf.Reset()
				lastFlush = time.Now()
			}
			var rSeq int
			var rBuf, allReasoning strings.Builder
			var rLastFlush time.Time
			// answerBuf accumulates this call's user-facing answer text so it can be streamed (coarse- throttled) into the persisted chat reply message. Reset when a tool call begins, so the chat bubble drops the interim narration and shows the thinking/tool indicator — mirroring the live console, which clears its streamed answer on a tool call.
			var answerBuf strings.Builder
			// Tracks which tool indices have already had a live "calling <tool>" marker emitted, so the repeated argument deltas for the same tool don't re-emit it.
			startedTools := make(map[int]bool)
			flushReasoning := func() {
				if rBuf.Len() == 0 {
					return
				}
				s.emitReasoningDelta(ctx, runID, accountID, rSeq, rBuf.String())
				rSeq++
				rBuf.Reset()
				rLastFlush = time.Now()
			}
			r, err := sp.StreamCompleteWithTools(callCtx, llmReq, func(ev llm.StreamEvent) {
				switch ev.Type {
				case "content_delta":
					if ev.ContentDelta == "" {
						return
					}
					deltaBuf.WriteString(ev.ContentDelta)
					if deltaBuf.Len() >= 20 || time.Since(lastFlush) >= 50*time.Millisecond {
						flushDelta()
					}
					// Stream the growing answer into the persisted chat reply message (coarse-throttled).
					answerBuf.WriteString(ev.ContentDelta)
					s.maybePatchChatStream(ctx, answerBuf.String())
				case "reasoning_delta":
					if ev.ReasoningDelta == "" {
						return
					}
					allReasoning.WriteString(ev.ReasoningDelta)
					rBuf.WriteString(ev.ReasoningDelta)
					if rBuf.Len() >= 20 || time.Since(rLastFlush) >= 50*time.Millisecond {
						flushReasoning()
					}
				case "tool_call_delta":
					// First signal that the model has begun a tool call — the block-start delta carries the tool name; the argument deltas that follow don't. Emit a live, ephemeral marker once per tool so chat can switch to the "calling <tool>" phase while the arguments still stream. The durable tool_call step (with the assembled input) is recorded post-turn by the agent loop.
					if ev.ToolName == "" || startedTools[ev.ToolIndex] {
						return
					}
					startedTools[ev.ToolIndex] = true
					s.emitToolCallStart(ctx, runID, accountID, ev.ToolCallID, ev.ToolName)
					// The model is taking an action this turn — clear the partial answer so the chat bubble reverts to the thinking/tool indicator instead of stranding interim text.
					answerBuf.Reset()
					s.patchChatStream(ctx, "")
				}
			})
			flushDelta()
			flushReasoning()
			// Reflect this call's final answer text in the chat bubble even if the throttle hasn't fired, so the persisted body is current during any tool-execution gap before the turn finalizes.
			s.patchChatStream(ctx, answerBuf.String())
			if r != nil && len(r.Thinking) == 0 && allReasoning.Len() > 0 {
				r.Thinking = []llm.ThinkingBlock{{Text: allReasoning.String()}}
			}
			return r, err
		}
		return provider.CompleteWithTools(callCtx, llmReq)
	}

	resp, llmErr := callLLM()
	if llmErr == nil {
		return resp, nil
	}
	var gatewayErr *llm.GatewayError
	if !(errors.As(llmErr, &gatewayErr) && gatewayErr.Retryable) {
		return nil, llmErr
	}
	retried := false
	retryCfgForCall := *retryCfg
	if ra := gatewayErr.RetryAfter(); ra > 0 && ra < retryCfgForCall.MaxWait {
		retryCfgForCall.InitialWait = ra
	}
	retryErr := retry.WithBackoff(ctx, &retryCfgForCall, func() error {
		retryResp, err := callLLM()
		if err != nil {
			var ge *llm.GatewayError
			if errors.As(err, &ge) && ge.Retryable {
				return err
			}
			llmErr = err
			return nil
		}
		resp = retryResp
		llmErr = nil
		retried = true
		return nil
	})
	if retryErr != nil {
		llmErr = retryErr
	}
	if retried {
		retryMeta, _ := json.Marshal(map[string]any{"original_status": gatewayErr.StatusCode, "resolved": true})
		s.emitEvent(ctx, runID, accountID, seq, "retry", "LLM call succeeded after retry", nil, nil, nil, retryMeta)
	}
	if llmErr != nil {
		return nil, llmErr
	}
	return resp, nil
}

// capStoppedResult builds the terminal RunResult for a run halted by a spending/billing limit. It emits a cap_exceeded event and returns a normal (non-error) result so the run finalizes cleanly with the limit message as its output, rather than surfacing as a failed run.
func (s *runnerSvc) capStoppedResult(ctx context.Context, run *sqlc.AgentRun, accountID string, seq *int, runCtx *domain.HandlerRunContext, message string, inputTokens, outputTokens int, providerName, modelName string) *domain.RunResult {
	s.emitEvent(ctx, run.ID, accountID, seq, "cap_exceeded", message, nil, nil, nil, nil)
	outputJSON, _ := json.Marshal(map[string]string{"response": message})
	return &domain.RunResult{
		Output:       outputJSON,
		Actions:      runCtx.Actions,
		Artifacts:    runCtx.Artifacts,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		LLMProvider:  providerName,
		LLMModel:     modelName,
	}
}

// runCancelled reports whether the run has been stopped out-of-band. CancelRun flips the run's DB status to "cancelled" from a separate request (and possibly a different replica), so the agent loop polls this cooperatively to halt promptly — including a runaway tool loop — rather than running to completion. A read error is treated as "not cancelled" so a transient blip never aborts a healthy run.
func (s *runnerSvc) runCancelled(ctx context.Context, runID string) bool {
	run, err := s.repos.NewAgentRunRepo().GetByID(ctx, runID)
	if err != nil || run == nil {
		return false
	}
	return run.StatusCode == domain.RunStatusCancelled
}

// cancelledResult builds the terminal RunResult for a run halted mid-flight. It emits a cancelled event (so the run timeline and live view show the stop) and returns a normal, non-error result that carries any work done before the stop. The caller finalizes the run as cancelled.
func (s *runnerSvc) cancelledResult(ctx context.Context, run *sqlc.AgentRun, accountID string, seq *int, runCtx *domain.HandlerRunContext, inputTokens, outputTokens int, providerName, modelName string) *domain.RunResult {
	msg := "Run stopped — the work was halted before completion at the user's request."
	s.emitEvent(ctx, run.ID, accountID, seq, "cancelled", "Run cancelled", &msg, nil, nil, nil)
	outputJSON, _ := json.Marshal(map[string]string{"response": msg})
	return &domain.RunResult{
		Output:       outputJSON,
		Actions:      runCtx.Actions,
		Artifacts:    runCtx.Artifacts,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		LLMProvider:  providerName,
		LLMModel:     modelName,
		Cancelled:    true,
	}
}

// summarizeBlockedAction generates a single plain-language sentence describing the tool call(s) a run is pausing on, for a human approver. It is used when the agent emitted no narration of its own (the usual case once native reasoning is on — the model's thinking is hidden and the tool turn carries no user-facing text). A compact transcript is sent so the model can resolve raw IDs to the human names it already fetched earlier in the run. Best-effort: any error or empty result returns "" and the caller falls back to naming the raw tools. Never blocks the pause.
func (s *runnerSvc) summarizeBlockedAction(ctx context.Context, provider llm.LLMProvider, modelName string, messages []llm.Message, blockedCalls []llm.ToolCall) string {
	if provider == nil || len(blockedCalls) == 0 {
		return ""
	}

	// Compact transcript digest (text, tool calls, tool results) so the summarizer has the context that names the records. Keep only the tail — the relevant lookups are recent and this bounds tokens.
	var ctxBuf strings.Builder
	for _, m := range messages {
		if c := strings.TrimSpace(m.Content); c != "" {
			fmt.Fprintf(&ctxBuf, "%s: %s\n", m.Role, truncateString(c, 400))
		}
		for _, tu := range m.ToolUse {
			fmt.Fprintf(&ctxBuf, "tool call %s: %s\n", tu.Name, truncateString(string(tu.Input), 200))
		}
		for _, tr := range m.ToolResults {
			fmt.Fprintf(&ctxBuf, "tool result: %s\n", truncateString(tr.Content, 400))
		}
	}
	contextDigest := ctxBuf.String()
	if maxLen := 3000; len(contextDigest) > maxLen {
		contextDigest = contextDigest[len(contextDigest)-maxLen:]
	}

	var actBuf strings.Builder
	for _, tc := range blockedCalls {
		fmt.Fprintf(&actBuf, "- %s with arguments %s\n", tc.Name, string(tc.Input))
	}

	system := "You write exactly one short sentence (max 20 words) describing what an assistant is about to do, for a human who must approve it. Use the human-readable name or number of any record (resolve IDs from the context); never show a raw ID. Output only the sentence."
	userMsg := fmt.Sprintf("Conversation context:\n%s\n\nPending action(s) awaiting approval:\n%s\nDescribe what will happen if approved.", contextDigest, actBuf.String())

	resp, err := provider.CompleteWithTools(ctx, &llm.ToolRequest{
		Model:       modelName,
		System:      system,
		Messages:    []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:   80,
		Temperature: 0,
	})
	if err != nil || resp == nil {
		slog.Warn("Failed to summarize blocked action for approval", "error", err, "model", modelName)
		return ""
	}
	return strings.TrimSpace(resp.Content)
}

// runAgentLoop is the core agentic tool loop shared by executeAgent and ContinueRun.
func (s *runnerSvc) runAgentLoop(
	ctx context.Context,
	run *sqlc.AgentRun,
	accountID string,
	identity *types.Identity,
	systemPrompt string,
	modelChain []string,
	toolDefs []llm.ToolDefinition,
	temperature float64,
	messages []llm.Message,
	seq *int,
	runCtx *domain.HandlerRunContext,
	spendingCapCents *int64,
	currentMonthSpendCents int64,
) (*domain.RunResult, error) {
	if len(modelChain) == 0 {
		return nil, fmt.Errorf("no model chain provided")
	}
	// Active model within the tier chain; on a provider outage we advance to the next (cross-provider) model and stay there for the rest of the run.
	modelIdx := 0
	modelName := modelChain[modelIdx]
	providerName := inferProvider(modelName)

	var totalInputTokens, totalOutputTokens int
	var runSpendCents int64
	startTime := time.Now()
	doomDetector := &doomLoopDetector{}

	retryCfg := (&retry.Config{
		MaxRetries:     3,
		InitialWait:    2 * time.Second,
		MaxWait:        30 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.1,
	}).WithDefaults()

	for range maxToolLoopIterations {
		iterStart := time.Now()

		// Cooperative cancellation: stop before spending another (potentially expensive) model call if the run was cancelled out-of-band.
		if s.runCancelled(ctx, run.ID) {
			return s.cancelledResult(ctx, run, accountID, seq, runCtx, totalInputTokens, totalOutputTokens, providerName, modelName), nil
		}

		modelName = modelChain[modelIdx]
		providerName = inferProvider(modelName)
		provider := s.llmProviders[providerName]

		// Proactive compaction: check estimated context usage BEFORE the LLM call.
		if llm.NeedsProactiveCompaction(systemPrompt, messages, toolDefs, modelName) {
			messages = llm.CopyMessages(messages)
			freed := pruneOldToolResults(messages)
			slog.Info("Proactive compaction: pruned old tool results",
				"run_id", run.ID, "tokens_freed", freed)

			if llm.NeedsProactiveCompaction(systemPrompt, messages, toolDefs, modelName) {
				compactProvider, compactModel := s.compactionTarget(provider, modelName)
				summary, compactErr := compactMessages(ctx, compactProvider, compactModel, systemPrompt, messages)
				if compactErr != nil {
					slog.Error("Proactive compaction failed, falling back to truncation",
						"run_id", run.ID, "error", compactErr)
				} else {
					lastMsg := messages[len(messages)-1]
					messages = []llm.Message{*summary, lastMsg}

					compactMeta, _ := json.Marshal(map[string]any{
						"tokens_freed":    freed,
						"compaction_type": "proactive_llm_summary",
					})
					s.emitEvent(ctx, run.ID, accountID, seq, "compaction", "Context compacted proactively", nil, nil, nil, compactMeta)
				}
			} else if freed > 0 {
				compactMeta, _ := json.Marshal(map[string]any{
					"tokens_freed":    freed,
					"compaction_type": "proactive_prune",
				})
				s.emitEvent(ctx, run.ID, accountID, seq, "compaction", "Context pruned proactively", nil, nil, nil, compactMeta)
			}
		}

		// ---- LLM call with cross-provider failover across the tier chain ----
		var resp *llm.ToolResponse
		var llmErr error
		for {
			modelName = modelChain[modelIdx]
			providerName = inferProvider(modelName)
			provider = s.llmProviders[providerName]
			if provider == nil {
				llmErr = fmt.Errorf("LLM provider %q not configured", providerName)
			} else {
				truncatedMessages := llm.TruncateMessages(systemPrompt, messages, toolDefs, modelName)
				llmReq := &llm.ToolRequest{
					Model:       modelName,
					System:      systemPrompt,
					Messages:    truncatedMessages,
					Tools:       toolDefs,
					MaxTokens:   4096,
					Temperature: temperature,
				}
				// Chat runs stream their reasoning into the conversation's live thinking panel, so turn on the provider's native reasoning. Only the native Anthropic path returns reasoning (adaptive/extended thinking with signatures); OpenAI-compatible /chat/completions doesn't surface chain-of-thought, and reasoning_effort 400s non-reasoning models there — so we leave ReasoningEffort unset and let non-Anthropic fallbacks degrade to no live reasoning.
				if isChatRun(run) {
					llmReq.EnableReasoning = true
				}
				resp, llmErr = s.completeWithRetry(ctx, run.ID, accountID, seq, provider, llmReq, retryCfg)
			}
			if llmErr == nil {
				break
			}
			var ge *llm.GatewayError
			isGatewayErr := errors.As(llmErr, &ge)
			// A billing/spend-limit rejection is account-wide (every model bills the same Stripe customer), so stop the run cleanly with the cap message instead of failing over across the whole chain.
			if isGatewayErr && ge.IsBillingLimitError() {
				return s.capStoppedResult(ctx, run, accountID, seq, runCtx,
					"Spending limit reached: the account's usage limit has been hit. Raise or remove the cap to continue.",
					totalInputTokens, totalOutputTokens, providerName, modelName), nil
			}
			// Fail over only on a provider-down/overloaded error that survived retries; a request-level error (bad input, context length) would fail identically on every model.
			if isGatewayErr && ge.Retryable && modelIdx+1 < len(modelChain) {
				failMeta, _ := json.Marshal(map[string]any{"from_model": modelName, "to_model": modelChain[modelIdx+1], "status": ge.StatusCode})
				s.emitEvent(ctx, run.ID, accountID, seq, "model_failover", fmt.Sprintf("Provider error on %s; failing over to %s", modelName, modelChain[modelIdx+1]), nil, nil, nil, failMeta)
				modelIdx++
				continue
			}
			break
		}
		if llmErr != nil {
			s.emitEvent(ctx, run.ID, accountID, seq, "error", "LLM call failed", new(llmErr.Error()), nil, nil, nil)
			// A retryable gateway error here means every model in the chain failed over and the whole chain is momentarily unavailable (per-call failover already absorbed transient single-model blips). If this turn produced no side effects yet, the run can be transparently re-attempted later; signal that to the caller via a typed error. With side effects already applied, a blind re-run could duplicate them, so fall through to a normal terminal failure.
			var ge *llm.GatewayError
			if errors.As(llmErr, &ge) && ge.Retryable && len(runCtx.Actions) == 0 {
				return nil, &retryableRunError{cause: llmErr}
			}
			return nil, fmt.Errorf("LLM call failed: %w", llmErr)
		}

		totalInputTokens += resp.InputTokens
		totalOutputTokens += resp.OutputTokens

		// Accumulate estimated cost for this iteration and check spending cap.
		iterCostCents := llm.EstimateTokenCostCents(resp.InputTokens, resp.OutputTokens, modelName)
		runSpendCents += iterCostCents
		if spendingCapCents != nil && (currentMonthSpendCents+runSpendCents) >= *spendingCapCents {
			capDollars := float64(*spendingCapCents) / 100.0
			capMsg := fmt.Sprintf("Monthly agent spending cap of $%.2f reached during run", capDollars)
			return s.capStoppedResult(ctx, run, accountID, seq, runCtx, capMsg,
				totalInputTokens, totalOutputTokens, providerName, modelName), nil
		}

		if resp.StopReason != "tool_use" || len(resp.ToolCalls) == 0 {
			// Record the final turn's reasoning before the answer (it already streamed live), so the timeline reads reasoning → answer.
			s.emitThinkingStep(ctx, run.ID, accountID, seq, resp, iterStart)

			// Emit assistant_message event (the user-facing answer is the final turn's text)
			if resp.Content != "" {
				s.emitEvent(ctx, run.ID, accountID, seq, "assistant_message", "Assistant response", new(resp.Content), nil, nil, nil)
			}

			// Emit completion event
			durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
			completionMeta, _ := json.Marshal(map[string]any{
				"actionsExecuted":   len(runCtx.Actions),
				"totalDurationMs":   durationMs,
				"totalInputTokens":  totalInputTokens,
				"totalOutputTokens": totalOutputTokens,
			})
			s.emitEvent(ctx, run.ID, accountID, seq, "completion", "Run completed", nil, &durationMs, nil, completionMeta)

			// Final response
			outputJSON, _ := json.Marshal(map[string]string{"response": resp.Content})
			return &domain.RunResult{
				Output:       outputJSON,
				Actions:      runCtx.Actions,
				Artifacts:    runCtx.Artifacts,
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				LLMProvider:  providerName,
				LLMModel:     modelName,
			}, nil
		}

		// A tool-using turn: record its reasoning on the timeline (already streamed live), then replay it (with signatures) plus any interim text and the tool calls into the next turn.
		s.emitThinkingStep(ctx, run.ID, accountID, seq, resp, iterStart)

		// Process tool calls
		assistantMsg := llm.Message{
			Role:     "assistant",
			Content:  resp.Content,
			Thinking: resp.Thinking,
		}
		for _, tc := range resp.ToolCalls {
			assistantMsg.ToolUse = append(assistantMsg.ToolUse, llm.ToolUseBlock(tc))
		}
		messages = append(messages, assistantMsg)

		// The set of tools currently exposed to this run. A tool is permitted to execute only if it is in this set, so a tool the agent was never given (hallucinated, or an endpoint-tool not revealed/granted) cannot run even if the model names it. The set grows as search reveals endpoint-tools.
		exposed := make(map[string]bool, len(toolDefs))
		for _, td := range toolDefs {
			exposed[td.Name] = true
		}

		toolResultMsg := llm.Message{Role: "user"}
		toolsBlocked := false
		for _, tc := range resp.ToolCalls {
			// Cooperative cancellation between tool calls: a cancel that lands mid-batch stops runaway tool use at the next tool boundary instead of draining the whole batch. Checked before the tool_call event is emitted so no half-recorded call is left in the timeline.
			if s.runCancelled(ctx, run.ID) {
				return s.cancelledResult(ctx, run, accountID, seq, runCtx, totalInputTokens, totalOutputTokens, providerName, modelName), nil
			}

			// Emit tool_call event
			toolCallMeta, _ := json.Marshal(map[string]any{"tool_use_id": tc.ID, "tool_name": tc.Name, "input": tc.Input})
			s.emitEvent(ctx, run.ID, accountID, seq, "tool_call", tc.Name, nil, nil, nil, toolCallMeta)

			// Enforcement: refuse any tool not currently exposed to this agent.
			if !exposed[tc.Name] {
				deniedMsg := fmt.Sprintf("Tool %q is not available to this agent.", tc.Name)
				slog.Warn("Tool execution denied — not granted to agent",
					"run_id", run.ID, "tool", tc.Name)
				deniedMeta, _ := json.Marshal(map[string]any{"tool_use_id": tc.ID, "tool_name": tc.Name, "denied": true})
				s.emitEvent(ctx, run.ID, accountID, seq, "tool_denied", tc.Name+" denied", new(deniedMsg), nil, nil, deniedMeta)
				toolResultMsg.ToolResults = append(toolResultMsg.ToolResults, llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   deniedMsg,
					IsError:   true,
				})
				continue
			}

			// Rejection: the human denied this review-gated tool on resume. Answer the call with a denial and keep going (no pause) — the model sees the result and proceeds without the tool. Checked before the approval guard so a denied tool never re-pauses the run. Matches a slug-level denial or a per-call denial keyed on (slug+input).
			if runCtx.RejectedSlugs[tc.Name] || runCtx.RejectedKeys[toolCallApprovalKey(tc.Name, tc.Input)] {
				deniedMsg := "[DENIED] The user denied this tool call. Do not retry it; continue without it."
				slog.Info("Tool execution denied by user", "run_id", run.ID, "tool", tc.Name)

				deniedMeta, _ := json.Marshal(map[string]any{"tool_use_id": tc.ID, "tool_name": tc.Name, "input": tc.Input, "denied": true})
				s.emitEvent(ctx, run.ID, accountID, seq, "tool_denied", tc.Name+" denied", new(deniedMsg), nil, nil, deniedMeta)

				toolResultMsg.ToolResults = append(toolResultMsg.ToolResults, llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   deniedMsg,
					IsError:   true,
				})
				continue
			}

			// Guard: block tools that require human approval unless explicitly approved — by slug (approve-all / slug list) or by this specific call's (slug+input) key.
			if runCtx.RequireReviewBySlug[tc.Name] && !runCtx.AlwaysAllowedSlugs[tc.Name] && !runCtx.OneTimeApprovedSlugs[tc.Name] && !runCtx.OneTimeApprovedKeys[toolCallApprovalKey(tc.Name, tc.Input)] {
				blockedMsg := "[REQUIRES APPROVAL] This tool requires human approval before it can be executed. The run will pause for review."
				slog.Info("Tool execution blocked — requires human approval",
					"run_id", run.ID, "tool", tc.Name)

				blockedMeta, _ := json.Marshal(map[string]any{"tool_use_id": tc.ID, "tool_name": tc.Name, "input": tc.Input, "blocked": true})
				s.emitEvent(ctx, run.ID, accountID, seq, "tool_blocked", tc.Name+" blocked", new(blockedMsg), nil, nil, blockedMeta)

				toolResultMsg.ToolResults = append(toolResultMsg.ToolResults, llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   blockedMsg,
				})

				// Record the blocked action so the input is persisted for review
				runCtx.Actions = append(runCtx.Actions, domain.PendingAction{
					ToolSlug:       tc.Name,
					Label:          tc.Name,
					Input:          tc.Input,
					RequiresReview: true,
				})

				toolsBlocked = true
				continue
			}

			// Doom loop detection: check if this tool+input has been called identically too many times.
			if doomDetector.Record(tc.Name, tc.Input) {
				doomMsg := fmt.Sprintf("Called %s 3 times with identical input. Try a different approach or parameters.", tc.Name)
				slog.Warn("Doom loop detected",
					"run_id", run.ID, "tool", tc.Name)

				doomMeta, _ := json.Marshal(map[string]any{"tool_use_id": tc.ID, "tool_name": tc.Name, "doom_loop": true})
				s.emitEvent(ctx, run.ID, accountID, seq, "doom_loop_detected", "Doom loop detected for "+tc.Name, new(doomMsg), nil, nil, doomMeta)

				toolResultMsg.ToolResults = append(toolResultMsg.ToolResults, llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   doomMsg,
					IsError:   true,
				})
				continue
			}

			toolStart := time.Now()
			result, err := s.handleToolCall(ctx, tc, runCtx)
			toolDurationMs := safeconv.Int64ToInt32(time.Since(toolStart).Milliseconds())

			// Consume one-time approval so the tool requires re-approval on next invocation (both the slug-level grant and this call's per-call key).
			delete(runCtx.OneTimeApprovedSlugs, tc.Name)
			delete(runCtx.OneTimeApprovedKeys, toolCallApprovalKey(tc.Name, tc.Input))

			if err != nil {
				// Emit tool_result event with error
				toolResultMeta, _ := json.Marshal(map[string]any{"tool_use_id": tc.ID, "is_error": true, "full_result": err.Error()})
				truncatedResult := truncateString(err.Error(), 500)
				s.emitEvent(ctx, run.ID, accountID, seq, "tool_result", tc.Name+" result", &truncatedResult, &toolDurationMs, nil, toolResultMeta)

				toolResultMsg.ToolResults = append(toolResultMsg.ToolResults, llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   fmt.Sprintf("Error: %s", err.Error()),
					IsError:   true,
				})
			} else {
				// Truncate tool output, tracking whether truncation occurred.
				truncResult := llm.TruncateToolOutputResult(result, tc.Name)

				// Build event metadata — store full result inline only if not truncated.
				var toolResultMeta json.RawMessage
				if truncResult.WasTruncated {
					// Store full result as an artifact instead of inlining in event metadata.
					artifactID, genErr := id.GenID(id.AgentArtifactIDPrefix, nil)
					if genErr != nil {
						artifactID = "unknown"
					}
					runCtx.Artifacts = append(runCtx.Artifacts, domain.PendingArtifact{
						ActionIndex:  len(runCtx.Actions),
						ArtifactType: "tool_output",
						Name:         fmt.Sprintf("%s_full_output", tc.Name),
						Content:      result,
						MimeType:     "text/plain",
					})
					toolResultMeta, _ = json.Marshal(map[string]any{
						"tool_use_id":   tc.ID,
						"is_error":      false,
						"artifact_id":   artifactID,
						"full_length":   truncResult.FullLength,
						"was_truncated": true,
					})
				} else {
					toolResultMeta, _ = json.Marshal(map[string]any{"tool_use_id": tc.ID, "is_error": false, "full_result": result})
				}

				truncatedEventContent := truncateString(truncResult.Content, 500)
				s.emitEvent(ctx, run.ID, accountID, seq, "tool_result", tc.Name+" result", &truncatedEventContent, &toolDurationMs, nil, toolResultMeta)

				toolResultMsg.ToolResults = append(toolResultMsg.ToolResults, llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   truncResult.Content,
				})

				// Record action for every successful tool call
				outputJSON, _ := json.Marshal(map[string]string{"result": truncResult.Content})
				runCtx.Actions = append(runCtx.Actions, domain.PendingAction{
					ToolSlug:       tc.Name,
					Label:          tc.Name,
					Input:          tc.Input,
					Output:         outputJSON,
					RequiresReview: runCtx.RequireReviewBySlug[tc.Name],
				})
			}
		}
		messages = append(messages, toolResultMsg)

		// Reveal: make endpoint-tools surfaced by search_api_tools this turn callable on the next turn by adding them to the live tool list. Only reveal tools the agent is actually granted (defense in depth — the search handler already scopes to the grant).
		for slug := range runCtx.RevealedToolSlugs {
			if exposed[slug] || !runCtx.AllowedEndpointToolSlugs[slug] {
				continue
			}
			d, ok := agents.LookupEndpointTool(slug)
			if !ok {
				continue
			}
			toolDefs = append(toolDefs, llm.ToolDefinition{
				Name:        d.Slug,
				Description: d.Description,
				InputSchema: json.RawMessage(d.InputSchema),
			})
			exposed[slug] = true
		}

		// Context compaction: if we're approaching the model's limit, prune then summarize.
		if needsCompaction(resp.InputTokens, modelName) {
			// Deep copy messages for mutation during pruning.
			messages = llm.CopyMessages(messages)
			freed := pruneOldToolResults(messages)
			slog.Info("Context compaction: pruned old tool results",
				"run_id", run.ID, "tokens_freed", freed, "input_tokens", resp.InputTokens)

			// If pruning wasn't enough, trigger LLM-based summarization.
			if llm.EstimateAllMessages(messages) >= (resp.InputTokens - compactionBuffer) {
				compactProvider, compactModel := s.compactionTarget(provider, modelName)
				summary, compactErr := compactMessages(ctx, compactProvider, compactModel, systemPrompt, messages)
				if compactErr != nil {
					slog.Error("Context compaction failed, falling back to truncation",
						"run_id", run.ID, "error", compactErr)
				} else {
					// Replace all messages with summary + most recent user message.
					lastMsg := messages[len(messages)-1]
					messages = []llm.Message{*summary, lastMsg}

					compactMeta, _ := json.Marshal(map[string]any{
						"input_tokens_before": resp.InputTokens,
						"tokens_freed":        freed,
						"compaction_type":     "llm_summary",
					})
					s.emitEvent(ctx, run.ID, accountID, seq, "compaction", "Context compacted via summarization", nil, nil, nil, compactMeta)
				}
			} else if freed > 0 {
				compactMeta, _ := json.Marshal(map[string]any{
					"input_tokens_before": resp.InputTokens,
					"tokens_freed":        freed,
					"compaction_type":     "prune",
				})
				s.emitEvent(ctx, run.ID, accountID, seq, "compaction", "Context compacted via pruning", nil, nil, nil, compactMeta)
			}
		}

		// Check if any tools were blocked for approval — pause after this turn
		if toolsBlocked {
			blockedTools := make([]string, 0)
			blockedCalls := make([]llm.ToolCall, 0)
			for _, tc := range resp.ToolCalls {
				key := toolCallApprovalKey(tc.Name, tc.Input)
				if runCtx.RequireReviewBySlug[tc.Name] && !runCtx.AlwaysAllowedSlugs[tc.Name] && !runCtx.OneTimeApprovedSlugs[tc.Name] && !runCtx.OneTimeApprovedKeys[key] && !runCtx.RejectedSlugs[tc.Name] && !runCtx.RejectedKeys[key] {
					blockedTools = append(blockedTools, tc.Name)
					blockedCalls = append(blockedCalls, tc)
				}
			}
			// Describe the pending action for the human approver in plain language. Prefer the agent's own narration of this turn (the assistant text alongside the tool call); but with native reasoning on, a tool turn usually carries no user-facing text, so fall back to a one-line summary generated from the transcript — which resolves raw IDs to the names the agent already looked up (e.g. "Remove the note from Pacific Coast Distributors."). The summary is also surfaced in the approval event metadata so the run console can show it.
			summary := strings.TrimSpace(resp.Content)
			if summary == "" {
				summary = s.summarizeBlockedAction(ctx, provider, modelName, messages, blockedCalls)
			}
			approvalMeta, _ := json.Marshal(map[string]any{
				"blocked_tools":     blockedTools,
				"summary":           summary,
				"totalInputTokens":  totalInputTokens,
				"totalOutputTokens": totalOutputTokens,
			})
			approvalMsg := fmt.Sprintf("I need your approval before I can run: %s.", strings.Join(blockedTools, ", "))
			if summary != "" {
				approvalMsg = summary
			}
			s.emitEvent(ctx, run.ID, accountID, seq, "awaiting_approval", "Waiting for tool approval", new(approvalMsg), nil, nil, approvalMeta)

			outputJSON, _ := json.Marshal(map[string]string{"response": approvalMsg})
			return &domain.RunResult{
				Output:           outputJSON,
				Actions:          runCtx.Actions,
				Artifacts:        runCtx.Artifacts,
				InputTokens:      totalInputTokens,
				OutputTokens:     totalOutputTokens,
				LLMProvider:      providerName,
				LLMModel:         modelName,
				AwaitingApproval: true,
			}, nil
		}

	}

	// Max iterations reached - emit completion event
	durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
	completionMeta, _ := json.Marshal(map[string]any{
		"actionsExecuted":   len(runCtx.Actions),
		"totalDurationMs":   durationMs,
		"maxIterationsHit":  true,
		"totalInputTokens":  totalInputTokens,
		"totalOutputTokens": totalOutputTokens,
	})
	s.emitEvent(ctx, run.ID, accountID, seq, "completion", "Run completed (max iterations)", nil, &durationMs, nil, completionMeta)

	outputJSON, _ := json.Marshal(map[string]string{"response": "Max tool loop iterations reached"})
	return &domain.RunResult{
		Output:       outputJSON,
		Actions:      runCtx.Actions,
		Artifacts:    runCtx.Artifacts,
		InputTokens:  totalInputTokens,
		OutputTokens: totalOutputTokens,
		LLMProvider:  providerName,
		LLMModel:     modelName,
	}, nil
}

// runResumedLoop finishes a resumed turn: it first executes any approved-but-blocked tool calls directly (so an approval actually performs its write instead of depending on the model to re-issue the call — see resumeApprovedBlockedCalls) and then runs the agent loop over the reconstructed transcript. ContinueRun's resume tail is exactly this call; keeping the two steps together in one method means the "execute on approval, then continue" contract is exercised end to end in tests rather than only wired inline.
func (s *runnerSvc) runResumedLoop(ctx context.Context, run *sqlc.AgentRun, accountID string, identity *types.Identity, systemPrompt string, modelChain []string, toolDefs []llm.ToolDefinition, temperature float64, messages []llm.Message, seq *int, runCtx *domain.HandlerRunContext, events []sqlc.AgentRunEvent, spendingCapCents *int64, currentMonthSpendCents int64) (*domain.RunResult, error) {
	s.resumeApprovedBlockedCalls(ctx, run, accountID, seq, runCtx, messages, events)
	return s.runAgentLoop(ctx, run, accountID, identity, systemPrompt, modelChain, toolDefs, temperature, messages, seq, runCtx, spendingCapCents, currentMonthSpendCents)
}

// approvesReviewedTool reports whether a ContinueRun resume approves a given pending review-gated tool — the single authority for what a resume lets through. A per-tool approval names the slugs (and only those pass; approveAllPending is ignored when slugs are present); an "Approve all" sets approveAllPending with no slugs; everything else — most importantly a typed-message continuation or a retry, which both arrive with no slugs and approveAllPending=false — approves NOTHING, so the tool stays blocked and re-prompts.
// Pure by design: this is the security-critical rule, kept unit-testable without standing up a full run.
func approvesReviewedTool(toolSlug string, approvedToolSlugs []string, approveAllPending bool) bool {
	if len(approvedToolSlugs) > 0 {
		return slices.Contains(approvedToolSlugs, toolSlug)
	}
	return approveAllPending
}

// toolCallApprovalKey identifies ONE specific tool call for per-call approval: its slug plus canonicalized
// input. Two calls of the same slug with different inputs get different keys, so one can be approved without
// the other — which slug-level approval (approvesReviewedTool) cannot express. Canonicalizing (decode then
// re-encode, which sorts object keys) keeps the key stable across the pause/resume boundary, where the model
// re-emits the call and its JSON may differ in key order or whitespace from the input recorded at block time.
func toolCallApprovalKey(slug string, input json.RawMessage) string {
	return slug + "\x00" + canonicalizeJSONInput(input)
}

// canonicalizeJSONInput returns a stable string form of a JSON value (Go's encoder sorts object keys). On any
// decode error it falls back to the trimmed raw bytes so distinct raw inputs still map to distinct keys.
func canonicalizeJSONInput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return strings.TrimSpace(string(raw))
	}
	b, err := json.Marshal(v)
	if err != nil {
		return strings.TrimSpace(string(raw))
	}
	return string(b)
}

// blockedCallKeysByID maps each blocked call's tool_use_id to its approval key, read from this run's
// tool_blocked events (which persist tool_use_id, tool_name, and input). The per-call approve/reject lists
// name tool_use_ids — the ids the frontend holds from the tool_blocked steps — and this resolves them to the
// (slug+input) keys the runner matches a retried call against, so no id needs to survive on the action row.
func blockedCallKeysByID(events []sqlc.AgentRunEvent) map[string]string {
	out := make(map[string]string)
	for _, e := range events {
		if e.StepType != "tool_blocked" || e.Metadata == nil {
			continue
		}
		var meta struct {
			ToolUseID string          `json:"tool_use_id"`
			ToolName  string          `json:"tool_name"`
			Input     json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(e.Metadata, &meta); err != nil || meta.ToolUseID == "" {
			continue
		}
		out[meta.ToolUseID] = toolCallApprovalKey(meta.ToolName, meta.Input)
	}
	return out
}

func (s *runnerSvc) ContinueRun(ctx context.Context, runID, accountID, message string, approvedToolSlugs []string, approveAllPending bool, rejectedToolSlugs []string, approvedToolCallIDs, rejectedToolCallIDs []string, actorID, actorType, actorName, replyToMessageID string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runnerTracer, "service.runner.continue_run")
	defer span.End()

	// An approval resume is itself a conversation turn: it fulfills the original chat request, so the agent's continuation must reach the thread. Detect it by actions still pending review on this run — "approve all" sends no approvedToolSlugs, so the slug count alone misses the common single-tool case.
	approvalResume := false
	if pending, perr := s.repos.NewAgentActionRepo().ListByRun(ctx, runID); perr == nil {
		for _, a := range pending {
			if a.StatusCode == domain.ActionStatusPendingReview {
				approvalResume = true
				break
			}
		}
	}

	// This turn came from the conversation when it's a reply in the thread (ReplyToMessageID set) or an approval that unblocks the original chat request — both should post the result back. A free-text message typed in the agent-run console has none of these: it's a private fork of the run, so its reply stays on the run and never reaches the thread.
	fromConversation := replyToMessageID != "" || len(approvedToolSlugs) > 0 || approvalResume
	ctx = withChatTurnFromConversation(ctx, fromConversation)

	startTime := time.Now()
	runRepo := s.repos.NewAgentRunRepo()
	configRepo := s.repos.NewAgentConfigRepo()
	defRepo := s.repos.NewAgentDefinitionRepo()
	eventRepo := s.repos.NewAgentRunEventRepo()

	run, runErr := runRepo.GetByID(ctx, runID)
	if runErr != nil {
		return apierror.NewInternalError(runErr, fmt.Sprintf("failed to load run %s", runID))
	}

	// Validate status — the gRPC handler already transitions awaiting_input → running
	if run.StatusCode != domain.RunStatusRunning {
		return apierror.NewInvariantViolationError(fmt.Sprintf("run %s is not in running state (status: %s)", runID, run.StatusCode))
	}

	// Mirror ExecuteRun: tell the conversation this run is live again so chat subscribers re-subscribe to its run:<id> step stream and render this turn's interim work inline. A chat continue-run reuses one run id across turns and rests at awaiting_input between them, so without this the bubble never re-mounts for a reply turn. Only for turns that post back to the thread — a private console fork (!fromConversation) must not surface its activity in the conversation.
	if fromConversation {
		s.emitChatRunStarted(ctx, accountID, run)
	}

	// Resolve which pending-review tools this resume approves, and post the "who approved what" notice — BEFORE beginChatStream below. Both the notice and the streaming reply are kind=agent messages whose render order is their conversation sequence (assigned when notification-service processes them in enqueue order), so the human decision must be enqueued first or it lands beneath the agent's reply.
	// Auto-approving a blocked, review-gated tool must be an explicit human decision. A per-tool approval names slugs; an "Approve all" sets approveAllPending. A typed-message continuation (the user replies instead of approving) or a retry does NEITHER — it must not silently let a blocked tool through, so it approves nothing and the tool re-prompts when next called. Without this guard a continuation's empty approvedToolSlugs read as "approve everything pending", running gated tools unreviewed.
	// Per-call decisions target one blocked call by its tool_use_id. Resolve those ids to (slug+input) keys
	// via the run's tool_blocked events, so two same-slug calls with different inputs can be decided apart —
	// which the slug lists below cannot express (a slug approval approves EVERY pending call of that slug).
	approvedKeys := make(map[string]bool)
	rejectedKeys := make(map[string]bool)
	if len(approvedToolCallIDs) > 0 || len(rejectedToolCallIDs) > 0 {
		if blockedEvents, evErr := eventRepo.ListByRunID(ctx, runID); evErr == nil {
			callIDToKey := blockedCallKeysByID(blockedEvents)
			for _, id := range approvedToolCallIDs {
				if k, ok := callIDToKey[id]; ok {
					approvedKeys[k] = true
				}
			}
			for _, id := range rejectedToolCallIDs {
				if k, ok := callIDToKey[id]; ok {
					rejectedKeys[k] = true
				}
			}
		}
	}

	// Auto-approving a blocked, review-gated tool must be an explicit human decision. A slug approval names slugs; a per-call approval names call ids (resolved to keys above); an "Approve all" sets approveAllPending. A typed-message continuation (the user replies instead of approving) or a retry does NONE of these — it must not silently let a blocked tool through, so it approves nothing and the tool re-prompts when next called. Without this guard a continuation's empty lists read as "approve everything pending", running gated tools unreviewed.
	// A slug/approve-all decision goes into oneTimeApproved (keyed by slug, so every pending call of that slug passes); a per-call decision goes into oneTimeApprovedKeys (keyed by slug+input, so only the chosen call passes). The gates check both.
	oneTimeApproved := make(map[string]bool)
	oneTimeApprovedKeys := make(map[string]bool)
	decidedSlugs := make(map[string]bool) // slugs with any decision — re-exposed to the tool list on resume
	approvedInOrder := make([]string, 0)  // distinct approved slugs, in action order — for the approval notice
	approvalRepo := s.repos.NewAgentActionRepo()
	if pendingActions, listErr := approvalRepo.ListByRun(ctx, runID); listErr == nil {
		for _, a := range pendingActions {
			if a.StatusCode != domain.ActionStatusPendingReview {
				continue
			}
			slugApproved := approvesReviewedTool(a.ToolSlug, approvedToolSlugs, approveAllPending)
			key := toolCallApprovalKey(a.ToolSlug, a.Input)
			keyApproved := approvedKeys[key]
			if !slugApproved && !keyApproved {
				continue
			}
			if !decidedSlugs[a.ToolSlug] {
				approvedInOrder = append(approvedInOrder, a.ToolSlug)
			}
			if slugApproved {
				oneTimeApproved[a.ToolSlug] = true
			}
			if keyApproved {
				oneTimeApprovedKeys[key] = true
			}
			decidedSlugs[a.ToolSlug] = true
			// Record the decision with who made it for the audit trail (reviewed_by/_at/actor).
			_ = approvalRepo.MarkReviewed(ctx, sqlc.MarkAgentActionReviewedParams{
				ID:                  a.ID,
				StatusCode:          domain.ActionStatusApproved,
				ReviewedBy:          agentdb.PgText(actorID),
				ReviewedByActorType: agentdb.PgText(actorType),
				ReviewedByActorName: agentdb.PgText(actorName),
			})
		}
	}

	// Resolve which pending-review tools this resume denies. Unlike a non-approval (which leaves a tool pending so it re-prompts on the next call), an explicit rejection is a terminal human decision: mark the action rejected and arm a one-time denial so the runner answers the tool's retry with a "denied by user" result and the run continues without it — it is never paused on again. Slug-level and per-call rejection mirror the approval split above.
	oneTimeRejected := make(map[string]bool)
	rejectedKeysActive := make(map[string]bool)
	rejectedInOrder := make([]string, 0)
	if len(rejectedToolSlugs) > 0 || len(rejectedKeys) > 0 {
		if pendingActions, listErr := approvalRepo.ListByRun(ctx, runID); listErr == nil {
			for _, a := range pendingActions {
				if a.StatusCode != domain.ActionStatusPendingReview {
					continue
				}
				slugRejected := slices.Contains(rejectedToolSlugs, a.ToolSlug)
				key := toolCallApprovalKey(a.ToolSlug, a.Input)
				keyRejected := rejectedKeys[key]
				if !slugRejected && !keyRejected {
					continue
				}
				if !slices.Contains(rejectedInOrder, a.ToolSlug) {
					rejectedInOrder = append(rejectedInOrder, a.ToolSlug)
				}
				if slugRejected {
					oneTimeRejected[a.ToolSlug] = true
				}
				if keyRejected {
					rejectedKeysActive[key] = true
				}
				decidedSlugs[a.ToolSlug] = true // re-expose the tool so a denied retry reaches the denial guard
				_ = approvalRepo.MarkReviewed(ctx, sqlc.MarkAgentActionReviewedParams{
					ID:                  a.ID,
					StatusCode:          domain.ActionStatusRejected,
					ReviewedBy:          agentdb.PgText(actorID),
					ReviewedByActorType: agentdb.PgText(actorType),
					ReviewedByActorName: agentdb.PgText(actorName),
				})
			}
		}
	}

	// Record who approved/denied what as a non-bubble timeline event in the conversation (chat-linked runs only).
	if fromConversation && len(approvedInOrder) > 0 {
		s.writeChatApprovalNotice(ctx, run, runID, accountID, actorName, approvedInOrder)
	}
	if fromConversation && len(rejectedInOrder) > 0 {
		s.writeChatRejectionNotice(ctx, run, runID, accountID, actorName, rejectedInOrder)
	}

	// A free-text continuation typed into the agent-run console is a private fork: its turns enter this run's transcript but never the conversation. Mark the (chat-linked) run as diverged so a later reply to the agent's message in the conversation starts a fresh run rather than inheriting this fork's hidden context. Sticky and idempotent — only the first fork needs to write.
	if !fromConversation && run.ConversationID.Valid && run.ConversationID.String != "" && !run.DivergedFromConversation {
		if markErr := runRepo.MarkDivergedFromConversation(ctx, runID); markErr != nil {
			return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to mark run diverged: %s", markErr.Error()))
		}
	}

	// Load config
	configID := ""
	if run.AgentConfigID.Valid {
		configID = run.AgentConfigID.String
	}
	if configID == "" {
		return apierror.NewInvariantViolationError(fmt.Sprintf("run %s has no agent config ID", runID))
	}
	config, cfgErr := configRepo.GetByID(ctx, configID)
	if cfgErr != nil {
		return apierror.NewInternalError(cfgErr, fmt.Sprintf("failed to load config %s", configID))
	}

	def, defErr := defRepo.GetByID(ctx, config.AgentDefinitionID)
	if defErr != nil {
		return apierror.NewInternalError(defErr, "failed to load definition")
	}

	// Each conversation turn gets its own streaming reply bubble: post it empty/streaming now (threaded under the user's reply that drives this turn) and fill it in as the model produces tokens. Self- gates on the conversation turn, so a console fork is a no-op.
	ctx = s.beginChatStream(ctx, run, runID, accountID, replyToMessageID, def.Name)

	// Parse agent config
	var agentCfg agentConfig
	if def.Config != nil {
		if err := json.Unmarshal(def.Config, &agentCfg); err != nil {
			return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("invalid agent config: %s", err.Error()))
		}
	}

	// Resolve the model chain from the agent's tier (or an explicit model override). The runner tries the primary first and fails over to the next, cross-provider model on a provider outage.
	var modelChain []string
	if agentCfg.Model != "" {
		if !domain.AllowedModels[agentCfg.Model] {
			return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("model %q is not allowed", agentCfg.Model))
		}
		modelChain = []string{agentCfg.Model}
	} else {
		// Tier precedence: an explicit per-agent tier in the config wins; otherwise auto-assign by trigger purpose (chat/manual → high, scheduled/event → balanced background work).
		tier := constants.ModelTier(agentCfg.Tier)
		if !tier.IsValid() {
			tier = tierForTrigger(run.TriggerType)
		}
		modelChain = tier.ModelChain()
	}

	// Resolve agent identity
	roleID := agentdb.StringFromPgText(def.RoleID)
	if roleID == nil || *roleID == "" {
		return s.failRun(ctx, runRepo, runID, startTime, "agent definition has no role_id")
	}
	permissions, err := s.coreClient.GetRolePermissions(ctx, *roleID)
	if err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to resolve agent permissions: %s", err.Error()))
	}
	roleTypeCode := string(constants.RoleTypeAgent)
	agentIdentity := &types.Identity{
		Type:        types.IdentityActorTypeAgent,
		Target:      &types.IdentityTarget{AccountID: accountID},
		AccountMode: constants.AccountModeProduction,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           def.ID,
			Name:         &def.Name,
			AccountID:    &accountID,
			RoleID:       roleID,
			RoleType:     &roleTypeCode,
			Permissions:  permissions,
		},
	}
	ctx = appctx.WithIdentity(ctx, agentIdentity)

	// Resolve billing context and inject Stripe customer ID. This must run after the identity is in context: the billing service resolves the Stripe customer from the caller's identity, so resolving billing earlier would send the gRPC call without identity metadata and fail.
	bc, billingErr := s.resolveBillingContext(ctx, accountID, modelChain[0])
	if billingErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, billingErr.Error())
	}
	ctx = llm.WithStripeCustomerID(ctx, bc.stripeCustomerID)

	// Load linked tools
	toolRepo := s.repos.NewAgentDefinitionToolRepo()
	linkedTools, toolErr := toolRepo.ListByAgentDefinitionID(ctx, def.ID)
	if toolErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to load tools: %s", toolErr.Error()))
	}

	// Resolve the agent's endpoint-tool grant and build the up-front tool list and review map.
	allowedEndpointTools := resolveAllowedEndpointTools(agentCfg.EndpointToolSlugs)
	toolDefs, requireReviewBySlug := s.buildAgentToolDefs(linkedTools, allowedEndpointTools, agentCfg.EndpointToolReview)

	// Resolve temperature
	temperature := 0.0
	if agentCfg.Temperature != nil {
		temperature = *agentCfg.Temperature
	}

	// Load memories and build system prompt
	systemPrompt := agentCfg.SystemPrompt
	memoryRepo := s.repos.NewAgentMemoryRepo()
	memories, memErr := memoryRepo.ListAccountMemories(ctx, accountID, accountID, 50)
	if memErr != nil {
		memories = nil
	}
	if len(memories) > 0 {
		var sb strings.Builder
		sb.WriteString(systemPrompt)
		sb.WriteString("\n\nRelevant memories from previous interactions:\n")
		for _, m := range memories {
			fmt.Fprintf(&sb, "- [%s] %s\n", m.Category, m.Content)
		}
		systemPrompt = sb.String()
	}

	// Prepend cross-cutting guidance (tool discovery; resource links for chat runs), same as the first turn, so a continued run keeps it.
	systemPrompt = s.augmentSystemPrompt(systemPrompt, run, allowedEndpointTools)

	// Get max sequence
	maxSeq, seqErr := eventRepo.GetMaxSequence(ctx, runID)
	if seqErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to get max sequence: %s", seqErr.Error()))
	}
	seq := int(maxSeq) + 1

	// Load all events and reconstruct messages
	events, eventsErr := eventRepo.ListByRunID(ctx, runID)
	if eventsErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to load events: %s", eventsErr.Error()))
	}

	messages := reconstructMessages(events)

	// Append the caller's new user message and emit the event. A retry/resume carries no new message — it just re-attempts the existing transcript — so skip the append in that case.
	if message != "" {
		messages = append(messages, llm.Message{Role: "user", Content: message})
		s.emitEventAs(ctx, run.ID, accountID, &seq, "user_message", "User message", &message, nil, nil, nil, actorID, actorType, actorName, false)
	}

	// There is no per-run "always allow": a tool that should run unsupervised belongs in the agent definition, not a bypass of its own require-approval gate. Every review-required tool re-prompts on each call; only the one-time approvals above let a specific blocked call through. AlwaysAllowedSlugs stays in the run context (the guard reads it) but is always empty.
	alwaysAllowed := map[string]bool{}

	// Create run context
	runCtx := &domain.HandlerRunContext{
		AccountID:                accountID,
		RunID:                    run.ID,
		Definition:               def,
		Config:                   config,
		Repos:                    s.repos,
		CoreClient:               s.coreClient,
		GatewayClient:            s.gatewayClient,
		NotificationClient:       s.notificationClient,
		ConversationID:           ptrutil.Deref(agentdb.StringFromPgText(run.ConversationID)),
		Identity:                 agentIdentity,
		RequireReviewBySlug:      requireReviewBySlug,
		AlwaysAllowedSlugs:       alwaysAllowed,
		OneTimeApprovedSlugs:     oneTimeApproved,
		RejectedSlugs:            oneTimeRejected,
		OneTimeApprovedKeys:      oneTimeApprovedKeys,
		RejectedKeys:             rejectedKeysActive,
		AllowedEndpointToolSlugs: allowedEndpointTools,
		RevealedToolSlugs:        make(map[string]bool),
	}

	// On resume the per-turn tool-exposure state is gone, so rebuild the live endpoint-tool list before the loop runs.
	exposedOnResume := make(map[string]bool, len(toolDefs))
	for _, td := range toolDefs {
		exposedOnResume[td.Name] = true
	}
	// exposeEndpointTool adds a granted endpoint-tool to the live tool list unless it is already present.
	exposeEndpointTool := func(slug string) {
		if exposedOnResume[slug] {
			return
		}
		d, ok := agents.LookupEndpointTool(slug)
		if !ok {
			return
		}
		toolDefs = append(toolDefs, llm.ToolDefinition{
			Name:        d.Slug,
			Description: d.Description,
			InputSchema: json.RawMessage(d.InputSchema),
		})
		exposedOnResume[slug] = true
	}

	// Re-expose endpoint-tools the human approved or rejected one-time — the model is about to retry them and execution enforcement would otherwise deny an approved/rejected tool that is no longer in the freshly-built tool list (a rejected tool must reach the denial guard so it gets a clean "denied" result rather than a generic "not available"). decidedSlugs covers both slug-level and per-call decisions (a per-call approval sets no slug map entry, but its slug still needs re-exposing).
	for slug := range allowedEndpointTools {
		if !alwaysAllowed[slug] && !decidedSlugs[slug] {
			continue
		}
		exposeEndpointTool(slug)
	}

	// Restore endpoint-tools that search_api_tools revealed on earlier turns. Reveals live only on the per-turn run context, so without this a follow-up message that calls an already-discovered tool is denied until the model re-runs search_api_tools — the deny-then-re-search retry we want to eliminate. Replaying this run's recorded search queries against the freshly resolved grant rebuilds that set and re-checks permission for free: a tool the agent has since lost access to no longer matches the replayed search, so it is not re-exposed and the execution guard still denies it.
	var searchQueries []string
	for _, m := range messages {
		for _, tu := range m.ToolUse {
			if tu.Name != agents.SearchAPIToolsSlug {
				continue
			}
			var p struct {
				Query string `json:"query"`
			}
			if len(tu.Input) > 0 {
				_ = json.Unmarshal(tu.Input, &p)
			}
			searchQueries = append(searchQueries, p.Query)
		}
	}
	runCtx.RevealedToolSlugs = agents.RevealedSlugsForQueries(searchQueries, allowedEndpointTools)
	for slug := range runCtx.RevealedToolSlugs {
		exposeEndpointTool(slug)
	}

	// Finish the resumed turn: execute any tool calls this resume approved (rather than trusting the model to re-issue them — this is what makes an approval actually perform the write), then run the agent loop.
	result, err := s.runResumedLoop(ctx, run, accountID, agentIdentity, systemPrompt, modelChain, toolDefs, temperature, messages, &seq, runCtx, events, bc.spendingCapCents, bc.currentSpendCents)
	if err != nil {
		// Transient, side-effect-free failures are re-enqueued with backoff instead of surfaced as a terminal failure.
		if s.maybeAutoRetry(ctx, runRepo, run, err) {
			return nil
		}
		return s.failRun(ctx, runRepo, runID, startTime, err.Error())
	}

	// Persist outputs
	if err := s.persistOutputs(ctx, runID, accountID, bc.billingAccountID, result); err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to persist outputs: %s", err.Error()))
	}

	// Finalize run - manual runs go back to awaiting_input unless tools need approval. Accumulate token counts from previous continuations.
	durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
	statusCode := domain.RunStatusAwaitingInput
	switch {
	case result.Cancelled:
		// Stopped mid-flight — stay cancelled rather than reopening for input. Cancellation is not a completion — leave completed_at/duration_ms null.
		cumulativeInputTokens := run.TotalInputTokens + int64(result.InputTokens)
		cumulativeOutputTokens := run.TotalOutputTokens + int64(result.OutputTokens)
		if cancelErr := runRepo.UpdateCancelled(ctx, sqlc.UpdateAgentRunCancelledParams{
			Output:            result.Output,
			TotalInputTokens:  cumulativeInputTokens,
			TotalOutputTokens: cumulativeOutputTokens,
			ID:                runID,
		}); cancelErr != nil {
			return apierror.NewInternalError(cancelErr, "failed to finalize cancelled run")
		}
		statusCode = domain.RunStatusCancelled
	case result.AwaitingApproval:
		statusCode = domain.RunStatusAwaitingApproval
	}
	if statusCode != domain.RunStatusCancelled {
		cumulativeInputTokens := run.TotalInputTokens + int64(result.InputTokens)
		cumulativeOutputTokens := run.TotalOutputTokens + int64(result.OutputTokens)
		if completeErr := runRepo.UpdateCompleted(ctx, sqlc.UpdateAgentRunCompletedParams{
			StatusCode:        statusCode,
			Output:            result.Output,
			DurationMs:        agentdb.PgInt4(durationMs),
			TotalInputTokens:  cumulativeInputTokens,
			TotalOutputTokens: cumulativeOutputTokens,
			ID:                runID,
		}); completeErr != nil {
			return apierror.NewInternalError(completeErr, "failed to finalize run")
		}
	}

	// Write outbox message for run completion
	s.writeRunCompletedEvent(ctx, runID, accountID, bc.billingAccountID, result)

	// Finalize this turn's streaming reply (no-op for non-chat runs, which have no conversation_id). It threads under the message that triggered this turn (the user's reply) so the chat thread keeps growing as one. A re-pause on a later turn finalizes with the approval-request body so the chat UI can surface the new pending tools.
	s.writeChatComplete(ctx, run, runID, accountID, result, replyToMessageID)

	slog.Info("Agent continue run completed",
		"run_id", runID,
		"duration_ms", durationMs,
		"input_tokens", result.InputTokens,
		"output_tokens", result.OutputTokens,
	)

	return nil
}

// resumeApprovedBlockedCalls executes the tool calls that were paused for human review and have now been approved, directly — instead of leaving the run to depend on the model re-issuing the identical call after approval. The model does that unreliably: it often assumes approval alone executed the action and reports success while no call ever ran (the "agent said it updated the customer but nothing changed" bug). For each approved blocked call this runs the tool now, emits a tool_result event (for the timeline and for a truthful transcript on any later resume), and rewrites the "[REQUIRES APPROVAL]" placeholder in this turn's in-memory transcript with the real outcome so the model continues from what actually happened. Approvals it satisfies are consumed so a stray re-issue re-blocks rather than double-executing.
func (s *runnerSvc) resumeApprovedBlockedCalls(ctx context.Context, run *sqlc.AgentRun, accountID string, seq *int, runCtx *domain.HandlerRunContext, messages []llm.Message, events []sqlc.AgentRunEvent) {
	executedSlugs := make(map[string]bool)
	executedKeys := make(map[string]bool)
	for _, e := range events {
		if e.StepType != "tool_blocked" || e.Metadata == nil {
			continue
		}
		var meta struct {
			ToolUseID string          `json:"tool_use_id"`
			ToolName  string          `json:"tool_name"`
			Input     json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(e.Metadata, &meta); err != nil || meta.ToolUseID == "" {
			continue
		}
		// Only run calls this resume actually approved (by slug/approve-all or by this call's slug+input key) — the same gate the loop uses.
		key := toolCallApprovalKey(meta.ToolName, meta.Input)
		if !runCtx.OneTimeApprovedSlugs[meta.ToolName] && !runCtx.OneTimeApprovedKeys[key] {
			continue
		}
		// Idempotency guard: skip a call an earlier resume already executed (it has a tool_result event), so re-approving or a re-delivered continuation can't double-run it.
		if hasToolResultEvent(events, meta.ToolUseID) {
			executedSlugs[meta.ToolName] = true
			executedKeys[key] = true
			continue
		}

		toolStart := time.Now()
		result, err := s.handleToolCall(ctx, llm.ToolCall{ID: meta.ToolUseID, Name: meta.ToolName, Input: meta.Input}, runCtx)
		durMs := safeconv.Int64ToInt32(time.Since(toolStart).Milliseconds())

		if err != nil {
			resultMeta, _ := json.Marshal(map[string]any{"tool_use_id": meta.ToolUseID, "is_error": true, "full_result": err.Error()})
			truncated := truncateString(err.Error(), 500)
			s.emitEvent(ctx, run.ID, accountID, seq, "tool_result", meta.ToolName+" result", &truncated, &durMs, nil, resultMeta)
			rewriteToolResult(messages, meta.ToolUseID, fmt.Sprintf("Error: %s", err.Error()), true)
		} else {
			trunc := llm.TruncateToolOutputResult(result, meta.ToolName)
			resultMeta, _ := json.Marshal(map[string]any{"tool_use_id": meta.ToolUseID, "is_error": false, "full_result": trunc.Content})
			truncatedEvent := truncateString(trunc.Content, 500)
			s.emitEvent(ctx, run.ID, accountID, seq, "tool_result", meta.ToolName+" result", &truncatedEvent, &durMs, nil, resultMeta)
			rewriteToolResult(messages, meta.ToolUseID, trunc.Content, false)
		}
		executedSlugs[meta.ToolName] = true
		executedKeys[key] = true
	}

	// Consume the approvals we just satisfied. Without this the slug/key stays armed and the model re-issuing the same call (new tool_use_id) would sail through the guard and write a second time.
	for slug := range executedSlugs {
		delete(runCtx.OneTimeApprovedSlugs, slug)
	}
	for key := range executedKeys {
		delete(runCtx.OneTimeApprovedKeys, key)
	}
}

// rewriteToolResult replaces the content of the tool_result block for toolUseID in the reconstructed transcript. Used to swap an approval placeholder for the tool's real outcome. Returns whether a block was found.
func rewriteToolResult(messages []llm.Message, toolUseID, content string, isError bool) bool {
	for i := range messages {
		for j := range messages[i].ToolResults {
			if messages[i].ToolResults[j].ToolUseID == toolUseID {
				messages[i].ToolResults[j].Content = content
				messages[i].ToolResults[j].IsError = isError
				return true
			}
		}
	}
	return false
}

// hasToolResultEvent reports whether any tool_result event was recorded for toolUseID — i.e. the call already executed on an earlier turn.
func hasToolResultEvent(events []sqlc.AgentRunEvent, toolUseID string) bool {
	for _, e := range events {
		if e.StepType != "tool_result" || e.Metadata == nil {
			continue
		}
		var meta struct {
			ToolUseID string `json:"tool_use_id"`
		}
		if err := json.Unmarshal(e.Metadata, &meta); err == nil && meta.ToolUseID == toolUseID {
			return true
		}
	}
	return false
}

// reconstructMessages rebuilds LLM message history from stored events.
func reconstructMessages(events []sqlc.AgentRunEvent) []llm.Message {
	var messages []llm.Message
	var pendingAssistant *llm.Message
	// Where each tool_use_id's result block lives, so a later terminal event for the same call overwrites the earlier block in place rather than appending a second one. A call that was paused for approval (tool_blocked) and then executed on resume (tool_result) emits two terminal events with the same id — without last-wins overwrite the transcript would carry two tool_result blocks for one tool_use_id and the Anthropic API rejects the next call.
	resultLoc := make(map[string][2]int)

	for _, event := range events {
		switch event.StepType {
		case "user_message":
			// Flush pending assistant message
			if pendingAssistant != nil {
				messages = append(messages, *pendingAssistant)
				pendingAssistant = nil
			}
			content := ""
			if event.Content.Valid {
				content = event.Content.String
			}
			messages = append(messages, llm.Message{Role: "user", Content: content})

		case "assistant_message":
			// Flush pending assistant message
			if pendingAssistant != nil {
				messages = append(messages, *pendingAssistant)
				pendingAssistant = nil
			}
			content := ""
			if event.Content.Valid {
				content = event.Content.String
			}
			messages = append(messages, llm.Message{Role: "assistant", Content: content})

		case "tool_call":
			// Parse metadata for tool_use_id and input
			var meta map[string]json.RawMessage
			if event.Metadata != nil {
				_ = json.Unmarshal(event.Metadata, &meta)
			}

			var toolUseID, toolName string
			var input json.RawMessage
			if raw, ok := meta["tool_use_id"]; ok {
				_ = json.Unmarshal(raw, &toolUseID)
			}
			if raw, ok := meta["tool_name"]; ok {
				_ = json.Unmarshal(raw, &toolName)
			}
			if raw, ok := meta["input"]; ok {
				input = raw
			}

			// Create or append to pending assistant message
			if pendingAssistant == nil {
				pendingAssistant = &llm.Message{Role: "assistant"}
			}
			pendingAssistant.ToolUse = append(pendingAssistant.ToolUse, llm.ToolUseBlock{
				ID:    toolUseID,
				Name:  toolName,
				Input: input,
			})

		case "tool_result", "tool_blocked", "tool_denied", "doom_loop_detected":
			// Every event that terminates a tool call must reconstruct into a tool_result block — otherwise the assistant's tool_use is left unanswered and the Anthropic API rejects the next call with "tool_use ids were found without tool_result blocks". tool_denied and doom_loop_detected carry their message in event.Content (not full_result), so they fall through to the content fallback below.
			//
			// Flush pending assistant message before tool results
			if pendingAssistant != nil {
				messages = append(messages, *pendingAssistant)
				pendingAssistant = nil
			}

			var meta map[string]json.RawMessage
			if event.Metadata != nil {
				_ = json.Unmarshal(event.Metadata, &meta)
			}

			var toolUseID string
			var isError bool
			var fullResult string
			if raw, ok := meta["tool_use_id"]; ok {
				_ = json.Unmarshal(raw, &toolUseID)
			}
			if raw, ok := meta["is_error"]; ok {
				_ = json.Unmarshal(raw, &isError)
			}
			if raw, ok := meta["full_result"]; ok {
				_ = json.Unmarshal(raw, &fullResult)
			}
			// Blocked/denied/doom-loop events store their message in event.Content rather than a full_result, so fall back to it when no inline result was recorded.
			if fullResult == "" && event.Content.Valid {
				fullResult = event.Content.String
			}
			// A tool result with no recoverable id can't be matched to its tool_use; skip it (the orphan sweep below backfills the dangling tool_use with a placeholder so nothing is left unanswered).
			if toolUseID == "" {
				continue
			}

			// Last-wins: if this call already has a result block (e.g. an earlier tool_blocked placeholder that a resume then executed), overwrite it in place so the transcript carries exactly one, current result per tool_use_id.
			if loc, ok := resultLoc[toolUseID]; ok {
				messages[loc[0]].ToolResults[loc[1]].Content = fullResult
				messages[loc[0]].ToolResults[loc[1]].IsError = isError
				continue
			}

			block := llm.ToolResultBlock{ToolUseID: toolUseID, Content: fullResult, IsError: isError}
			// Find or create the last user message with tool results
			if len(messages) > 0 {
				last := &messages[len(messages)-1]
				if last.Role == "user" && len(last.ToolResults) > 0 {
					last.ToolResults = append(last.ToolResults, block)
					resultLoc[toolUseID] = [2]int{len(messages) - 1, len(last.ToolResults) - 1}
					continue
				}
			}
			messages = append(messages, llm.Message{
				Role:        "user",
				ToolResults: []llm.ToolResultBlock{block},
			})
			resultLoc[toolUseID] = [2]int{len(messages) - 1, 0}
		}
	}

	// Flush any remaining pending assistant
	if pendingAssistant != nil {
		messages = append(messages, *pendingAssistant)
	}

	return ensureToolResults(messages)
}

// ensureToolResults enforces the Anthropic invariant that every assistant tool_use block is answered by a tool_result in the immediately following message. Reconstruction can leave a gap — an unknown terminal event type, a tool_call whose result event lost its id, or a run that died mid-turn before the result was recorded. A single orphaned tool_use makes the gateway reject the next call with a 400 ("tool_use ids were found without tool_result blocks") and, because the bad history is replayed on every resume, permanently bricks the run. This backfills any missing result with a synthetic error block so the run can always make forward progress.
func ensureToolResults(messages []llm.Message) []llm.Message {
	out := make([]llm.Message, 0, len(messages))
	for i := range messages {
		msg := messages[i]
		out = append(out, msg)
		if msg.Role != "assistant" || len(msg.ToolUse) == 0 {
			continue
		}

		// Results for this turn live in the next message when it's a tool_result-bearing user turn.
		var next *llm.Message
		answered := make(map[string]bool, len(msg.ToolUse))
		if i+1 < len(messages) && messages[i+1].Role == "user" && len(messages[i+1].ToolResults) > 0 {
			next = &messages[i+1]
			for _, tr := range next.ToolResults {
				answered[tr.ToolUseID] = true
			}
		}

		var missing []llm.ToolResultBlock
		for _, tu := range msg.ToolUse {
			if !answered[tu.ID] {
				missing = append(missing, llm.ToolResultBlock{
					ToolUseID: tu.ID,
					Content:   "Tool result unavailable — the previous run ended before this tool reported back.",
					IsError:   true,
				})
			}
		}
		if len(missing) == 0 {
			continue
		}
		if next != nil {
			// Mutating the slice element backfills the message the outer loop appends on its next pass.
			next.ToolResults = append(missing, next.ToolResults...)
		} else {
			out = append(out, llm.Message{Role: "user", ToolResults: missing})
		}
	}
	return out
}

func (s *runnerSvc) handleToolCall(ctx context.Context, tc llm.ToolCall, runCtx *domain.HandlerRunContext) (string, error) {
	handler, ok := s.toolRegistry.Get(tc.Name)
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", tc.Name), nil
	}
	// Expose the current tool-use ID so handlers can derive a deterministic idempotency key (RunID+ToolUseID) for mutating gateway calls. Calls are handled sequentially, so a per-context field is safe.
	runCtx.ToolUseID = tc.ID
	return handler(ctx, tc.Input, runCtx)
}

func (s *runnerSvc) persistOutputs(ctx context.Context, runID, accountID, billingAccountID string, result *domain.RunResult) error {
	ctx, span := tracing.StartSpan(ctx, runnerTracer, "service.runner.persist_outputs")
	defer span.End()

	actionRepo := s.repos.NewAgentActionRepo()
	tokenRepo := s.repos.NewAgentTokenUsageRepo()

	// Persist actions
	for _, a := range result.Actions {
		actionID, err := id.GenID(id.AgentActionIDPrefix, nil)
		if err != nil {
			return fmt.Errorf("failed to generate action ID: %w", err)
		}

		statusCode := domain.ActionStatusAutoApproved
		if a.RequiresReview {
			statusCode = domain.ActionStatusPendingReview
		}

		actionInput := a.Input
		if actionInput == nil {
			actionInput = json.RawMessage(`{}`)
		}
		actionOutput := a.Output
		if actionOutput == nil {
			actionOutput = json.RawMessage(`{}`)
		}

		if insertErr := actionRepo.Insert(ctx, sqlc.InsertAgentActionParams{
			ID:             actionID,
			AccountID:      accountID,
			AgentRunID:     runID,
			ToolSlug:       a.ToolSlug,
			StatusCode:     statusCode,
			Label:          agentdb.PgText(a.Label),
			Description:    agentdb.PgText(a.Description),
			Input:          actionInput,
			Output:         actionOutput,
			RequiresReview: a.RequiresReview,
		}); insertErr != nil {
			return fmt.Errorf("failed to insert action: %w", insertErr)
		}
	}

	// Upsert token usage
	tokenID, err := id.GenID(id.AgentTokenUsageIDPrefix, nil)
	if err != nil {
		return fmt.Errorf("failed to generate token usage ID: %w", err)
	}

	if upsertErr := tokenRepo.Upsert(ctx, sqlc.UpsertAgentTokenUsageParams{
		ID:           tokenID,
		AccountID:    billingAccountID,
		Date:         agentdb.PgDate(time.Now().Truncate(24 * time.Hour)),
		InputTokens:  int64(result.InputTokens),
		OutputTokens: int64(result.OutputTokens),
		TotalCost:    0, // cost calculation deferred
		RunCount:     1,
	}); upsertErr != nil {
		return fmt.Errorf("failed to upsert token usage: %w", upsertErr)
	}

	return nil
}

func (s *runnerSvc) writeRunCompletedEvent(ctx context.Context, runID, accountID, billingAccountID string, result *domain.RunResult) {
	length := id.IDLength22
	msgID, err := id.GenID(id.MessageIDPrefix, &length)
	if err != nil {
		slog.Error("Failed to generate message ID for run completed event", "error", err)
		return
	}

	data := messaging.AgentRunCompletedData{
		AgentRunID:       runID,
		AccountID:        accountID,
		BillingAccountID: billingAccountID,
		InputTokens:      result.InputTokens,
		OutputTokens:     result.OutputTokens,
		TotalTokens:      result.InputTokens + result.OutputTokens,
		LLMProvider:      result.LLMProvider,
		LLMModel:         result.LLMModel,
	}
	dataBytes, _ := json.Marshal(data)

	// Publish directly for real-time WebSocket delivery (fire-and-forget).
	if s.broker != nil {
		_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange,
			string(contracts.AgentEventRunCompleted), contracts.AmqpMessage{Data: dataBytes})
	}

	// Also write to outbox for reliable downstream consumers (e.g. billing/token usage).
	if _, outboxErr := s.outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		MessageID:   msgID,
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.AgentEventRunCompleted),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.AgentEventRunCompleted),
		Payload: contracts.AmqpMessage{
			Data:      dataBytes,
			MessageID: msgID,
		},
		MaxAttempts: 3,
	}); outboxErr != nil {
		slog.Error("Failed to write run completed event to outbox", "error", outboxErr, "run_id", runID)
	}
	// Publish the run-completed event now rather than on the enqueuer's next idle poll.
	s.kickOutbox()
}

// writeRunCompletedNotification enqueues an in-app "agent run completed" notification to the user who triggered the run. Best-effort: it fires only for user-triggered runs (not scheduled/system/agent triggers), attributing the notification to the agent as sender and linking to the run. The notification-service fan-out resolves the recipient user id to the per-account account_user id and pushes it to the bell in real time.
func (s *runnerSvc) writeRunCompletedNotification(ctx context.Context, run *sqlc.AgentRun, def *sqlc.AgentDefinition, runID, accountID string) {
	identityType := agentdb.StringFromPgText(run.TriggeredByIdentityType)
	actorID := agentdb.StringFromPgText(run.TriggeredByActorID)
	if identityType == nil || *identityType != string(types.IdentityActorTypeUser) || actorID == nil || *actorID == "" {
		return // only the human who started the run gets the bell notification
	}

	// The sender is the agent; attribute by its config id when present, else its definition id.
	senderID := def.ID
	if cfgID := agentdb.StringFromPgText(run.AgentConfigID); cfgID != nil && *cfgID != "" {
		senderID = *cfgID
	}

	data := messaging.AlertFanoutData{
		AccountID:        accountID,
		Category:         "agent.run_completed",
		Kind:             "alert",
		Title:            fmt.Sprintf("%s finished a run", def.Name),
		Body:             "Your agent run has completed.",
		LinkResourceType: string(constants.ObjectTypeAgentRun),
		LinkResourceID:   runID,
		Priority:         "normal",
		SenderType:       string(constants.NotificationSenderTypeAgent),
		SenderID:         senderID,
		SenderName:       def.Name,
		RecipientUserIDs: []string{*actorID},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal run completed notification", "error", err, "run_id", runID)
		return
	}

	length := id.IDLength22
	msgID, genErr := id.GenID(id.MessageIDPrefix, &length)
	if genErr != nil {
		slog.Error("Failed to generate message ID for run completed notification", "error", genErr)
		return
	}

	if _, outboxErr := s.outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		MessageID:   msgID,
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdFanout),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdFanout),
		Payload:     contracts.AmqpMessage{Data: payload, MessageID: msgID},
		MaxAttempts: 3,
	}); outboxErr != nil {
		slog.Error("Failed to enqueue run completed notification", "error", outboxErr, "run_id", runID)
	}
}

// beginChatStream posts a chat turn's reply message empty and in a streaming state, returning a context carrying the stream handle so subsequent token patches and the terminal finalize address the same record. The agent owns the message id (so it can patch before notification-service has acked the create). No-op (returns ctx unchanged) for non-conversation turns and runs without a conversation — the same gate as the reply itself.
func (s *runnerSvc) beginChatStream(ctx context.Context, run *sqlc.AgentRun, runID, accountID, replyToMessageID, agentName string) context.Context {
	if !chatTurnFromConversation(ctx) {
		return ctx
	}
	conversationID := agentdb.StringFromPgText(run.ConversationID)
	if conversationID == nil || *conversationID == "" {
		return ctx
	}
	// Per-turn idempotency key (the run's event sequence advances during the turn, so capture it now and reuse it for the finalize — stable across redelivery, distinct per turn).
	turn := int32(0)
	if maxSeq, seqErr := s.repos.NewAgentRunEventRepo().GetMaxSequence(ctx, runID); seqErr == nil {
		turn = maxSeq
	}
	length := id.IDLength22
	messageID, genErr := id.GenID(id.MessageIDPrefix, &length)
	if genErr != nil {
		slog.Error("Failed to generate streaming reply message id", "error", genErr, "run_id", runID)
		return ctx
	}
	css := &chatStreamState{
		messageID:       messageID,
		accountID:       accountID,
		conversationID:  *conversationID,
		clientMessageID: fmt.Sprintf("agentreply:%s:%d", runID, turn),
		agentName:       agentName,
	}
	s.enqueueAgentReply(ctx, messaging.AgentReplyData{
		AccountID:        accountID,
		ConversationID:   *conversationID,
		AgentConfigID:    run.AgentDefinitionID,
		AgentName:        agentName,
		AgentRunID:       runID,
		MessageID:        messageID,
		ClientMessageID:  css.clientMessageID,
		ReplyToMessageID: replyToMessageID,
		Phase:            "start",
	})
	return withChatStream(ctx, css)
}

// writeChatComplete finalizes a chat turn's streaming reply: it sets the completed answer and flips the message to complete. If the stream never started (an early infra failure before beginChatStream), it falls back to creating the reply outright — the legacy single-shot path, which still drops an empty body. replyToMessageID threads this turn's reply under the message that triggered it.
func (s *runnerSvc) writeChatComplete(ctx context.Context, run *sqlc.AgentRun, runID, accountID string, result *domain.RunResult, replyToMessageID string) {
	var out struct {
		Response string `json:"response"`
	}
	_ = json.Unmarshal(result.Output, &out)
	s.finalizeChatReply(ctx, run, runID, accountID, out.Response, replyToMessageID, false)
}

// finalizeChatReply sends the terminal "final" reply for a turn. When a stream is in flight it addresses the existing record (and always finalizes, even on an empty body, so a started bubble never orphans); otherwise it creates a fresh message and preserves the legacy "drop empty replies" behavior.
func (s *runnerSvc) finalizeChatReply(ctx context.Context, run *sqlc.AgentRun, runID, accountID, body, replyToMessageID string, failed bool) {
	if !chatTurnFromConversation(ctx) {
		return
	}
	conversationID := agentdb.StringFromPgText(run.ConversationID)
	if conversationID == nil || *conversationID == "" {
		return
	}
	body = strings.TrimSpace(body)

	css := chatStreamFromContext(ctx)
	messageID := ""
	clientMessageID := ""
	if css != nil {
		messageID = css.messageID
		clientMessageID = css.clientMessageID
	} else {
		// Legacy single-shot create-and-complete: nothing was started, so a truly empty reply is dropped exactly as before (no orphan bubble to resolve).
		if body == "" {
			return
		}
		turn := int32(0)
		if maxSeq, seqErr := s.repos.NewAgentRunEventRepo().GetMaxSequence(ctx, runID); seqErr == nil {
			turn = maxSeq
		}
		clientMessageID = fmt.Sprintf("agentreply:%s:%d", runID, turn)
	}

	s.enqueueAgentReply(ctx, messaging.AgentReplyData{
		AccountID:        accountID,
		ConversationID:   *conversationID,
		AgentConfigID:    run.AgentDefinitionID,
		AgentName:        s.chatAgentName(ctx, run),
		AgentRunID:       runID,
		MessageID:        messageID,
		Body:             body,
		ClientMessageID:  clientMessageID,
		ReplyToMessageID: replyToMessageID,
		Phase:            "final",
		Failed:           failed,
	})
}

// writeChatApprovalNotice posts a non-bubble timeline event into the run's conversation recording who approved which tool(s) (e.g. "Dane approved update_customer"), so the thread shows the human decision before the agent's resumed reply. No-op for runs not linked to a conversation. Keyed on the run id + current turn so a redelivered continue doesn't double-post.
func (s *runnerSvc) writeChatApprovalNotice(ctx context.Context, run *sqlc.AgentRun, runID, accountID, approverName string, approvedSlugs []string) {
	conversationID := agentdb.StringFromPgText(run.ConversationID)
	if conversationID == nil || *conversationID == "" || len(approvedSlugs) == 0 {
		return
	}
	who := strings.TrimSpace(approverName)
	if who == "" {
		who = "Someone"
	}
	turn := int32(0)
	if maxSeq, seqErr := s.repos.NewAgentRunEventRepo().GetMaxSequence(ctx, runID); seqErr == nil {
		turn = maxSeq
	}
	s.enqueueAgentReply(ctx, messaging.AgentReplyData{
		AccountID:       accountID,
		ConversationID:  *conversationID,
		AgentRunID:      runID,
		Body:            fmt.Sprintf("%s approved %s", who, strings.Join(approvedSlugs, ", ")),
		ClientMessageID: fmt.Sprintf("approval:%s:%d", runID, turn),
		ApprovalEvent:   true,
	})
}

// writeChatRejectionNotice mirrors writeChatApprovalNotice for the deny side: it posts a non-bubble "who denied what" timeline event so a chat thread records the rejection decision.
func (s *runnerSvc) writeChatRejectionNotice(ctx context.Context, run *sqlc.AgentRun, runID, accountID, reviewerName string, rejectedSlugs []string) {
	conversationID := agentdb.StringFromPgText(run.ConversationID)
	if conversationID == nil || *conversationID == "" || len(rejectedSlugs) == 0 {
		return
	}
	who := strings.TrimSpace(reviewerName)
	if who == "" {
		who = "Someone"
	}
	turn := int32(0)
	if maxSeq, seqErr := s.repos.NewAgentRunEventRepo().GetMaxSequence(ctx, runID); seqErr == nil {
		turn = maxSeq
	}
	s.enqueueAgentReply(ctx, messaging.AgentReplyData{
		AccountID:       accountID,
		ConversationID:  *conversationID,
		AgentRunID:      runID,
		Body:            fmt.Sprintf("%s denied %s", who, strings.Join(rejectedSlugs, ", ")),
		ClientMessageID: fmt.Sprintf("rejection:%s:%d", runID, turn),
		ApprovalEvent:   true,
	})
}

// enqueueAgentReply writes a durable agent-reply command (start or final) to the outbox.
// notification-service resolves the agent participant from (conversation_id, agent_config_id — the participant stores the agent definition id) and creates/finalizes a kind=agent message linked to the run. The outbox envelope id is distinct from data.MessageID (the message row the reply targets).
func (s *runnerSvc) enqueueAgentReply(ctx context.Context, data messaging.AgentReplyData) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal agent chat reply", "error", err, "run_id", data.AgentRunID, "phase", data.Phase)
		return
	}
	length := id.IDLength22
	msgID, genErr := id.GenID(id.MessageIDPrefix, &length)
	if genErr != nil {
		slog.Error("Failed to generate message id for agent chat reply", "error", genErr)
		return
	}
	if _, outboxErr := s.outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		MessageID:   msgID,
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdAgentReply),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdAgentReply),
		Payload:     contracts.AmqpMessage{Data: payload, MessageID: msgID},
		MaxAttempts: 3,
	}); outboxErr != nil {
		slog.Error("Failed to enqueue agent chat reply", "error", outboxErr, "run_id", data.AgentRunID, "phase", data.Phase)
	}
	// Post the reply bubble / its finalize now rather than on the enqueuer's next idle poll.
	s.kickOutbox()
}

// maybePatchChatStream streams the growing answer into the in-flight chat reply message, throttled to the chatStreamPatch cadence. No-op when no stream is in flight.
func (s *runnerSvc) maybePatchChatStream(ctx context.Context, body string) {
	css := chatStreamFromContext(ctx)
	if css == nil {
		return
	}
	css.mu.Lock()
	due := time.Since(css.lastFlush) >= chatStreamPatchInterval || len(body)-css.lastLen >= chatStreamPatchMinChars
	if due {
		css.lastFlush = time.Now()
		css.lastLen = len(body)
	}
	css.mu.Unlock()
	if due {
		s.publishChatPatch(ctx, css, body)
	}
}

// patchChatStream pushes the body to the in-flight chat reply message immediately, bypassing the throttle (used at iteration boundaries — a tool call clears the bubble; the end of a call flushes its final text). No-op when no stream is in flight.
func (s *runnerSvc) patchChatStream(ctx context.Context, body string) {
	css := chatStreamFromContext(ctx)
	if css == nil {
		return
	}
	css.mu.Lock()
	css.lastFlush = time.Now()
	css.lastLen = len(body)
	css.mu.Unlock()
	s.publishChatPatch(ctx, css, body)
}

// publishChatPatch fires a best-effort partial-body update straight to the exchange (not the outbox):
// patches are lossy by design — the full accumulated body is last-write-wins and the "final" reply reconciles the persisted row.
func (s *runnerSvc) publishChatPatch(ctx context.Context, css *chatStreamState, body string) {
	if s.broker == nil {
		return
	}
	data, err := json.Marshal(messaging.AgentReplyPatchData{
		AccountID:      css.accountID,
		ConversationID: css.conversationID,
		MessageID:      css.messageID,
		Body:           body,
	})
	if err != nil {
		return
	}
	_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange,
		string(contracts.NotificationCmdAgentReplyPatch), contracts.AmqpMessage{Data: data})
}

// emitChatRunStarted publishes a best-effort "agent run started" event on the run's conversation topic so chat subscribers learn the run id the moment it begins and can subscribe to its live step stream (run:<id>), rendering the agent's interim work inline. It mirrors the conversation typing event (published straight to the realtime fanout, bypassing the outbox); a run not linked to a conversation is a no-op, and publish failures are swallowed — the signal is disposable (the persisted reply still arrives, and the message's agent_run_id covers post-hoc expansion).
func (s *runnerSvc) emitChatRunStarted(ctx context.Context, accountID string, run *sqlc.AgentRun) {
	if s.broker == nil {
		return
	}
	conversationID := agentdb.StringFromPgText(run.ConversationID)
	if conversationID == nil || *conversationID == "" {
		return
	}
	triggerMessageID := ""
	if tm := agentdb.StringFromPgText(run.TriggerMessageID); tm != nil {
		triggerMessageID = *tm
	}
	payload, _ := json.Marshal(map[string]string{
		"run_id":             run.ID,
		"agent_config_id":    run.AgentDefinitionID,
		"trigger_message_id": triggerMessageID,
	})
	data, err := json.Marshal(messaging.RealtimeDeliveryData{
		AccountID:      accountID,
		ConversationID: *conversationID,
		Event:          "agent_run_started",
		Payload:        payload,
	})
	if err != nil {
		return
	}
	_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange,
		string(contracts.NotificationEventDelivered), contracts.AmqpMessage{Data: data})
}

// retryableRunError marks a run failure that is transient (the whole model chain was momentarily unavailable) and side-effect-free (no tools executed this turn), so the runner may transparently re-enqueue the run instead of surfacing a terminal failure. It wraps the underlying gateway error so callers that don't auto-retry still get the original cause via errors.Unwrap.
type retryableRunError struct {
	cause error
}

func (e *retryableRunError) Error() string { return e.cause.Error() }
func (e *retryableRunError) Unwrap() error { return e.cause }

// autoRetryBackoff schedules the re-enqueue delay for bounded auto-retry. The waits are deliberately long (tens of seconds, growing toward two minutes) because the trigger is "the entire model chain is down": a no-delay re-enqueue would just hit the still-down providers and burn the retry budget. attempt is 0-indexed by the run's current retry_count.
var autoRetryBackoff = &retry.Config{
	InitialWait:    15 * time.Second,
	MaxWait:        2 * time.Minute,
	Multiplier:     3.0,
	JitterFraction: 0.2,
}

// maybeAutoRetry transparently re-enqueues a run that failed on a transient, side-effect-free error (signalled by retryableRunError) instead of failing it, as long as the run is still under the auto-retry cap. It bumps retry_count (guarded on status='running' so a concurrent transition can't double-enqueue), then writes a delayed AgentCmdContinueRun outbox message so the run resumes after the chain has had time to recover. Returns true when the run was re-enqueued (the caller must NOT then mark it failed); false means the caller should fall through to its normal terminal-failure handling.
func (s *runnerSvc) maybeAutoRetry(ctx context.Context, runRepo domain.AgentRunRepo, run *sqlc.AgentRun, err error) bool {
	var rr *retryableRunError
	if !errors.As(err, &rr) {
		return false
	}
	if int(run.RetryCount) >= domain.MaxAutoRetries {
		return false
	}

	// Bump the counter first (this also bounds the loop). On failure the run is no longer running — a stop/complete raced us — so there is nothing to re-enqueue.
	if _, markErr := runRepo.MarkAutoRetrying(ctx, run.ID); markErr != nil {
		return false
	}

	delay := retry.CalculateDelay(autoRetryBackoff, int(run.RetryCount))
	delaySecs := max(int(delay.Seconds()), 1)

	length := id.IDLength22
	msgID, genErr := id.GenID(id.MessageIDPrefix, &length)
	if genErr != nil {
		slog.Error("auto-retry: failed to generate message ID", "error", genErr, "run_id", run.ID)
		s.revertAutoRetry(ctx, runRepo, run.ID, err)
		return false
	}

	// Resume the existing transcript with no new input, mirroring the manual retry path. Reply back into the original thread (if any) so a chat-triggered run's resumed reply is posted, not kept private.
	replyTo := ""
	if tid := agentdb.StringFromPgText(run.TriggerMessageID); tid != nil {
		replyTo = *tid
	}
	data := messaging.AgentContinueRunData{
		AgentRunID:       run.ID,
		AccountID:        run.AccountID,
		Message:          "",
		ReplyToMessageID: replyTo,
	}
	dataBytes, _ := json.Marshal(data)

	if _, outboxErr := s.outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		MessageID:   msgID,
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.AgentCmdContinueRun),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.AgentCmdContinueRun),
		Payload:     contracts.AmqpMessage{Data: dataBytes, MessageID: msgID},
		MaxAttempts: 3,
		// Hold the resume command back so the chain has time to recover. Intentionally NOT kicking the enqueuer here — the whole point is to wait out next_run_at, not publish immediately.
		DelaySeconds: delaySecs,
	}); outboxErr != nil {
		slog.Error("auto-retry: failed to enqueue resume", "error", outboxErr, "run_id", run.ID)
		s.revertAutoRetry(ctx, runRepo, run.ID, err)
		return false
	}

	// Best-effort timeline marker so the live run view shows the wait instead of looking stalled.
	meta, _ := json.Marshal(map[string]any{"retry_in_seconds": delaySecs, "attempt": int(run.RetryCount) + 1})
	content := fmt.Sprintf("The model provider is temporarily unavailable. Retrying in %ds…", delaySecs)
	seq := 0
	if maxSeq, seqErr := s.repos.NewAgentRunEventRepo().GetMaxSequence(ctx, run.ID); seqErr == nil {
		seq = int(maxSeq) + 1
	}
	s.emitEvent(ctx, run.ID, run.AccountID, &seq, "retry_scheduled", "Retry scheduled", &content, nil, nil, meta)

	slog.Info("auto-retrying run after transient failure", "run_id", run.ID, "attempt", int(run.RetryCount)+1, "delay_seconds", delaySecs)
	return true
}

// revertAutoRetry rolls a run back to a terminal failed state after an auto-retry bumped the counter but could not enqueue the resume, so the increment doesn't leave the run stuck in 'running' with no pending work. Manual retry is still possible afterward (retry_count is preserved).
func (s *runnerSvc) revertAutoRetry(ctx context.Context, runRepo domain.AgentRunRepo, runID string, cause error) {
	if failErr := runRepo.UpdateFailed(ctx, sqlc.UpdateAgentRunFailedParams{
		ErrorMessage: agentdb.PgText(cause.Error()),
		ID:           runID,
	}); failErr != nil {
		slog.Error("auto-retry: failed to revert run to failed", "error", failErr, "run_id", runID)
	}
}

func (s *runnerSvc) failRun(ctx context.Context, runRepo domain.AgentRunRepo, runID string, startTime time.Time, errMsg string) *apierror.APIError {
	durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
	if failErr := runRepo.UpdateFailed(ctx, sqlc.UpdateAgentRunFailedParams{
		ErrorMessage: agentdb.PgText(errMsg),
		DurationMs:   agentdb.PgInt4(durationMs),
		ID:           runID,
	}); failErr != nil {
		slog.Error("Failed to mark run as failed", "error", failErr, "run_id", runID)
	}

	// Surface the failure to users. Without this, a failed run just stops: the timeline ends with no terminal marker, the live run view never updates, and a chat-triggered run looks silently dropped.
	// Loaded once and reused for both the timeline event and the chat reply. Best-effort.
	if run, loadErr := runRepo.GetByID(ctx, runID); loadErr == nil && run != nil {
		s.emitFailureEvent(ctx, run, errMsg)
		s.writeChatFailureReply(ctx, run, runID, run.AccountID)
	}

	return apierror.NewInternalError(errors.New(errMsg), "agent run failed: "+errMsg)
}

// emitFailureEvent appends a terminal "error" step to the run timeline so the run-detail view and the live stream show that the work was attempted and failed, instead of the run ending with no marker.
// The raw error (which may carry internal/provider detail) is kept in metadata for operators; the user-facing content is generic.
func (s *runnerSvc) emitFailureEvent(ctx context.Context, run *sqlc.AgentRun, errMsg string) {
	maxSeq, err := s.repos.NewAgentRunEventRepo().GetMaxSequence(ctx, run.ID)
	if err != nil {
		return
	}
	seq := int(maxSeq) + 1
	content := "The run failed before it could finish."
	meta, _ := json.Marshal(map[string]any{"error": errMsg})
	// Terminal: also drives a run_complete WS frame so the live run view leaves its loading state.
	s.emitTerminalEvent(ctx, run.ID, run.AccountID, &seq, "error", "Run failed", &content, nil, nil, meta)
}

// writeChatFailureReply posts a brief, user-facing failure notice into the run's conversation so the thread learns the work failed instead of assuming it was silently dropped. The detailed error is kept on the run's error_message for operators; chat sees only a generic message (raw errors may contain internal/provider detail). No-op for runs without a conversation.
func (s *runnerSvc) writeChatFailureReply(ctx context.Context, run *sqlc.AgentRun, runID, accountID string) {
	// Gating (conversation turn + linked conversation) is handled in finalizeChatReply. If a streaming bubble was already posted for this turn, this resolves it to the error message rather than leaving it stuck "thinking".
	var replyTo string
	if tid := agentdb.StringFromPgText(run.TriggerMessageID); tid != nil {
		replyTo = *tid
	}
	const msg = "Sorry — I ran into an error and couldn't finish that request. Please try again."
	s.finalizeChatReply(ctx, run, runID, accountID, msg, replyTo, true)
}

// resolveAllowedEndpointTools turns an agent's configured endpoint_tool_slugs allow-list into the concrete set of catalog slugs it may use. The single entry "*" grants the whole catalog; otherwise only slugs that exist in the catalog are included.
func resolveAllowedEndpointTools(slugs []string) map[string]bool {
	allowed := make(map[string]bool)
	for _, s := range slugs {
		if s == "*" {
			for _, et := range agents.EndpointTools {
				allowed[et.Slug] = true
			}
			return allowed
		}
		if _, ok := agents.LookupEndpointTool(s); ok {
			allowed[s] = true
		}
	}
	return allowed
}

// buildAgentToolDefs assembles the tools an agent run exposes up front and the per-tool require-review map. Up-front tools are the agent's explicitly-linked DB tools (hand-crafted), its linked built-in tools, and — only when the agent has been granted endpoint-tools — the search_api_tools meta-tool. The endpoint-tools themselves are NOT injected here; the agent discovers them on demand via search (progressive disclosure) and the runner reveals matches into the live tool list. allowedEndpointTools is the agent's resolved grant; each granted endpoint-tool is gated only when endpointReview marks its slug true (default off, mirroring linked built-in tools), so gating applies once it is revealed and called.
func (s *runnerSvc) buildAgentToolDefs(linkedTools []sqlc.ListToolsByAgentDefinitionIDRow, allowedEndpointTools map[string]bool, endpointReview map[string]bool) ([]llm.ToolDefinition, map[string]bool) {
	requireReviewBySlug := make(map[string]bool, len(linkedTools)+len(allowedEndpointTools))

	// Linked tools are built-in tools, granted by slug. Their description and input schema come from the code catalog (agents.BuiltinTools), not the database.
	var toolDefs []llm.ToolDefinition
	for _, lt := range linkedTools {
		slug := lt.ToolSlug
		if slug == "" {
			continue
		}
		bt, ok := agents.LookupBuiltinTool(slug)
		// A built-in tool requires review when the agent's link opts in OR when the tool is mutating (an externally-visible / irreversible action like send_email). The mutating floor cannot be cleared per-agent, so such a tool always pauses the run in awaiting_approval regardless of the link's flag.
		requireReviewBySlug[slug] = lt.RequireReview || (ok && bt.Mutating)
		if !ok {
			continue // unknown slug (e.g. a tool that has since been removed from the catalog)
		}
		// Only include tools that have a registered handler.
		if _, ok := s.toolRegistry.Get(slug); !ok {
			continue
		}
		toolDefs = append(toolDefs, llm.ToolDefinition{
			Name:        slug,
			Description: bt.Description,
			InputSchema: json.RawMessage(bt.InputSchema),
		})
	}

	// Seed review state for the agent's granted endpoint-tools from its per-agent override so gating applies when they are revealed and called. Default is off (no review) for any slug the override doesn't mark true, mirroring linked built-in tools.
	for slug := range allowedEndpointTools {
		if _, set := requireReviewBySlug[slug]; set {
			continue
		}
		requireReviewBySlug[slug] = endpointReview[slug]
	}

	// Expose the discovery meta-tool only when the agent has been granted at least one endpoint-tool. Agents with no grant never see it.
	if len(allowedEndpointTools) > 0 {
		toolDefs = append(toolDefs, llm.ToolDefinition{
			Name:        agents.SearchAPIToolsSlug,
			Description: agents.SearchAPIToolsDescription,
			InputSchema: json.RawMessage(agents.SearchAPIToolsInputSchema),
		})
	}

	return toolDefs, requireReviewBySlug
}

// tierForTrigger picks a default model tier from how a run was triggered, used only when the agent config doesn't pin a tier of its own. Customer-facing runs (chat/manual) get the high tier; background runs (scheduled/event) get the cheaper balanced tier. Unknown triggers fall back to the default tier.
func tierForTrigger(triggerType string) constants.ModelTier {
	switch constants.AgentTriggerType(triggerType) {
	case constants.AgentTriggerTypeChat, constants.AgentTriggerTypeManual:
		return constants.ModelTierHigh
	case constants.AgentTriggerTypeScheduled, constants.AgentTriggerTypeEvent:
		return constants.ModelTierBalanced
	default:
		return constants.DefaultModelTier
	}
}

// compactionTarget returns the provider+model to use for background context compaction. Summarizing the conversation is cheap background work, so it always runs on the balanced tier rather than the agent's own (possibly frontier) model. Falls back to the supplied run provider/model if the balanced provider isn't registered.
func (s *runnerSvc) compactionTarget(fallbackProvider llm.LLMProvider, fallbackModel string) (llm.LLMProvider, string) {
	compactModel := constants.ModelTierBalanced.ModelChain()[0]
	compactProvider := s.llmProviders[inferProvider(compactModel)]
	if compactProvider == nil {
		return fallbackProvider, fallbackModel
	}
	return compactProvider, compactModel
}

func inferProvider(model string) string {
	switch {
	case strings.HasPrefix(model, "claude-"):
		return "anthropic"
	case strings.HasPrefix(model, "gemini-"):
		return "google"
	case strings.HasPrefix(model, "grok-"):
		return "xai"
	case strings.HasPrefix(model, "gpt-"),
		strings.HasPrefix(model, "codex-"),
		strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3"),
		strings.HasPrefix(model, "o4"):
		return "openai"
	default:
		return "anthropic"
	}
}
