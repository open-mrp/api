package constants

// Model is the stable identifier for an LLM model that agents can be configured to use. Values match the Stripe AI Gateway naming convention (no provider prefix; the gateway client adds it).
type Model string

// The model catalog — every model agents may use. Each has a named constant (the constants-adherence test requires one per EnumValues entry); ModelCatalog pairs each with display metadata.
const (
	ModelClaudeOpus48   Model = "claude-opus-4.8"
	ModelClaudeOpus47   Model = "claude-opus-4.7"
	ModelClaudeOpus46   Model = "claude-opus-4.6"
	ModelClaudeOpus45   Model = "claude-opus-4.5"
	ModelClaudeSonnet46 Model = "claude-sonnet-4.6"
	ModelClaudeSonnet45 Model = "claude-sonnet-4.5"
	ModelClaudeSonnet4  Model = "claude-sonnet-4"
	ModelClaudeHaiku45  Model = "claude-haiku-4.5"
	ModelClaude37Sonnet Model = "claude-3.7-sonnet"
	ModelClaude35Sonnet Model = "claude-3.5-sonnet"
	ModelClaude35Haiku  Model = "claude-3.5-haiku"
	ModelGPT55          Model = "gpt-5.5"
	ModelGPT54          Model = "gpt-5.4"
	ModelGPT52          Model = "gpt-5.2"
	ModelGPT51          Model = "gpt-5.1"
	ModelGPT5           Model = "gpt-5"
	ModelGPT5Mini       Model = "gpt-5-mini"
	ModelGPT4o          Model = "gpt-4o"
	ModelGPT4oMini      Model = "gpt-4o-mini"
	ModelGPT41Mini      Model = "gpt-4.1-mini"
	ModelGPT4           Model = "gpt-4"
	ModelGPT35Turbo     Model = "gpt-3.5-turbo"
	ModelGemini3Flash   Model = "gemini-3-flash"
	ModelGemini25Flash  Model = "gemini-2.5-flash"
	ModelGemini25Pro    Model = "gemini-2.5-pro"
	ModelGrok4          Model = "grok-4"
	ModelGrok3          Model = "grok-3"
	ModelGrok3Mini      Model = "grok-3-mini"
)

// ModelSpec is a catalog entry: the gateway model id plus its display metadata.
type ModelSpec struct {
	ID       Model
	Name     string
	Provider string
}

// ModelCatalog is the full set of LLM models agents may use — the single source of truth for validation and the model enum. IDs mirror the models available in the Stripe AI Gateway dashboard.
var ModelCatalog = []ModelSpec{
	// Anthropic
	{ModelClaudeOpus48, "Claude Opus 4.8", "Anthropic"},
	{ModelClaudeOpus47, "Claude Opus 4.7", "Anthropic"},
	{ModelClaudeOpus46, "Claude Opus 4.6", "Anthropic"},
	{ModelClaudeOpus45, "Claude Opus 4.5", "Anthropic"},
	{ModelClaudeSonnet46, "Claude Sonnet 4.6", "Anthropic"},
	{ModelClaudeSonnet45, "Claude Sonnet 4.5", "Anthropic"},
	{ModelClaudeSonnet4, "Claude Sonnet 4", "Anthropic"},
	{ModelClaudeHaiku45, "Claude Haiku 4.5", "Anthropic"},
	{ModelClaude37Sonnet, "Claude 3.7 Sonnet", "Anthropic"},
	{ModelClaude35Sonnet, "Claude 3.5 Sonnet", "Anthropic"},
	{ModelClaude35Haiku, "Claude 3.5 Haiku", "Anthropic"},

	// OpenAI
	{ModelGPT55, "GPT-5.5", "OpenAI"},
	{ModelGPT54, "GPT-5.4", "OpenAI"},
	{ModelGPT52, "GPT-5.2", "OpenAI"},
	{ModelGPT51, "GPT-5.1", "OpenAI"},
	{ModelGPT5, "GPT-5", "OpenAI"},
	{ModelGPT5Mini, "GPT-5 mini", "OpenAI"},
	{ModelGPT4o, "GPT-4o", "OpenAI"},
	{ModelGPT4oMini, "GPT-4o mini", "OpenAI"},
	{ModelGPT41Mini, "GPT-4.1 mini", "OpenAI"},
	{ModelGPT4, "GPT-4", "OpenAI"},
	{ModelGPT35Turbo, "GPT-3.5 Turbo", "OpenAI"},

	// Google
	{ModelGemini3Flash, "Gemini 3 Flash", "Google"},
	{ModelGemini25Flash, "Gemini 2.5 Flash", "Google"},
	{ModelGemini25Pro, "Gemini 2.5 Pro", "Google"},

	// xAI
	{ModelGrok4, "Grok 4", "xAI"},
	{ModelGrok3, "Grok 3", "xAI"},
	{ModelGrok3Mini, "Grok 3 mini", "xAI"},
}

var modelSet = func() map[Model]bool {
	m := make(map[Model]bool, len(ModelCatalog))
	for _, s := range ModelCatalog {
		m[s.ID] = true
	}
	return m
}()

func (m Model) IsValid() bool { return modelSet[m] }

func (m Model) EnumValues() []string {
	out := make([]string, len(ModelCatalog))
	for i, s := range ModelCatalog {
		out[i] = string(s.ID)
	}
	return out
}

// ModelTier selects an intelligence/cost level instead of a specific model. Callers pick a tier; the harness resolves it to an ordered model chain (primary first, then fallbacks). Higher tiers cost more and reason better; reserve them for genuinely hard work and use cheaper tiers for background reasoning, extraction, and routine transforms.
type ModelTier string

const (
	// ModelTierFrontier is the hardest-intelligence tier: multi-step planning, ambiguous agent work, hard coding/architecture, tool-heavy workflows.
	ModelTierFrontier ModelTier = "frontier"
	// ModelTierHigh is the default agent tier: normal planning, code edits, synthesis, customer-facing reasoning.
	ModelTierHigh ModelTier = "high"
	// ModelTierBalanced is for research, summarization, classification, structured extraction, and light tool use.
	ModelTierBalanced ModelTier = "balanced"
	// ModelTierCheap is for simple transforms, validation, formatting, keyword lookup, and routing.
	ModelTierCheap ModelTier = "cheap"
	// ModelTierLegacy is for compatibility / regression comparison only; avoid unless needed.
	ModelTierLegacy ModelTier = "legacy"
)

// DefaultModelTier is used when a caller doesn't specify one (e.g. a normal agent run).
const DefaultModelTier = ModelTierHigh

// modelTierChains maps each tier to its ordered model preference: the primary (Claude-first, per our preference) followed by fallbacks. The chain interleaves providers so a provider outage (e.g.
// Anthropic down) fails over to an equivalent-sized model on another provider.
var modelTierChains = map[ModelTier][]Model{
	ModelTierFrontier: {ModelClaudeOpus48, ModelGPT55, ModelClaudeSonnet46, ModelGPT54},
	ModelTierHigh:     {ModelClaudeSonnet46, ModelGPT55, ModelClaudeSonnet4, ModelGPT52},
	ModelTierBalanced: {ModelClaudeHaiku45, ModelGPT5Mini, ModelGemini3Flash, ModelGPT4oMini, ModelGPT41Mini},
	ModelTierCheap:    {ModelGPT4oMini, ModelGPT41Mini, ModelClaude35Haiku, ModelGemini25Flash},
	ModelTierLegacy:   {ModelGPT4, ModelGPT35Turbo, ModelClaude35Sonnet},
}

// IsValid reports whether the tier is a known tier.
func (t ModelTier) IsValid() bool {
	_, ok := modelTierChains[t]
	return ok
}

func (t ModelTier) EnumValues() []string {
	return []string{
		string(ModelTierFrontier),
		string(ModelTierHigh),
		string(ModelTierBalanced),
		string(ModelTierCheap),
		string(ModelTierLegacy),
	}
}

// ModelChain returns the ordered model chain (primary first, then fallbacks) for the tier, falling back to the default tier for an unknown/empty tier. The runner tries each model in order, advancing to the next on a provider failure.
func (t ModelTier) ModelChain() []string {
	chain, ok := modelTierChains[t]
	if !ok {
		chain = modelTierChains[DefaultModelTier]
	}
	out := make([]string, len(chain))
	for i, m := range chain {
		out[i] = string(m)
	}
	return out
}
