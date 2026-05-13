package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/augno/api/services/agent-service/internal/domain"
	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/agent-service/internal/llm"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/retry"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

const maxToolLoopIterations = 20

var runnerTracer = tracing.GetTracer("agent-service.runner")

// agentConfig is the subset of AgentDefinitionConfig relevant to execution.
type agentConfig struct {
	SystemPrompt string   `json:"system_prompt"`
	Model        string   `json:"model"`
	Provider     string   `json:"provider"`
	Temperature  *float64 `json:"temperature"`
}

type RunnerConfig struct {
	Repos         domain.RepoFactory
	ToolRegistry  *domain.ToolHandlerRegistry
	LLMProviders  map[string]llm.LLMProvider
	OutboxRepo    messaging.OutboxRepo
	CoreClient    domain.CoreClient
	Broker        messaging.MessageBroker
	BillingClient domain.BillingCustomerResolver
}

type runnerSvc struct {
	repos         domain.RepoFactory
	toolRegistry  *domain.ToolHandlerRegistry
	llmProviders  map[string]llm.LLMProvider
	outboxRepo    messaging.OutboxRepo
	coreClient    domain.CoreClient
	broker        messaging.MessageBroker
	billingClient domain.BillingCustomerResolver
}

func NewRunnerSvc(config *RunnerConfig) domain.RunnerSvc {
	if config.Repos == nil {
		panic(fmt.Errorf("runner service: repos is required"))
	}
	if config.ToolRegistry == nil {
		panic(fmt.Errorf("runner service: tool registry is required"))
	}
	if config.LLMProviders == nil {
		panic(fmt.Errorf("runner service: llm providers is required"))
	}
	if config.OutboxRepo == nil {
		panic(fmt.Errorf("runner service: outbox repo is required"))
	}
	if config.BillingClient == nil {
		panic(fmt.Errorf("runner service: billing client is required"))
	}

	return &runnerSvc{
		repos:         config.Repos,
		toolRegistry:  config.ToolRegistry,
		llmProviders:  config.LLMProviders,
		outboxRepo:    config.OutboxRepo,
		coreClient:    config.CoreClient,
		broker:        config.Broker,
		billingClient: config.BillingClient,
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

	// If a cap is set, compute estimated spend for the current month.
	if bc.spendingCapCents != nil {
		monthStart := time.Now().UTC()
		monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)

		tokenRepo := s.repos.NewAgentTokenUsageRepo()
		inputTokens, outputTokens, tokenErr := tokenRepo.GetMonthlyUsage(ctx, billingAccountID, monthStart)
		if tokenErr != nil {
			return nil, fmt.Errorf("failed to query monthly token usage: %w", tokenErr)
		}

		bc.currentSpendCents = llm.EstimateTokenCostCents(int(inputTokens), int(outputTokens), model)

		if bc.currentSpendCents >= *bc.spendingCapCents {
			capDollars := float64(*bc.spendingCapCents) / 100.0
			return nil, fmt.Errorf("monthly agent spending cap of $%.2f has been reached", capDollars)
		}
	}

	return bc, nil
}

func (s *runnerSvc) ExecuteRun(ctx context.Context, runID, configID, accountID, triggerType string) error {
	ctx, span := tracing.StartSpan(ctx, runnerTracer, "service.runner.execute_run")
	defer span.End()

	startTime := time.Now()
	runRepo := s.repos.NewAgentRunRepo()
	configRepo := s.repos.NewAgentConfigRepo()
	defRepo := s.repos.NewAgentDefinitionRepo()

	// Load run
	run, runErr := runRepo.GetByID(ctx, runID)
	if runErr != nil {
		return fmt.Errorf("failed to load run %s: %w", runID, runErr)
	}

	// Load config
	config, cfgErr := configRepo.GetByID(ctx, configID)
	if cfgErr != nil {
		return fmt.Errorf("failed to load config %s: %w", configID, cfgErr)
	}

	// Load definition
	def, defErr := defRepo.GetByID(ctx, config.AgentDefinitionID)
	if defErr != nil {
		return fmt.Errorf("failed to load definition: %w", defErr)
	}

	// Mark as running
	if startErr := runRepo.UpdateStarted(ctx, runID); startErr != nil {
		return fmt.Errorf("failed to mark run as started: %w", startErr)
	}

	// Parse agent config from definition
	var agentCfg agentConfig
	if def.Config != nil {
		if err := json.Unmarshal(def.Config, &agentCfg); err != nil {
			return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("invalid agent config: %s", err.Error()))
		}
	}

	// Resolve provider and model
	modelName := agentCfg.Model
	if modelName == "" {
		modelName = domain.DefaultModel
	}
	if !domain.AllowedModels[modelName] {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("model %q is not allowed", modelName))
	}
	providerName := agentCfg.Provider
	if providerName == "" {
		providerName = inferProvider(modelName)
	}

	// Resolve billing context and inject Stripe customer ID
	bc, billingErr := s.resolveBillingContext(ctx, accountID, modelName)
	if billingErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, billingErr.Error())
	}
	ctx = llm.WithStripeCustomerID(ctx, bc.stripeCustomerID)

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

	// Load linked tools from DB
	toolRepo := s.repos.NewAgentDefinitionToolRepo()
	linkedTools, toolErr := toolRepo.ListByAgentDefinitionID(ctx, def.ID)
	if toolErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to load tools: %s", toolErr.Error()))
	}

	// Build require_review lookup map
	requireReviewBySlug := make(map[string]bool, len(linkedTools))
	for _, lt := range linkedTools {
		slug := agentdb.StringFromPgText(lt.ToolSlug)
		if slug != nil && *slug != "" {
			requireReviewBySlug[*slug] = lt.RequireReview
		}
	}

	// Build tool definitions and resolve handlers
	var toolDefs []llm.ToolDefinition
	for _, lt := range linkedTools {
		slug := agentdb.StringFromPgText(lt.ToolSlug)
		if slug == nil || *slug == "" {
			continue
		}
		// Only include tools that have a registered handler
		if _, ok := s.toolRegistry.Get(*slug); !ok {
			continue
		}
		if lt.ToolInputSchema == nil {
			continue
		}
		desc := ""
		if d := agentdb.StringFromPgText(lt.ToolDescription); d != nil {
			desc = *d
		}
		toolDefs = append(toolDefs, llm.ToolDefinition{
			Name:        *slug,
			Description: desc,
			InputSchema: lt.ToolInputSchema,
		})
	}

	// Only include built-in tools that are explicitly linked to this agent definition.
	// Built-in tools not linked to the agent are intentionally omitted so the LLM
	// does not believe it has access to tools the agent has not been set up with.
	toolDefs = s.appendLinkedBuiltinToolDefs(toolDefs, linkedTools)

	// Resolve temperature
	temperature := 0.0
	if agentCfg.Temperature != nil {
		temperature = *agentCfg.Temperature
	}

	// Execute the agent
	result, err := s.executeAgent(ctx, run, config, def, accountID, agentIdentity, agentCfg.SystemPrompt, modelName, providerName, toolDefs, temperature, requireReviewBySlug, bc)
	if err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, err.Error())
	}

	// Persist outputs
	if err := s.persistOutputs(ctx, runID, accountID, bc.billingAccountID, result); err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to persist outputs: %s", err.Error()))
	}

	// Finalize run
	durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
	statusCode := domain.RunStatusCompleted
	if result.AwaitingApproval {
		statusCode = domain.RunStatusAwaitingApproval
	} else if triggerType == domain.TriggerManual {
		statusCode = domain.RunStatusAwaitingInput
	}
	if completeErr := runRepo.UpdateCompleted(ctx, sqlc.UpdateAgentRunCompletedParams{
		StatusCode:        statusCode,
		Output:            result.Output,
		DurationMs:        agentdb.PgInt4(durationMs),
		TotalInputTokens:  int64(result.InputTokens),
		TotalOutputTokens: int64(result.OutputTokens),
		ID:                runID,
	}); completeErr != nil {
		return fmt.Errorf("failed to finalize run: %w", completeErr)
	}

	// Write outbox message for run completion
	s.writeRunCompletedEvent(ctx, runID, accountID, bc.billingAccountID, result)

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

func (s *runnerSvc) emitEvent(ctx context.Context, runID, accountID string, seq *int, stepType, title string, content *string, durationMs *int32, actionID *string, metadata json.RawMessage) {
	// Extract actor from context identity
	var actorID, actorType, actorName string
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil {
		actorType = string(identity.Type)
		if identity.Actor != nil {
			actorID = identity.Actor.ID
			if identity.Actor.Name != nil {
				actorName = *identity.Actor.Name
			}
		}
	}
	s.emitEventAs(ctx, runID, accountID, seq, stepType, title, content, durationMs, actionID, metadata, actorID, actorType, actorName)
}

func (s *runnerSvc) emitEventAs(ctx context.Context, runID, accountID string, seq *int, stepType, title string, content *string, durationMs *int32, actionID *string, metadata json.RawMessage, actorID, actorType, actorName string) {
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

func (s *runnerSvc) executeAgent(
	ctx context.Context,
	run *sqlc.AgentRun,
	config *sqlc.AgentConfig,
	def *sqlc.AgentDefinition,
	accountID string,
	identity *types.Identity,
	systemPrompt string,
	modelName string,
	providerName string,
	toolDefs []llm.ToolDefinition,
	temperature float64,
	requireReviewBySlug map[string]bool,
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

	// Build initial messages from run input
	var inputText string
	hasUserMessage := false
	if run.Input != nil {
		var inputData struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(run.Input, &inputData); err == nil && inputData.Message != "" {
			inputText = inputData.Message
			hasUserMessage = true
		} else if raw := strings.TrimSpace(string(run.Input)); raw != "" && raw != "{}" {
			inputText = raw
			hasUserMessage = true
		}
	}
	if inputText == "" {
		inputText = "Process the input and take appropriate action."
	}

	// Emit trigger_received event
	s.emitEvent(ctx, run.ID, accountID, &seq, "trigger_received", "Run triggered", &inputText, nil, nil, nil)

	// Only emit user_message event if the client provided an actual message
	if hasUserMessage {
		s.emitEvent(ctx, run.ID, accountID, &seq, "user_message", "User message", &inputText, nil, nil, nil)
	}

	messages := []llm.Message{
		{Role: "user", Content: inputText},
	}

	// Create run context for handler
	runCtx := &domain.HandlerRunContext{
		AccountID:            accountID,
		RunID:                run.ID,
		Definition:           def,
		Config:               config,
		Repos:                s.repos,
		CoreClient:           s.coreClient,
		Identity:             identity,
		RequireReviewBySlug:  requireReviewBySlug,
		AlwaysAllowedSlugs:   make(map[string]bool),
		OneTimeApprovedSlugs: make(map[string]bool),
	}

	return s.runAgentLoop(ctx, run, accountID, identity, systemPrompt, modelName, providerName, toolDefs, temperature, messages, &seq, runCtx, bc.spendingCapCents, bc.currentSpendCents)

}

// runAgentLoop is the core agentic tool loop shared by executeAgent and ContinueRun.
func (s *runnerSvc) runAgentLoop(
	ctx context.Context,
	run *sqlc.AgentRun,
	accountID string,
	identity *types.Identity,
	systemPrompt string,
	modelName string,
	providerName string,
	toolDefs []llm.ToolDefinition,
	temperature float64,
	messages []llm.Message,
	seq *int,
	runCtx *domain.HandlerRunContext,
	spendingCapCents *int64,
	currentMonthSpendCents int64,
) (*domain.RunResult, error) {
	provider, ok := s.llmProviders[providerName]
	if !ok {
		return nil, fmt.Errorf("LLM provider %q not configured", providerName)
	}

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

		// Proactive compaction: check estimated context usage BEFORE the LLM call.
		if llm.NeedsProactiveCompaction(systemPrompt, messages, toolDefs, modelName) {
			messages = llm.CopyMessages(messages)
			freed := pruneOldToolResults(messages)
			slog.Info("Proactive compaction: pruned old tool results",
				"run_id", run.ID, "tokens_freed", freed)

			if llm.NeedsProactiveCompaction(systemPrompt, messages, toolDefs, modelName) {
				summary, compactErr := compactMessages(ctx, provider, modelName, systemPrompt, messages)
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

		// Truncate messages to fit within the model's context window.
		truncatedMessages := llm.TruncateMessages(systemPrompt, messages, toolDefs, modelName)

		llmReq := &llm.ToolRequest{
			Model:       modelName,
			System:      systemPrompt,
			Messages:    truncatedMessages,
			Tools:       toolDefs,
			MaxTokens:   4096,
			Temperature: temperature,
		}

		var resp *llm.ToolResponse
		var llmErr error

		callLLM := func() (*llm.ToolResponse, error) {
			if sp, ok := provider.(llm.StreamingLLMProvider); ok {
				var deltaSeq int
				var deltaBuf strings.Builder
				var lastFlush time.Time

				flushDelta := func() {
					if deltaBuf.Len() == 0 {
						return
					}
					s.emitDelta(ctx, run.ID, accountID, deltaSeq, deltaBuf.String())
					deltaSeq++
					deltaBuf.Reset()
					lastFlush = time.Now()
				}

				r, err := sp.StreamCompleteWithTools(ctx, llmReq, func(ev llm.StreamEvent) {
					if ev.Type != "content_delta" || ev.ContentDelta == "" {
						return
					}
					deltaBuf.WriteString(ev.ContentDelta)
					if deltaBuf.Len() >= 20 || time.Since(lastFlush) >= 50*time.Millisecond {
						flushDelta()
					}
				})
				flushDelta()
				return r, err
			}
			return provider.CompleteWithTools(ctx, llmReq)
		}

		resp, llmErr = callLLM()
		if llmErr != nil {
			// Retry only if the error is a retryable GatewayError.
			var gatewayErr *llm.GatewayError
			if errors.As(llmErr, &gatewayErr) && gatewayErr.Retryable {
				retried := false
				retryCfgForCall := *retryCfg
				// Honor Retry-After header if present.
				if ra := gatewayErr.RetryAfter(); ra > 0 && ra < retryCfgForCall.MaxWait {
					retryCfgForCall.InitialWait = ra
				}

				retryErr := retry.WithBackoff(ctx, &retryCfgForCall, func() error {
					var retryResp *llm.ToolResponse
					var err error
					retryResp, err = callLLM()
					if err != nil {
						var ge *llm.GatewayError
						if errors.As(err, &ge) && ge.Retryable {
							return err // continue retrying
						}
						llmErr = err // non-retryable, stop retrying
						return nil   // break out of retry loop
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
					retryMeta, _ := json.Marshal(map[string]any{
						"original_status": gatewayErr.StatusCode,
						"resolved":        true,
					})
					s.emitEvent(ctx, run.ID, accountID, seq, "retry", "LLM call succeeded after retry", nil, nil, nil, retryMeta)
				}
			}
		}
		if llmErr != nil {
			s.emitEvent(ctx, run.ID, accountID, seq, "error", "LLM call failed", new(llmErr.Error()), nil, nil, nil)
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
			s.emitEvent(ctx, run.ID, accountID, seq, "cap_exceeded", capMsg, nil, nil, nil, nil)

			outputJSON, _ := json.Marshal(map[string]string{"response": capMsg})
			return &domain.RunResult{
				Output:       outputJSON,
				Actions:      runCtx.Actions,
				Artifacts:    runCtx.Artifacts,
				Memories:     runCtx.Memories,
				Alerts:       runCtx.Alerts,
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				LLMProvider:  providerName,
				LLMModel:     modelName,
			}, nil
		}

		if resp.StopReason != "tool_use" || len(resp.ToolCalls) == 0 {
			// Emit assistant_message event
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
				Memories:     runCtx.Memories,
				Alerts:       runCtx.Alerts,
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				LLMProvider:  providerName,
				LLMModel:     modelName,
			}, nil
		}

		// Emit thinking event
		iterDurationMs := safeconv.Int64ToInt32(time.Since(iterStart).Milliseconds())
		thinkingMeta, _ := json.Marshal(map[string]any{
			"input_tokens":  resp.InputTokens,
			"output_tokens": resp.OutputTokens,
		})
		if resp.Content != "" {
			s.emitEvent(ctx, run.ID, accountID, seq, "thinking", "Reasoning", new(resp.Content), &iterDurationMs, nil, thinkingMeta)
		}

		// Process tool calls
		assistantMsg := llm.Message{
			Role:    "assistant",
			Content: resp.Content,
		}
		for _, tc := range resp.ToolCalls {
			assistantMsg.ToolUse = append(assistantMsg.ToolUse, llm.ToolUseBlock(tc))
		}
		messages = append(messages, assistantMsg)

		toolResultMsg := llm.Message{Role: "user"}
		toolsBlocked := false
		for _, tc := range resp.ToolCalls {
			// Emit tool_call event
			toolCallMeta, _ := json.Marshal(map[string]any{"tool_use_id": tc.ID, "tool_name": tc.Name, "input": tc.Input})
			s.emitEvent(ctx, run.ID, accountID, seq, "tool_call", tc.Name, nil, nil, nil, toolCallMeta)

			// Guard: block tools that require human approval unless explicitly approved
			if runCtx.RequireReviewBySlug[tc.Name] && !runCtx.AlwaysAllowedSlugs[tc.Name] && !runCtx.OneTimeApprovedSlugs[tc.Name] {
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

			// Consume one-time approval so the tool requires re-approval on next invocation
			delete(runCtx.OneTimeApprovedSlugs, tc.Name)

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

		// Context compaction: if we're approaching the model's limit, prune then summarize.
		if needsCompaction(resp.InputTokens, modelName) {
			// Deep copy messages for mutation during pruning.
			messages = llm.CopyMessages(messages)
			freed := pruneOldToolResults(messages)
			slog.Info("Context compaction: pruned old tool results",
				"run_id", run.ID, "tokens_freed", freed, "input_tokens", resp.InputTokens)

			// If pruning wasn't enough, trigger LLM-based summarization.
			if llm.EstimateAllMessages(messages) >= (resp.InputTokens - compactionBuffer) {
				summary, compactErr := compactMessages(ctx, provider, modelName, systemPrompt, messages)
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
			for _, tc := range resp.ToolCalls {
				if runCtx.RequireReviewBySlug[tc.Name] && !runCtx.AlwaysAllowedSlugs[tc.Name] && !runCtx.OneTimeApprovedSlugs[tc.Name] {
					blockedTools = append(blockedTools, tc.Name)
				}
			}
			approvalMeta, _ := json.Marshal(map[string]any{
				"blocked_tools":     blockedTools,
				"totalInputTokens":  totalInputTokens,
				"totalOutputTokens": totalOutputTokens,
			})
			approvalMsg := fmt.Sprintf("Run paused — the following tools require human approval before execution: %s", strings.Join(blockedTools, ", "))
			s.emitEvent(ctx, run.ID, accountID, seq, "awaiting_approval", "Waiting for tool approval", new(approvalMsg), nil, nil, approvalMeta)

			outputJSON, _ := json.Marshal(map[string]string{"response": approvalMsg})
			return &domain.RunResult{
				Output:           outputJSON,
				Actions:          runCtx.Actions,
				Artifacts:        runCtx.Artifacts,
				Memories:         runCtx.Memories,
				Alerts:           runCtx.Alerts,
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
		Memories:     runCtx.Memories,
		Alerts:       runCtx.Alerts,
		InputTokens:  totalInputTokens,
		OutputTokens: totalOutputTokens,
		LLMProvider:  providerName,
		LLMModel:     modelName,
	}, nil
}

func (s *runnerSvc) ContinueRun(ctx context.Context, runID, accountID, message string, approvedToolSlugs []string, allowedToolSlugs []string, actorID, actorType, actorName string) error {
	ctx, span := tracing.StartSpan(ctx, runnerTracer, "service.runner.continue_run")
	defer span.End()

	startTime := time.Now()
	runRepo := s.repos.NewAgentRunRepo()
	configRepo := s.repos.NewAgentConfigRepo()
	defRepo := s.repos.NewAgentDefinitionRepo()
	eventRepo := s.repos.NewAgentRunEventRepo()

	// Load run
	run, runErr := runRepo.GetByID(ctx, runID)
	if runErr != nil {
		return fmt.Errorf("failed to load run %s: %w", runID, runErr)
	}

	// Validate status — the gRPC handler already transitions awaiting_input → running
	if run.StatusCode != domain.RunStatusRunning {
		return fmt.Errorf("run %s is not in running state (status: %s)", runID, run.StatusCode)
	}

	// Load config
	configID := ""
	if run.AgentConfigID.Valid {
		configID = run.AgentConfigID.String
	}
	if configID == "" {
		return fmt.Errorf("run %s has no agent config ID", runID)
	}
	config, cfgErr := configRepo.GetByID(ctx, configID)
	if cfgErr != nil {
		return fmt.Errorf("failed to load config %s: %w", configID, cfgErr)
	}

	// Load definition
	def, defErr := defRepo.GetByID(ctx, config.AgentDefinitionID)
	if defErr != nil {
		return fmt.Errorf("failed to load definition: %w", defErr)
	}

	// Parse agent config
	var agentCfg agentConfig
	if def.Config != nil {
		if err := json.Unmarshal(def.Config, &agentCfg); err != nil {
			return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("invalid agent config: %s", err.Error()))
		}
	}

	// Resolve provider and model
	modelName := agentCfg.Model
	if modelName == "" {
		modelName = domain.DefaultModel
	}
	if !domain.AllowedModels[modelName] {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("model %q is not allowed", modelName))
	}
	providerName := agentCfg.Provider
	if providerName == "" {
		providerName = inferProvider(modelName)
	}

	// Resolve billing context and inject Stripe customer ID
	bc, billingErr := s.resolveBillingContext(ctx, accountID, modelName)
	if billingErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, billingErr.Error())
	}
	ctx = llm.WithStripeCustomerID(ctx, bc.stripeCustomerID)

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

	// Load linked tools
	toolRepo := s.repos.NewAgentDefinitionToolRepo()
	linkedTools, toolErr := toolRepo.ListByAgentDefinitionID(ctx, def.ID)
	if toolErr != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to load tools: %s", toolErr.Error()))
	}

	// Build require_review lookup map
	requireReviewBySlug := make(map[string]bool, len(linkedTools))
	for _, lt := range linkedTools {
		slug := agentdb.StringFromPgText(lt.ToolSlug)
		if slug != nil && *slug != "" {
			requireReviewBySlug[*slug] = lt.RequireReview
		}
	}

	var toolDefs []llm.ToolDefinition
	for _, lt := range linkedTools {
		slug := agentdb.StringFromPgText(lt.ToolSlug)
		if slug == nil || *slug == "" {
			continue
		}
		if _, ok := s.toolRegistry.Get(*slug); !ok {
			continue
		}
		if lt.ToolInputSchema == nil {
			continue
		}
		desc := ""
		if d := agentdb.StringFromPgText(lt.ToolDescription); d != nil {
			desc = *d
		}
		toolDefs = append(toolDefs, llm.ToolDefinition{
			Name:        *slug,
			Description: desc,
			InputSchema: lt.ToolInputSchema,
		})
	}
	// Only include built-in tools that are explicitly linked to this agent
	toolDefs = s.appendLinkedBuiltinToolDefs(toolDefs, linkedTools)

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

	// Append new user message and emit event with the caller's actor identity
	messages = append(messages, llm.Message{Role: "user", Content: message})
	s.emitEventAs(ctx, run.ID, accountID, &seq, "user_message", "User message", &message, nil, nil, nil, actorID, actorType, actorName)

	// Build one-time approved slugs from pending_review actions.
	// If approvedToolSlugs is provided, only approve matching tools (per-tool approval).
	// If empty, approve all pending tools (backward compat / "Approve All").
	oneTimeApproved := make(map[string]bool)
	actionRepo := s.repos.NewAgentActionRepo()
	existingActions, actionsErr := actionRepo.ListByRun(ctx, runID)
	if actionsErr == nil {
		approveSet := make(map[string]bool, len(approvedToolSlugs))
		for _, slug := range approvedToolSlugs {
			approveSet[slug] = true
		}
		selectiveApproval := len(approvedToolSlugs) > 0

		for _, a := range existingActions {
			if a.StatusCode == domain.ActionStatusPendingReview {
				if selectiveApproval && !approveSet[a.ToolSlug] {
					continue
				}
				oneTimeApproved[a.ToolSlug] = true
				_ = actionRepo.UpdateStatus(ctx, sqlc.UpdateAgentActionStatusParams{
					ID:         a.ID,
					StatusCode: domain.ActionStatusApproved,
				})
			}
		}
	}

	// Merge allowed tool slugs: combine previously persisted + newly allowed.
	// Allowed tools bypass review for the rest of this run.
	var existingAllowed []string
	if run.AllowedToolSlugs != nil {
		_ = json.Unmarshal(run.AllowedToolSlugs, &existingAllowed)
	}
	alwaysAllowed := make(map[string]bool, len(existingAllowed)+len(allowedToolSlugs))
	for _, slug := range existingAllowed {
		alwaysAllowed[slug] = true
	}
	for _, slug := range allowedToolSlugs {
		alwaysAllowed[slug] = true
	}

	// Persist merged allowed tool slugs if there are new additions.
	if len(allowedToolSlugs) > 0 {
		allSlugs := make([]string, 0, len(alwaysAllowed))
		for slug := range alwaysAllowed {
			allSlugs = append(allSlugs, slug)
		}
		slugsJSON, _ := json.Marshal(allSlugs)
		runRepo.UpdateAllowedToolSlugs(ctx, runID, slugsJSON)
	}

	// Remove one-time approvals that are already always-allowed (always-allow supersedes).
	for slug := range alwaysAllowed {
		delete(oneTimeApproved, slug)
	}

	// Create run context
	runCtx := &domain.HandlerRunContext{
		AccountID:            accountID,
		RunID:                run.ID,
		Definition:           def,
		Config:               config,
		Repos:                s.repos,
		CoreClient:           s.coreClient,
		Identity:             agentIdentity,
		RequireReviewBySlug:  requireReviewBySlug,
		AlwaysAllowedSlugs:   alwaysAllowed,
		OneTimeApprovedSlugs: oneTimeApproved,
	}

	// Execute the agent loop
	result, err := s.runAgentLoop(ctx, run, accountID, agentIdentity, systemPrompt, modelName, providerName, toolDefs, temperature, messages, &seq, runCtx, bc.spendingCapCents, bc.currentSpendCents)
	if err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, err.Error())
	}

	// Persist outputs
	if err := s.persistOutputs(ctx, runID, accountID, bc.billingAccountID, result); err != nil {
		return s.failRun(ctx, runRepo, runID, startTime, fmt.Sprintf("failed to persist outputs: %s", err.Error()))
	}

	// Finalize run - manual runs go back to awaiting_input unless tools need approval.
	// Accumulate token counts from previous continuations.
	durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
	statusCode := domain.RunStatusAwaitingInput
	if result.AwaitingApproval {
		statusCode = domain.RunStatusAwaitingApproval
	}
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
		return fmt.Errorf("failed to finalize run: %w", completeErr)
	}

	// Write outbox message for run completion
	s.writeRunCompletedEvent(ctx, runID, accountID, bc.billingAccountID, result)

	slog.Info("Agent continue run completed",
		"run_id", runID,
		"duration_ms", durationMs,
		"input_tokens", result.InputTokens,
		"output_tokens", result.OutputTokens,
	)

	return nil
}

// reconstructMessages rebuilds LLM message history from stored events.
func reconstructMessages(events []sqlc.AgentRunEvent) []llm.Message {
	var messages []llm.Message
	var pendingAssistant *llm.Message

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

		case "tool_result", "tool_blocked":
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
			// For blocked tool events, use the event content as the result
			if event.StepType == "tool_blocked" && fullResult == "" && event.Content.Valid {
				fullResult = event.Content.String
			}

			// Find or create the last user message with tool results
			if len(messages) > 0 {
				last := &messages[len(messages)-1]
				if last.Role == "user" && len(last.ToolResults) > 0 {
					last.ToolResults = append(last.ToolResults, llm.ToolResultBlock{
						ToolUseID: toolUseID,
						Content:   fullResult,
						IsError:   isError,
					})
					continue
				}
			}
			messages = append(messages, llm.Message{
				Role: "user",
				ToolResults: []llm.ToolResultBlock{
					{
						ToolUseID: toolUseID,
						Content:   fullResult,
						IsError:   isError,
					},
				},
			})
		}
	}

	// Flush any remaining pending assistant
	if pendingAssistant != nil {
		messages = append(messages, *pendingAssistant)
	}

	return messages
}

func (s *runnerSvc) handleToolCall(ctx context.Context, tc llm.ToolCall, runCtx *domain.HandlerRunContext) (string, error) {
	handler, ok := s.toolRegistry.Get(tc.Name)
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", tc.Name), nil
	}
	return handler(ctx, tc.Input, runCtx)
}

func (s *runnerSvc) persistOutputs(ctx context.Context, runID, accountID, billingAccountID string, result *domain.RunResult) error {
	ctx, span := tracing.StartSpan(ctx, runnerTracer, "service.runner.persist_outputs")
	defer span.End()

	actionRepo := s.repos.NewAgentActionRepo()
	memoryRepo := s.repos.NewAgentMemoryRepo()
	alertRepo := s.repos.NewAgentAlertRepo()
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

	// Persist memories
	for _, m := range result.Memories {
		memoryID, err := id.GenID(id.AgentMemoryIDPrefix, nil)
		if err != nil {
			return fmt.Errorf("failed to generate memory ID: %w", err)
		}

		if insertErr := memoryRepo.Insert(ctx, sqlc.InsertAgentMemoryParams{
			ID:         memoryID,
			AccountID:  accountID,
			Category:   m.Category,
			Content:    m.Content,
			Metadata:   m.Metadata,
			EntityType: agentdb.PgText(m.EntityType),
			EntityID:   agentdb.PgText(m.EntityID),
			Importance: m.Importance,
		}); insertErr != nil {
			return fmt.Errorf("failed to insert memory: %w", insertErr)
		}
	}

	// Persist alerts
	for _, a := range result.Alerts {
		alertID, err := id.GenID(id.AgentAlertIDPrefix, nil)
		if err != nil {
			return fmt.Errorf("failed to generate alert ID: %w", err)
		}

		if insertErr := alertRepo.Insert(ctx, sqlc.InsertAgentAlertParams{
			ID:           alertID,
			AccountID:    accountID,
			AgentRunID:   agentdb.PgText(runID),
			SeverityCode: a.SeverityCode,
			StatusCode:   domain.AlertStatusOpen,
			Title:        a.Title,
			Message:      agentdb.PgText(a.Message),
			Metadata:     a.Metadata,
		}); insertErr != nil {
			return fmt.Errorf("failed to insert alert: %w", insertErr)
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
}

func (s *runnerSvc) failRun(ctx context.Context, runRepo domain.AgentRunRepo, runID string, startTime time.Time, errMsg string) error {
	durationMs := safeconv.Int64ToInt32(time.Since(startTime).Milliseconds())
	if failErr := runRepo.UpdateFailed(ctx, sqlc.UpdateAgentRunFailedParams{
		ErrorMessage: agentdb.PgText(errMsg),
		DurationMs:   agentdb.PgInt4(durationMs),
		ID:           runID,
	}); failErr != nil {
		slog.Error("Failed to mark run as failed", "error", failErr, "run_id", runID)
	}
	return fmt.Errorf("agent run failed: %s", errMsg)
}

type builtinToolEntry struct {
	slug        string
	description string
	inputSchema json.RawMessage
}

func builtinToolDefs() []builtinToolEntry {
	return []builtinToolEntry{
		{
			slug:        "save_memory",
			description: "Save an observation about a customer or product for future reference.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"category":{"type":"string","description":"Memory category (e.g., customer_preference, ordering_pattern, product_alias)"},"content":{"type":"string","description":"The observation to remember"},"entity_type":{"type":"string","description":"Type of entity this relates to (e.g., account_relation, product)"},"entity_id":{"type":"string","description":"ID of the related entity"},"importance":{"type":"number","description":"Importance score from 0.0 to 1.0"}},"required":["category","content","importance"]}`),
		},
		{
			slug:        "create_alert",
			description: "Create an alert that requires human attention.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"severity":{"type":"string","enum":["info","warning","urgent","critical"],"description":"Alert severity level"},"title":{"type":"string","description":"Short alert title"},"message":{"type":"string","description":"Detailed alert message"}},"required":["severity","title","message"]}`),
		},
		{
			slug:        "search_products",
			description: "Search for products by keyword or phrase.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query for products"}},"required":["query"]}`),
		},
		{
			slug:        "list_products",
			description: "List all products in the account catalog.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			slug:        "lookup_customer",
			description: "Look up a customer by their email address.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"email":{"type":"string","description":"Customer email address"}},"required":["email"]}`),
		},
		{
			slug:        "create_artifact",
			description: "Create an artifact such as a report, document, or data export.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"artifact_type":{"type":"string","description":"Type of artifact (e.g., report, document, csv)"},"name":{"type":"string","description":"Artifact name"},"content":{"type":"string","description":"Artifact content"},"mime_type":{"type":"string","description":"MIME type of the content (e.g., text/plain, text/csv, application/json)"}},"required":["artifact_type","name","content","mime_type"]}`),
		},
		{
			slug:        "update_memory",
			description: "Update an existing memory entry.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"memory_id":{"type":"string","description":"ID of the memory to update"},"category":{"type":"string","description":"Memory category"},"content":{"type":"string","description":"Updated memory content"},"importance":{"type":"number","description":"Importance score from 0.0 to 1.0"},"entity_type":{"type":"string","description":"Type of entity this relates to"},"entity_id":{"type":"string","description":"ID of the related entity"}},"required":["memory_id","category","content","importance"]}`),
		},
		{
			slug:        "delete_memory",
			description: "Delete a memory entry that is no longer relevant.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"memory_id":{"type":"string","description":"ID of the memory to delete"}},"required":["memory_id"]}`),
		},
		{
			slug: "read_doc",
			description: "Read the content of an Augno documentation page. " +
				"To find the right page, first fetch https://docs.augno.com/llms.txt which lists all available pages with descriptions. " +
				"Then call this tool again with the URL of the page you want to read.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"The full URL of the documentation page to read (must be from docs.augno.com). Start with https://docs.augno.com/llms.txt to discover available pages."}},"required":["url"]}`),
		},
		{
			slug: "fetch_url",
			description: "Fetch the content of a public URL. Returns the response body as text. Only HTTPS URLs are allowed. " +
				"When fetching a website for the first time, check if the site has an /llms.txt file (e.g. https://example.com/llms.txt). " +
				"This file follows the llms.txt standard and lists markdown-formatted URLs optimized for LLM consumption. " +
				"If llms.txt exists and contains relevant URLs, prefer fetching those markdown URLs instead of the raw HTML pages. " +
				"Also look for llms-full.txt for comprehensive content. Skip the llms.txt check for direct links to files, APIs, or non-website URLs.",
			inputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"The HTTPS URL to fetch"}},"required":["url"]}`),
		},
	}
}

// appendLinkedBuiltinToolDefs adds built-in tool definitions only when they
// appear in the agent's linked tools but were skipped during the main loop
// (e.g. because they had no ToolInputSchema in the DB row). This ensures
// agents only see built-in tools that have been explicitly assigned to them.
func (s *runnerSvc) appendLinkedBuiltinToolDefs(toolDefs []llm.ToolDefinition, linkedTools []sqlc.ListToolsByAgentDefinitionIDRow) []llm.ToolDefinition {
	addedSlugs := make(map[string]bool, len(toolDefs))
	for _, td := range toolDefs {
		addedSlugs[td.Name] = true
	}

	// Build set of slugs that are linked to this agent definition
	linkedSlugs := make(map[string]bool, len(linkedTools))
	for _, lt := range linkedTools {
		slug := agentdb.StringFromPgText(lt.ToolSlug)
		if slug != nil && *slug != "" {
			linkedSlugs[*slug] = true
		}
	}

	for _, bt := range builtinToolDefs() {
		if addedSlugs[bt.slug] {
			continue
		}
		if !linkedSlugs[bt.slug] {
			continue
		}
		if _, ok := s.toolRegistry.Get(bt.slug); ok {
			toolDefs = append(toolDefs, llm.ToolDefinition{
				Name:        bt.slug,
				Description: bt.description,
				InputSchema: bt.inputSchema,
			})
		}
	}
	return toolDefs
}

func inferProvider(model string) string {
	switch {
	case strings.HasPrefix(model, "claude-"):
		return "anthropic"
	case strings.HasPrefix(model, "gpt-"), strings.HasPrefix(model, "o1"), strings.HasPrefix(model, "o3"):
		return "openai"
	default:
		return "anthropic"
	}
}
