// Package llm drafts text with an LLM through the Stripe AI Gateway.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/tracing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var drafterTracer = tracing.GetTracer("platform-service.llm.account_followup_drafter")

const (
	defaultGatewayBaseURL = "https://llm.stripe.com"

	// accountFollowupPromptVersion is stored with every draft, so a reviewer comparing drafts over time can tell a prompt change from a change in registrants.
	accountFollowupPromptVersion = "2026-10-05"

	// drafterRequestTimeout bounds one attempt. A short paragraph returns in seconds; the SDK retries transient failures on top of this.
	drafterRequestTimeout = 60 * time.Second

	// maxParagraphChars caps the personal paragraph. The model is asked for two to four sentences; anything far past that is a model that ignored the brief, and is better retried than sent for review.
	maxParagraphChars = 900
)

// AccountFollowupDrafterConfig configures the drafter.
type AccountFollowupDrafterConfig struct {
	// StripeSecretKey (required) authenticates to the Stripe AI Gateway. No customer is attached, so usage is billed to the platform's own account rather than metered to a tenant.
	StripeSecretKey string

	// Model (required) is the gateway model name without its provider prefix (e.g. "claude-haiku-4.5").
	Model string

	// BaseURL (optional; default: "https://llm.stripe.com") overrides the gateway, for tests.
	BaseURL string
}

func (c *AccountFollowupDrafterConfig) withDefaults() *AccountFollowupDrafterConfig {
	if c.BaseURL == "" {
		c.BaseURL = defaultGatewayBaseURL
	}
	return c
}

func (c *AccountFollowupDrafterConfig) validate() error {
	if c.StripeSecretKey == "" {
		return errors.New("account follow-up drafter: stripe secret key is required")
	}
	if c.Model == "" {
		return errors.New("account follow-up drafter: model is required")
	}
	return nil
}

type accountFollowupDrafterImpl struct {
	client anthropic.Client
	model  string
}

func NewAccountFollowupDrafter(config *AccountFollowupDrafterConfig) (domain.AccountFollowupDrafter, error) {
	config = config.withDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &accountFollowupDrafterImpl{
		client: anthropic.NewClient(
			option.WithBaseURL(config.BaseURL),
			// The gateway authenticates with the Stripe key in Authorization; the SDK still requires an API key to be set.
			option.WithAPIKey("unused"),
			option.WithHeader("Authorization", "Bearer "+config.StripeSecretKey),
			option.WithRequestTimeout(drafterRequestTimeout),
		),
		model: config.Model,
	}, nil
}

const accountFollowupSystemPrompt = `You help Dane, who builds OpenMRP, write a short personal follow-up to someone who signed up for OpenMRP about a day ago. OpenMRP is manufacturing resource planning software for small manufacturers: catalog and bill of materials, inventory, purchasing, production runs and scheduling, sales orders, shipping, and an API.

You receive the registrant's first name, their company name, and a summary of what they did in the product since signing up. You write two things.

For Dane, who decides whether to send:
- engagement: "low" (little or nothing beyond signing up), "medium" (explored or set a few things up), or "high" (built real data, came back, or used the API).
- internal_summary: two to four plain sentences on what they appear to be evaluating, how far they got, and anything that may have gone wrong or slowed them down. Speculate carefully and say when you are guessing.

For the registrant, as Dane:
- subject: a short, lowercase-friendly subject a person would write to one other person. No marketing language, no exclamation marks, no emoji.
- paragraph: two to four sentences that open the email after "Hi <name>,". Thank them for trying OpenMRP and, when their activity supports it, refer lightly to one area they seemed to be working on, phrased as an impression rather than an observation ("looks like you've started setting up your locations"). If they barely used it, don't pretend otherwise; say you'd love to know what they were hoping to find.

The paragraph must never sound like they were watched. Never mention timestamps, counts, request numbers, session lengths, routes, HTTP codes, error messages, sandboxes, or API keys by name. Never promise features, pricing, timelines, or fixes. Do not ask for feedback or offer a call; Dane's closing lines already do that. Write in plain, warm, direct English, the way the person who built the product writes to one customer.`

// draftToolName is the tool the model is made to call. The answer comes back as the tool's input rather than through structured outputs, because the gateway routes Claude through a Vertex project whose policy disallows structured outputs (and with it strict tool use), while forced tool use is allowed.
const draftToolName = "write_followup"

// draftToolSchema describes exactly the fields the follow-up needs. Without strict mode it guides rather than guarantees the shape, so parseDraft validates what comes back.
var draftToolSchema = anthropic.ToolInputSchemaParam{
	Properties: map[string]any{
		"engagement":       map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
		"internal_summary": map[string]any{"type": "string"},
		"subject":          map[string]any{"type": "string"},
		"paragraph":        map[string]any{"type": "string"},
	},
	Required: []string{"engagement", "internal_summary", "subject", "paragraph"},
}

type draftResponse struct {
	Engagement      string `json:"engagement"`
	InternalSummary string `json:"internal_summary"`
	Subject         string `json:"subject"`
	Paragraph       string `json:"paragraph"`
}

func (d *accountFollowupDrafterImpl) Draft(ctx context.Context, input domain.AccountFollowupDrafterInput) (*domain.AccountFollowupDrafterOutput, error) {
	ctx, span := drafterTracer.Start(ctx, "llm.account_followup_drafter.draft")
	defer span.End()

	activity, err := json.MarshalIndent(input.Activity, "", "  ")
	if err != nil {
		return nil, recordError(span, fmt.Errorf("encode activity: %w", err))
	}
	user := fmt.Sprintf("First name: %s\nCompany: %s\n\nActivity since signing up:\n%s", input.FirstName, input.AccountName, activity)

	resp, err := d.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "anthropic/" + d.model,
		MaxTokens: 4000,
		System:    []anthropic.TextBlockParam{{Text: accountFollowupSystemPrompt}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(user))},
		Tools:     []anthropic.ToolUnionParam{anthropic.ToolUnionParamOfTool(draftToolSchema, draftToolName)},
		// Forcing a named tool works on Haiku 4.5; current larger models reject it, so a model change here needs a different way to get the answer back.
		ToolChoice: anthropic.ToolChoiceParamOfTool(draftToolName),
	})
	if err != nil {
		return nil, recordError(span, fmt.Errorf("draft account follow-up: %w", err))
	}
	if resp.StopReason != anthropic.StopReasonToolUse {
		return nil, recordError(span, fmt.Errorf("draft account follow-up: model stopped with %q", resp.StopReason))
	}

	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.ToolUseBlock); ok && t.Name == draftToolName {
			out, err := parseDraft(string(t.Input), d.model)
			if err != nil {
				return nil, recordError(span, err)
			}
			return out, nil
		}
	}
	return nil, recordError(span, errors.New("draft account follow-up: model did not call the draft tool"))
}

func recordError(span trace.Span, err error) error {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// parseDraft validates the tool input. The schema only guides its shape, so this checks every field, and rejects what a schema cannot, like an empty paragraph or one long enough to mean the brief was ignored.
func parseDraft(text, model string) (*domain.AccountFollowupDrafterOutput, error) {
	var r draftResponse
	if err := json.Unmarshal([]byte(text), &r); err != nil {
		return nil, fmt.Errorf("parse account follow-up draft: %w", err)
	}

	out := &domain.AccountFollowupDrafterOutput{
		Model:           model,
		PromptVersion:   accountFollowupPromptVersion,
		Engagement:      domain.AccountFollowupEngagement(r.Engagement),
		InternalSummary: strings.TrimSpace(r.InternalSummary),
		Subject:         strings.TrimSpace(r.Subject),
		Paragraph:       strings.TrimSpace(r.Paragraph),
	}
	switch {
	case !out.Engagement.IsValid():
		return nil, fmt.Errorf("account follow-up draft has invalid engagement %q", r.Engagement)
	case out.Subject == "" || out.Paragraph == "" || out.InternalSummary == "":
		return nil, errors.New("account follow-up draft is missing a field")
	case len(out.Subject) > 120:
		return nil, fmt.Errorf("account follow-up draft subject is %d characters", len(out.Subject))
	case len(out.Paragraph) > maxParagraphChars:
		return nil, fmt.Errorf("account follow-up draft paragraph is %d characters", len(out.Paragraph))
	}
	return out, nil
}
