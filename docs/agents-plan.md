# *Agent Platform Architecture Plan*

## *Context*

*Augno needs an AI agent system to automate operational workflows: processing inbound emails into sales orders, monitoring customer ordering cadence, settling AR accounts, generating production schedules from MRP data, providing trend digests, and conducting sales research. No AI/LLM code exists today. The platform has a mature Go microservices backend (api/) and an Express.js/Next.js dashboard (dashboard/) sharing a PlanetScale MySQL database.*

*The architecture must support:*

- *6+ distinct agent types with different triggers (email, schedule, manual)*
- *Human review workflow (review-first by default, configurable auto-approval)*
- *Agent memories scoped to entities (customers, addresses, users)*
- *Side-by-side source-of-truth views for agent actions*
- *Token usage metering with a certain number of free tokens for the paid tiers (starter, pro, enterprise)*
- *Inbound email via AWS SES*
- *Multi-provider LLM support (Claude, OpenAI, etc.)*

---

## *Architecture Overview*

```
                           ┌─────────────────┐
                           │  AWS SES Inbound │
                           │  → S3 → SNS     │
                           └────────┬────────┘
                                    │ webhook
┌──────────────┐          ┌────────▼────────┐          ┌──────────────────┐
│  Dashboard   │◄── gRPC ─┤  api-gateway    ├── gRPC ──►  agent-service   │
│  (Next.js)   │          └────────┬────────┘          │  (new Go svc)    │
└──────┬───────┘                   │                   │                  │
       │                    ┌──────▼──────┐            │  - Scheduler     │
       │ HTTP               │  MySQL      │            │  - LLM Executor  │
       │                    │  (shared)   │            │  - Run Engine    │
┌──────▼───────┐            │  ┌────────┐ │            └───────┬──────────┘
│  Dashboard   │────────────┤  │ outbox │ │                    │
│  API (Express)│           │  │ inbox  │ │◄───────────────────┘
│              │            │  └───┬────┘ │     writes outbox in same
│  - Agent CRUD│            └─────┼───────┘     transaction as domain data
│  - Review UX │                  │
│  - Execute   │            ┌─────▼───────┐
└──────┬───────┘            │  RabbitMQ   │  Enqueuer polls outbox →
       │                    │             │  publishes to RabbitMQ →
       └────────────────────┤  Queues:    │  consumers use inbox dedup
                            │  execute_run│
                            │  process_email│
                            │  execute_action│
                            │  run_completed│
                            └─────────────┘
```

*All async flows use the transactional outbox/inbox pattern:*

1. *Business write + outbox insert in same DB transaction (never loses a message)*
2. *Enqueuer goroutine polls outbox → publishes to RabbitMQ (with retry + lock expiry)*
3. *Consumer wraps handler with InboxConsumer.Wrap() (dedup by message_id + handler)*
4. *Crash recovery: inbox detects "received" but not "processed" → safe retry*

***Split responsibility:***

- *agent-service (Go): Scheduling (writes outbox), LLM execution (consumes run messages), email parsing (consumes email messages), producing proposed actions (writes outbox for execution)*
- *Dashboard API (Express): Agent management CRUD, review workflow (writes outbox for approved actions), consuming execution commands (calls existing OrderSvc/InvoiceSvc), token metering to Stripe*
- *RabbitMQ + outbox/inbox: Durable bridge — every cross-service operation goes through the outbox table for at-least-once delivery with deduplication*

---

## *1. New agent-service (Go)*

***Location:*** `api/services/agent-service/`

*Follows the existing service structure:*

```
services/agent-service/
├── cmd/
│   ├── main.go
│   ├── run.go              # DB + RabbitMQ + scheduler + gRPC init
│   └── config.go           # ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.
├── internal/
│   ├── service/
│   │   ├── runner.go        # Agent run lifecycle engine
│   │   └── scheduler.go     # Periodic run scheduler
│   ├── domain/
│   │   ├── types.go         # Agent domain types
│   │   ├── interfaces.go    # LLM provider, agent handler interfaces
│   │   └── mocks/
│   ├── llm/
│   │   ├── provider.go      # LLMProvider interface
│   │   ├── anthropic.go     # Claude implementation
│   │   ├── openai.go        # OpenAI implementation
│   │   └── types.go         # Message, Tool, ToolResult types
│   ├── agents/
│   │   ├── handler.go       # AgentHandler interface
│   │   ├── email_order.go   # Agent 1: Email → Sales Order
│   │   ├── cadence.go       # Agent 2: Customer cadence monitoring
│   │   ├── ar_email.go      # Agent 3: AR email processing
│   │   ├── mrp.go           # Agent 4: MRP → Production schedule
│   │   ├── trends.go        # Agent 5: Trends/metrics digest
│   │   └── sales_research.go # Agent 6: Sales research
│   ├── email/
│   │   ├── ingest.go        # SES S3 email fetching + parsing
│   │   └── parser.go        # MIME parsing, attachment extraction
│   ├── infrastructure/
│   │   ├── repository/      # sqlc repos for agent tables + read-only business data
│   │   ├── grpc/            # gRPC handler + clients
│   │   └── queries/         # SQL queries
│   └── event/
│       ├── email_consumer.go     # Consumes inbound email notifications
│       └── schedule_consumer.go  # Consumes scheduled run triggers
└── pkg/
    └── types/               # Shared types for other services
```

***Reference files for service template:***

- `api/services/billing-service/cmd/run.go` *— initialization pattern*
- `api/shared/messaging/enqueuer.go` *— background polling pattern*
- `api/shared/contracts/amqp.go` *— message envelope and routing keys*

---

## *2. Database Schema*

***New SQL tables** (in* `api/shared/db/migrations/`*)*

*All tables are account-scoped for multi-tenancy. The agent-service reads/writes these via sqlc. The dashboard API accesses them via Prisma (models added to the Prisma schema).*

### *agent_definition — Global catalog of agent types*

```sql
CREATE TABLE agent_definition (
    id              VARCHAR(255) NOT NULL PRIMARY KEY,
    code            VARCHAR(255) NOT NULL UNIQUE,          -- 'email_order', 'cadence', 'ar_email', etc.
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    category        VARCHAR(255) NOT NULL,                 -- 'sales', 'finance', 'production', 'analytics'
    version         VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    created_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
);
```

### *agent_config — Per-account agent enablement and settings*

```sql
CREATE TABLE agent_config (
    id                          VARCHAR(255) NOT NULL PRIMARY KEY,
    account_id                  VARCHAR(255) NOT NULL,
    agent_definition_id         VARCHAR(255) NOT NULL,
    is_enabled                  BOOLEAN NOT NULL DEFAULT FALSE,
    schedule                    VARCHAR(255),              -- cron expression, null = event-driven only
    settings                    JSON NOT NULL DEFAULT ('{}'), -- agent-specific thresholds
    requires_approval           BOOLEAN NOT NULL DEFAULT TRUE,
    auto_approve_min_confidence DOUBLE,                    -- null = never auto-approve
    created_at                  TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                  TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_agent_config (account_id, agent_definition_id),
    KEY idx_agent_config_account (account_id)
);
```

### *agent_run — Single execution of an agent*

```sql
CREATE TABLE agent_run (
    id                    VARCHAR(255) NOT NULL PRIMARY KEY,
    agent_config_id       VARCHAR(255) NOT NULL,
    agent_definition_id   VARCHAR(255) NOT NULL,
    account_id            VARCHAR(255) NOT NULL,
    status_code           VARCHAR(50) NOT NULL,            -- 'running', 'completed', 'failed', 'cancelled'
    trigger_type          VARCHAR(50) NOT NULL,            -- 'scheduled', 'manual', 'event'
    trigger_ref           VARCHAR(255),                    -- event ID, cron timestamp, etc.
    input_summary         TEXT,
    output_summary        TEXT,
    input_tokens          INT DEFAULT 0,
    output_tokens         INT DEFAULT 0,
    total_tokens          INT DEFAULT 0,
    llm_provider          VARCHAR(50),                     -- 'anthropic', 'openai'
    llm_model             VARCHAR(100),                    -- 'claude-sonnet-4', etc.
    duration_ms           INT,
    error_message         TEXT,
    started_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    completed_at          TIMESTAMP(3),
    created_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_agent_run_config (agent_config_id),
    KEY idx_agent_run_account (account_id),
    KEY idx_agent_run_status (status_code),
    KEY idx_agent_run_started (started_at)
);
```

### *agent_action — Proposed/executed action from a run*

```sql
CREATE TABLE agent_action (
    id                    VARCHAR(255) NOT NULL PRIMARY KEY,
    agent_run_id          VARCHAR(255) NOT NULL,
    account_id            VARCHAR(255) NOT NULL,
    status_code           VARCHAR(50) NOT NULL,            -- 'pending_review', 'approved', 'rejected', 'auto_approved', 'modified', 'executed', 'failed'
    action_type           VARCHAR(255) NOT NULL,           -- 'create_sales_order', 'send_alert', 'create_production_run', etc.
    target_entity_type    VARCHAR(255),                    -- 'sales_order', 'invoice', etc.
    target_entity_id      VARCHAR(255),                    -- null if creating new
    confidence            DOUBLE NOT NULL,                 -- 0.0 to 1.0
    reasoning             TEXT,                            -- agent's explanation
    proposed_payload      JSON NOT NULL,                   -- what the agent wants to create/modify
    executed_payload      JSON,                            -- actual data after user modifications
    reviewed_by_user_id   VARCHAR(255),
    review_note           TEXT,
    reviewed_at           TIMESTAMP(3),
    executed_at           TIMESTAMP(3),
    created_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_agent_action_run (agent_run_id),
    KEY idx_agent_action_account (account_id),
    KEY idx_agent_action_status (status_code),
    KEY idx_agent_action_created (created_at)
);
```

### *agent_artifact — Source data and result drafts linked to actions*

```sql
CREATE TABLE agent_artifact (
    id                    VARCHAR(255) NOT NULL PRIMARY KEY,
    account_id            VARCHAR(255) NOT NULL,
    agent_action_id       VARCHAR(255) NOT NULL,
    role                  VARCHAR(50) NOT NULL,            -- 'source' or 'result'
    artifact_type         VARCHAR(255) NOT NULL,           -- 'email', 'document', 'sales_order_draft', 'digest'
    title                 VARCHAR(255),
    content_text          LONGTEXT,                        -- plain text / markdown
    content_html          LONGTEXT,                        -- original HTML (emails)
    content_json          JSON,                            -- structured data
    s3_key                VARCHAR(1024),                   -- file attachments
    mime_type             VARCHAR(255),
    metadata              JSON DEFAULT ('{}'),             -- email headers, etc.
    created_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_agent_artifact_action (agent_action_id),
    KEY idx_agent_artifact_account (account_id),
    KEY idx_agent_artifact_type (artifact_type)
);
```

### *agent_memory — Persistent agent observations scoped to entities*

```sql
CREATE TABLE agent_memory (
    id                      VARCHAR(255) NOT NULL PRIMARY KEY,
    account_id              VARCHAR(255) NOT NULL,
    agent_definition_code   VARCHAR(255),                  -- null = cross-agent visible
    entity_type             VARCHAR(255) NOT NULL,         -- 'account_relation', 'address', 'account_user', 'item', 'account'
    entity_id               VARCHAR(255) NOT NULL,
    category                VARCHAR(255) NOT NULL,         -- 'observation', 'preference', 'warning', 'pattern'
    content                 TEXT NOT NULL,
    confidence              DOUBLE NOT NULL DEFAULT 1.0,
    source_run_id           VARCHAR(255),
    last_reinforced_at      TIMESTAMP(3),
    reinforcement_count     INT NOT NULL DEFAULT 1,
    expires_at              TIMESTAMP(3),
    is_archived             BOOLEAN NOT NULL DEFAULT FALSE,
    created_by_user_id      VARCHAR(255),                  -- if manually created
    created_at              TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at              TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_agent_memory_account (account_id),
    KEY idx_agent_memory_entity (entity_type, entity_id),
    KEY idx_agent_memory_agent (agent_definition_code),
    KEY idx_agent_memory_archived (is_archived),
    FULLTEXT idx_agent_memory_content (content)
);
```

### *agent_alert — Notifications routed to users/roles*

```sql
CREATE TABLE agent_alert (
    id                    VARCHAR(255) NOT NULL PRIMARY KEY,
    account_id            VARCHAR(255) NOT NULL,
    agent_run_id          VARCHAR(255),
    agent_action_id       VARCHAR(255),
    severity_code         VARCHAR(50) NOT NULL,            -- 'info', 'warning', 'urgent', 'critical'
    title                 VARCHAR(255) NOT NULL,
    message               TEXT NOT NULL,
    target_entity_type    VARCHAR(255),
    target_entity_id      VARCHAR(255),
    routing_rule          VARCHAR(255) NOT NULL,           -- 'role:admin', 'user:acus_xxx', 'permission:sales_orders:read'
    is_dismissed          BOOLEAN NOT NULL DEFAULT FALSE,
    dismissed_by_user_id  VARCHAR(255),
    dismissed_at          TIMESTAMP(3),
    created_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_agent_alert_account (account_id),
    KEY idx_agent_alert_run (agent_run_id),
    KEY idx_agent_alert_severity (severity_code),
    KEY idx_agent_alert_dismissed (is_dismissed),
    KEY idx_agent_alert_routing (routing_rule),
    KEY idx_agent_alert_created (created_at)
);
```

### *agent_token_usage — Daily token usage tracking per account*

```sql
CREATE TABLE agent_token_usage (
    id                    VARCHAR(255) NOT NULL PRIMARY KEY,
    account_id            VARCHAR(255) NOT NULL,
    period_date           DATE NOT NULL,                   -- daily aggregation
    input_tokens          BIGINT NOT NULL DEFAULT 0,
    output_tokens         BIGINT NOT NULL DEFAULT 0,
    total_tokens          BIGINT NOT NULL DEFAULT 0,
    run_count             INT NOT NULL DEFAULT 0,
    created_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_agent_token_usage (account_id, period_date),
    KEY idx_agent_token_usage_account (account_id)
);
```

### *Entity ID Prefixes*

*Add to* `api/shared/id/id_prefix_values.go`*:*

```go
// New vocabulary
VocAgent = "ag"
VocArtifact = "af"
VocMemory = "mm"

// New prefixes
AgentDefinitionIDPrefix = composePrefix(VocAgent, VocDefinition)   // agdf_
AgentConfigIDPrefix     = composePrefix(VocAgent, VocAccount)      // agac_
AgentRunIDPrefix        = composePrefix(VocAgent, VocRun)          // agrn_
AgentActionIDPrefix     = composePrefix(VocAgent, VocAction)       // agax_
AgentArtifactIDPrefix   = composePrefix(VocAgent, VocArtifact)     // agaf_
AgentMemoryIDPrefix     = composePrefix(VocAgent, VocMemory)       // agmm_
AgentAlertIDPrefix      = composePrefix(VocAgent, VocNotification) // agnf_
AgentTokenUsageIDPrefix = composePrefix(VocAgent, VocToken)        // agtk_
```

### *Prisma Schema*

*Mirror all tables above as Prisma models in* `dashboard/packages/db/prisma/schema/schema.prisma`*. Add relations to existing Account and AccountUser models.*

---

## *3. Multi-Provider LLM Layer*

***Location:*** `api/services/agent-service/internal/llm/`

`provider.go` *— Core abstraction:*

```go
type LLMProvider interface {
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    CompleteWithTools(ctx context.Context, req *ToolCompletionRequest) (*ToolCompletionResponse, error)
    Name() string
}

type CompletionRequest struct {
    Model       string
    System      string
    Messages    []Message
    MaxTokens   int
    Temperature float64
}

type CompletionResponse struct {
    Content      string
    InputTokens  int
    OutputTokens int
    StopReason   string
}

type ToolCompletionRequest struct {
    CompletionRequest
    Tools []ToolDefinition
}

type ToolCompletionResponse struct {
    Content      string
    ToolCalls    []ToolCall
    InputTokens  int
    OutputTokens int
    StopReason   string
}
```

*Implementations:*

- `anthropic.go` *— Uses anthropic-sdk-go for Claude models*
- `openai.go` *— Uses openai-go for OpenAI models*

*Provider selection is per-agent-config via the settings JSON field:*

```json
{
  "llm_provider": "anthropic",
  "llm_model": "claude-sonnet-4"
}
```

---

## *4. Durable Messaging: Outbox/Inbox Pattern for All Agent Flows*

*Every async operation in the agent system follows the established transactional outbox/inbox pattern from* `shared/messaging/`*. This ensures at-least-once delivery with deduplication — no messages are ever lost, even if processes crash.*

***Reference implementations:***

- *Outbox write:* `api/services/auth-service/internal/event/notification_publisher.go` *— OutboxRepo.Create() inside the same DB transaction as the business write*
- *Outbox publish:* `api/shared/messaging/enqueuer.go` *— polls outbox table, acquires locks, publishes to RabbitMQ, marks published/failed with retry backoff*
- *Inbox consume:* `api/shared/messaging/inbox_consumer.go` *— InboxConsumer.Wrap() deduplicates by (message_id, handler), handles crash recovery*
- *Consumer example:* `api/services/billing-service/internal/event/stripe_webhook_consumer.go` *— Listen() → ConsumeMessages() → inboxConsumer.Wrap()*

### *Flow 1: Scheduled Agent Runs*

```
scheduler.go (time.Ticker poll loop)
  │
  ├─ Query agent_config for enabled agents with schedule due
  ├─ For each due agent:
  │     1. INSERT agent_run (status='pending') — DB transaction
  │     2. INSERT outbox message (agent.cmd.execute_run) — SAME transaction
  │     (atomicity: run record + publish intent committed together)
  │
  ├─ Enqueuer (background goroutine, already running in agent-service)
  │     polls outbox → publishes to RabbitMQ → marks published
  │
  └─ RunConsumer.Listen() (agent-service event consumer)
        consumes from AgentCmdExecuteRunQueue
        wrapped with InboxConsumer.Wrap("agent.execute_run", handler)
        handler: loads config, loads memories, calls LLM, produces actions
```

### *Flow 2: Inbound Email Processing*

```
Email arrives at orders+{accountSlug}@agents.augno.com
  → SES receives (MX record configured)
  → SES stores raw email in S3 (augno-agent-emails bucket)
  → SES publishes SNS notification
  → SNS calls api-gateway webhook endpoint (/v1/webhooks/ses-inbound)
  │
  ├─ api-gateway handler:
  │     1. Validate SNS signature
  │     2. INSERT outbox message (agent.cmd.process_email) — DB transaction
  │        payload: { s3_bucket, s3_key, recipient, sender, subject }
  │
  ├─ Enqueuer (api-gateway's outbox enqueuer)
  │     polls outbox → publishes to RabbitMQ → marks published
  │
  └─ EmailConsumer.Listen() (agent-service event consumer)
        consumes from AgentCmdProcessEmailQueue
        wrapped with InboxConsumer.Wrap("agent.process_email", handler)
        handler:
          1. Fetch raw email from S3
          2. Parse MIME (net/mail + mime/multipart)
          3. Route by To address: orders+slug → email_order, ar+slug → ar_email
          4. Resolve account from slug
          5. INSERT agent_run + outbox(agent.cmd.execute_run) — SAME transaction
          6. Run consumer picks it up (Flow 1 from here)
```

### *Flow 3: Action Execution (Auto-Approved)*

```
Agent run completes, produces actions:
  │
  ├─ If requires_approval=true AND confidence < threshold:
  │     INSERT agent_action (status='pending_review') — no outbox needed
  │     INSERT agent_alert (routing_rule='role:admin') — for UI notification
  │     INSERT outbox(notification.cmd.send_email) — email alert to reviewers
  │
  ├─ If auto-approved (requires_approval=false OR confidence >= threshold):
  │     1. INSERT agent_action (status='auto_approved') — DB transaction
  │     2. INSERT outbox(agent.cmd.execute_action) — SAME transaction
  │        payload: { action_id, action_type, proposed_payload }
  │
  │     Enqueuer publishes → RabbitMQ
  │
  │     ActionExecutionConsumer (dashboard API or agent-service)
  │       consumes from AgentCmdExecuteActionQueue
  │       wrapped with InboxConsumer.Wrap("agent.execute_action", handler)
  │       handler: calls existing service layer (OrderSvc.create, etc.)
  │       on success: UPDATE agent_action status='executed'
  │       on failure: UPDATE agent_action status='failed', error_message
  │
  └─ Run completion:
        INSERT outbox(agent.event.run_completed) — token usage data
        Consumed by billing integration for Stripe meter reporting
```

### *Flow 4: Manual Action Review (Dashboard API)*

```
User approves/rejects/modifies via PUT /v1/agents/actions/:id/review
  │
  ├─ If rejected: UPDATE agent_action status='rejected' — done
  │
  ├─ If approved/modified:
  │     1. UPDATE agent_action (status='approved'/'modified') — DB transaction
  │     2. INSERT outbox(agent.cmd.execute_action) — SAME transaction
  │
  │     Same execution flow as auto-approved actions (Flow 3)
  │
  └─ The outbox/inbox guarantees the execution attempt survives crashes:
       - If dashboard API crashes after DB commit but before outbox publish:
         enqueuer picks it up on next poll
       - If consumer crashes mid-execution:
         inbox dedup detects "received" status → crash recovery retry
```

### *New RabbitMQ Infrastructure*

*Queue definitions — add to* `api/shared/messaging/queues.go`*:*

```go
AgentCmdExecuteRunQueue    = "agent_cmd_execute_run"
AgentCmdProcessEmailQueue  = "agent_cmd_process_email"
AgentCmdExecuteActionQueue = "agent_cmd_execute_action"
AgentEventRunCompletedQueue = "agent_event_run_completed"
```

*Routing keys — add to* `api/shared/contracts/amqp.go`*:*

```go
AgentCmdExecuteRun     AmqpRoutingKey = "agent.cmd.execute_run"
AgentCmdProcessEmail   AmqpRoutingKey = "agent.cmd.process_email"
AgentCmdExecuteAction  AmqpRoutingKey = "agent.cmd.execute_action"
AgentEventRunCompleted AmqpRoutingKey = "agent.event.run_completed"
```

*Message payloads — add to* `api/shared/messaging/queues.go`*:*

```go
type AgentExecuteRunData struct {
    AgentRunID    string `json:"agent_run_id"`
    AgentConfigID string `json:"agent_config_id"`
    AccountID     string `json:"account_id"`
    TriggerType   string `json:"trigger_type"` // "scheduled", "manual", "event"
}

type AgentProcessEmailData struct {
    S3Bucket  string `json:"s3_bucket"`
    S3Key     string `json:"s3_key"`
    Recipient string `json:"recipient"` // full To address
    Sender    string `json:"sender"`
    Subject   string `json:"subject"`
}

type AgentExecuteActionData struct {
    AgentActionID   string          `json:"agent_action_id"`
    ActionType      string          `json:"action_type"`
    ProposedPayload json.RawMessage `json:"proposed_payload"`
    AccountID       string          `json:"account_id"`
}

type AgentRunCompletedData struct {
    AgentRunID   string `json:"agent_run_id"`
    AccountID    string `json:"account_id"`
    InputTokens  int    `json:"input_tokens"`
    OutputTokens int    `json:"output_tokens"`
    TotalTokens  int    `json:"total_tokens"`
    LLMProvider  string `json:"llm_provider"`
    LLMModel     string `json:"llm_model"`
}
```

### *agent-service run.go Initialization*

*Following* `billing-service/cmd/run.go`*:*

```go
func Run(ctx context.Context, ...) error {
    // 1. Config, logger, tracing, DB pool — same as billing-service
    // 2. RabbitMQ connection
    // 3. sqlc prepared queries
    // 4. Outbox enqueuer (polls agent-service outbox messages → RabbitMQ)
    // 5. Inbox purger (cleans up processed inbox records)
    // 6. LLM provider initialization (Anthropic + OpenAI clients)
    // 7. Repository factory
    // 8. Agent runner service (business logic)
    // 9. Consumers:
    //    - RunConsumer.Listen() on AgentCmdExecuteRunQueue
    //    - EmailConsumer.Listen() on AgentCmdProcessEmailQueue
    //    All wrapped with InboxConsumer.Wrap() for deduplication
    // 10. Scheduler (background goroutine, writes to outbox)
    // 11. gRPC server (for manual trigger RPCs from api-gateway)
    // 12. server.Serve(ctx, cfg.Port)
}
```

---

## *5. Email Ingestion (AWS SES Inbound)*

***Email address format:***

- *orders+{accountSlug}@agents.augno.com → routes to email_order agent*
- *ar+{accountSlug}@agents.augno.com → routes to ar_email agent*
- *Fallback: match sender email against AccountRelation contacts to find the account*

***Email parser** (*`internal/email/parser.go`*):*

- *Parse MIME (use net/mail + mime/multipart)*
- *Extract: sender, recipients, subject, plain text body, HTML body, attachments*
- *Store attachments in S3, reference via s3_key in artifacts*

***SES/S3/SNS setup** (infrastructure, not code):*

- *SES receipt rule: receive on agents.augno.com → store in S3 bucket → publish to SNS topic*
- *SNS subscription: HTTPS endpoint → api-gateway /v1/webhooks/ses-inbound*
- *S3 bucket: augno-agent-emails with lifecycle policy (30-day expiry)*

---

## *6. Agent Memory System*

### *Memory Scoping*

| *Scope* | *entity_type* | *entity_id* | *Example* |
| --- | --- | --- | --- |
| *Customer* | *account_relation* | *AccountRelation ID* | *"Always orders in Q4"* |
| *Ship-to address* | *address* | *Address ID* | *"Dock hours 6am-2pm"* |
| *User preference* | *account_user* | *AccountUser ID* | *"Prefers weekly AR digest"* |
| *Item* | *item* | *Item ID* | *"Seasonal demand spikes"* |
| *Account-wide* | *account* | *Account ID* | *"Holiday shutdown Dec 20-Jan 3"* |

### *Cross-Agent vs Agent-Specific*

- *agent_definition_code = NULL → visible to all agents (e.g., customer notes)*
- *agent_definition_code = 'email_order' → only loaded by that agent*

### *Retrieval Strategy*

*Filtered SQL queries (no vector search needed initially):*

1. *Load memories for the specific entities being processed in the current run*
2. *Load account-wide memories*
3. *Filter by agent_definition_code (null OR matching agent)*
4. *Exclude archived and expired memories*
5. *Order by reinforcement_count DESC, cap at 50*

*MySQL FULLTEXT index on content enables search in the memory browser UI.*

### *Memory Lifecycle*

- ***Creation:** Agent writes memories during runs via structured tool calls*
- ***Reinforcement:** If an agent observes something already stored, increment reinforcement_count + update last_reinforced_at instead of duplicating*
- ***Expiry:** Optional expires_at for transient observations*
- ***Manual:** Users can create/edit/delete memories via the UI*

---

## *7. Token Usage Metering*

### *Tracking*

- *Each agent_run records input_tokens, output_tokens, total_tokens, llm_provider, llm_model*
- *Agent-service upserts agent_token_usage daily aggregate after each run*
- *Agent-service checks token budget before starting a run (query current period usage vs plan limit)*

### *Stripe Integration*

*Add a new Stripe meter in* `dashboard/apps/api/src/integrations/stripe-meters.ts`*:*

```ts
enum StripeMeterEvents {
  // ... existing
  agentTokens = "augno_agent_tokens",
}
```

*Dashboard API reports token usage to Stripe after runs complete (consuming a agent_event_run_completed message).*

### *Free Tier*

- *Store free tier token limit as a plan-level constant*
- *Agent-service checks agent_token_usage for current billing period before executing*
- *If over limit, skip run and create an agent_alert with severity_code = 'warning'*
- *Allow users to add more credits, in batches of $20, $40, $80, $100 or a custom amount with one click in frontend*
- *Allow admin users to setup token limits for use in sandbox environements so all are not spent in testing*

---

## *8. Agent Run Lifecycle (Inside RunConsumer Handler)*

*When the RunConsumer receives an agent.cmd.execute_run message (via inbox dedup), the handler executes:*

1. ***UPDATE** agent_run status='running', started_at=now()*
2. ***LOAD context:***
   - *Agent config + settings (from agent_config table)*
   - *Relevant memories for entities in scope (from agent_memory table)*
   - *Business data via sqlc (orders, customers, etc. — read-only)*
3. ***CHECK token budget:***
   - *Query agent_token_usage for current billing period*
   - *If over limit: UPDATE run status='failed', create alert, return*
4. ***EXECUTE LLM** (via LLMProvider interface):*
   - *Build system prompt with agent instructions + loaded memories*
   - *Provide tool definitions (create_order, create_alert, save_memory, etc.)*
   - *Run agentic loop: LLM calls tools → handler returns results → LLM continues*
   - *Accumulate token usage across all LLM calls*
5. ***PRODUCE outputs** (all in a single DB transaction + outbox writes):*
   - *INSERT agent_action records (status per auto-approval logic)*
   - *INSERT agent_artifact records (source + result linked to actions)*
   - *INSERT/UPSERT agent_memory records (new observations, reinforcements)*
   - *INSERT agent_alert records (for pending_review actions)*
   - *INSERT outbox messages for:*
     - *Each auto-approved action → agent.cmd.execute_action*
     - *Each alert needing email → notification.cmd.send_email*
     - *Run completion event → agent.event.run_completed*
   - *(All outbox writes in SAME transaction = atomic with domain writes)*
6. ***FINALIZE:***
   - *UPDATE agent_run status='completed', token counts, duration_ms*
   - *UPSERT agent_token_usage daily aggregate*

### *Review Workflow (Dashboard API)*

```
pending_review ──► approved ──► executed
               ├─► rejected
               └─► modified ──► executed
                                   └─► failed
```

- *When a user approves/modifies, the dashboard API writes the action status update + an outbox message (agent.cmd.execute_action) in the same transaction*
- *The ActionExecutionConsumer picks it up via inbox dedup and calls the existing service layer (e.g., OrderSvc.create() for create_sales_order)*
- *The executed_payload stores the final data (original or user-modified)*
- *On success: UPDATE action status='executed'. On failure: UPDATE status='failed' with error_message*

---

## *9. API Surface*

### *gRPC (agent-service)*

*Proto file:* `api/proto/agent/agent.proto`

```protobuf
service AgentService {
    rpc TriggerRun(TriggerRunRequest) returns (TriggerRunResponse);      // Manual run trigger
    rpc GetRunStatus(GetRunStatusRequest) returns (GetRunStatusResponse); // Poll run status
    rpc CancelRun(CancelRunRequest) returns (CancelRunResponse);
}
```

*Manual trigger flow: api-gateway receives HTTP request → calls AgentService.TriggerRun via gRPC → agent-service inserts agent_run + outbox message (agent.cmd.execute_run) in same transaction → returns run ID → run executes asynchronously via consumer.*

### *HTTP Endpoints (Dashboard API via api-gateway)*

| *Method* | *Path* | *Description* |
| --- | --- | --- |
| *GET* | */v1/agents/definitions* | *List available agent types* |
| *GET* | */v1/agents/configs* | *List agent configs for account* |
| *PUT* | */v1/agents/configs/:id* | *Update agent config (enable, schedule, thresholds)* |
| *POST* | */v1/agents/runs/trigger* | *Manually trigger an agent run (proxied to gRPC)* |
| *GET* | */v1/agents/runs* | *List runs with filtering* |
| *GET* | */v1/agents/runs/:id* | *Get run detail with actions + artifacts* |
| *GET* | */v1/agents/actions* | *Action review queue (filterable)* |
| *PUT* | */v1/agents/actions/:id/review* | *Approve/reject/modify (writes outbox for execution)* |
| *GET* | */v1/agents/memories* | *Search/browse memories* |
| *POST* | */v1/agents/memories* | *Manually create a memory* |
| *PUT* | */v1/agents/memories/:id* | *Edit a memory* |
| *DELETE* | */v1/agents/memories/:id* | *Delete a memory* |
| *GET* | */v1/agents/alerts* | *List alerts for user* |
| *PUT* | */v1/agents/alerts/:id/dismiss* | *Dismiss an alert* |
| *GET* | */v1/agents/usage* | *Token usage summary* |
| *POST* | */v1/webhooks/ses-inbound* | *SES inbound email (writes outbox for processing)* |

---

## *10. Dashboard Integration*

### *New Files (Express API)*

*Following existing patterns in* `dashboard/apps/api/src/`*:*

- ***Controllers:** agent-config.ctrl.ts, agent-run.ctrl.ts, agent-action.ctrl.ts, agent-memory.ctrl.ts, agent-alert.ctrl.ts*
- ***Services:** agent-config.svc.ts, agent-run.svc.ts, agent-action.svc.ts, agent-memory.svc.ts, agent-alert.svc.ts*
- ***Repositories:** One repo + interface per model (6 pairs)*
- ***DTOs:** packages/dtos/src/sections/agents.ts — Zod schemas for all endpoints*
- ***Endpoint registration:** Add to apps/api/src/index.ts via regEP()*

### *New Permissions*

*Add to* `packages/static/src/enums.ts`*:*

- *agents:read, agents:write, agents:execute*
- *agent_memories:read, agent_memories:write*
- *agent_actions:read, agent_actions:review*

### *Frontend Routes*

*Under* `apps/frontend/src/app/(user)/dashboard/(default-padding)/agents/`*:*

```
agents/
├── page.tsx                    # Overview: enabled agents, pending count, recent activity
├── [agentCode]/
│   ├── page.tsx                # Config panel + run history table
│   └── runs/[runID]/page.tsx   # Run detail with side-by-side view
├── actions/page.tsx            # Cross-agent review queue
├── memories/page.tsx           # Memory browser (filter by entity, agent, category)
└── alerts/page.tsx             # Alert feed
```

***Key UI components:***

- *SideBySideView: Split panel — source artifact (email/document) on left, proposed action on right with approve/reject/modify buttons*
- *ConfidenceBadge: Color-coded chip (green >0.9, blue >0.7, amber >0.5, red <0.5)*
- *EntityMemoryPanel: Reusable panel embedded in existing customer/item/address detail pages showing memories for that entity*
- *ActionReviewDialog: Modal for approve/reject/modify with notes field and editable payload*

***Zustand store:** agent-store.ts — tracks pending action count and active alert count for nav badges*

---

## *11. Agent Descriptions*

| *#* | *Agent* | *Trigger* | *Reads* | *Produces* |
| --- | --- | --- | --- | --- |
| *1* | *Email → Order* | *Inbound email (event)* | *Email content, customer contacts, item catalog* | *Sales order draft, forwarding alert* |
| *2* | *Customer Cadence* | *Scheduled (daily)* | *Order history, customer relations* | *Alerts for off-cadence customers* |
| *3* | *AR Email* | *Inbound email (event)* | *Email content, invoices, transactions* | *Payment allocations, AR alerts* |
| *4* | *MRP Scheduler* | *Scheduled/manual* | *Orders, inventory, production formulas* | *Production run drafts* |
| *5* | *Trends Digest* | *Scheduled (daily)* | *Orders, invoices, shipments, inventory* | *Digest artifact, alerts* |
| *6* | *Sales Research* | *Scheduled/manual* | *Customer data, territories, order history* | *Research artifacts, alerts* |

---

## *12. Implementation Phases*

### *Phase 1: Foundation (schema + service skeleton)*

1. *Add migration SQL for all 7 agent tables*
2. *Add Prisma models and generate client*
3. *Add ID prefixes to api/shared/id/*
4. *Add RabbitMQ queues and routing keys*
5. *Create agent-service skeleton (cmd/, config, gRPC server)*
6. *Add agent.proto with basic RPCs*
7. *Run make sqlc agent and make proto*

### *Phase 2: LLM Layer + Run Engine*

1. *Implement LLMProvider interface with Anthropic + OpenAI providers*
2. *Implement agent run lifecycle engine (runner.go)*
3. *Implement scheduler with cron parsing*
4. *Write agent handler interface and one agent (email_order) as proof of concept*

### *Phase 3: Email Ingestion*

1. *Configure SES inbound rule + S3 bucket + SNS topic*
2. *Add webhook endpoint to api-gateway for SNS notifications*
3. *Implement email parser and routing logic*
4. *Connect to agent runner*

### *Phase 4: Dashboard API*

1. *Create DTOs, controllers, services, repositories for all agent entities*
2. *Register endpoints*
3. *Implement review workflow (approve/reject/modify → execute via existing services)*
4. *Add Stripe meter for agent tokens*
5. *Add agent permissions*

### *Phase 5: Frontend*

1. *Agent overview page*
2. *Agent config panel*
3. *Run history + detail with side-by-side view*
4. *Action review queue*
5. *Memory browser*
6. *Alert feed*
7. *Entity memory panels on customer/item/address pages*

### *Phase 6: Remaining Agents*

1. *Customer cadence agent*
2. *AR email agent*
3. *MRP scheduling agent*
4. *Trends digest agent*
5. *Sales research agent*

---

## *Verification*

1. ***Unit tests:** Each agent handler, LLM provider, email parser, scheduler — standard go test in agent-service*
2. ***Integration test:** Trigger a manual run via gRPC → verify run/action/artifact records created → approve action via dashboard API → verify order created*
3. ***Email flow:** Send test email to SES inbound address → verify it reaches agent-service → verify action appears in review queue*
4. ***Token metering:** Run an agent → verify agent_token_usage updated → verify Stripe meter event reported*
5. ***Memory lifecycle:** Run agent that creates a memory → run again → verify reinforcement (not duplication)*
6. ***Frontend:** Navigate to agents dashboard → configure an agent → view runs → review actions side-by-side → manage memories*
