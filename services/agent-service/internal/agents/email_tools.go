package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

// HandleSendEmail sends an email reply through the conversation's bound inbox. The tool is gated by human review (see send_email's RequireReview wiring), so by the time this handler runs the send has been approved.
func HandleSendEmail(ctx context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Cc      []string `json:"cc"`
		Subject string   `json:"subject"`
		Body    string   `json:"body"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid send_email input: %w", err)
	}
	if strings.TrimSpace(params.Body) == "" {
		return "", fmt.Errorf("send_email requires a body")
	}
	if err := requireEmailContext(runCtx); err != nil {
		return "", err
	}

	_, err := runCtx.NotificationClient.SendInboxReply(ctx, domain.SendInboxReplyRequest{
		ConversationID: runCtx.ConversationID,
		Subject:        params.Subject,
		Body:           params.Body,
		Cc:             params.Cc,
		AgentConfigID:  emailAgentConfigID(runCtx),
		AgentRunID:     runCtx.RunID,
		Identity:       runCtx.Identity,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}
	return "Email sent to the customer through the inbox and recorded in the conversation.", nil
}

// HandleDraftReply proposes a reply to the case's external party as a real status=draft message held for
// human approval. Works on any case regardless of who it's with (customer, supplier, other contact) — the
// channel is resolved server-side (email if bridged, else a portal message) and the draft surfaces in the
// reply-drafts bar for a human to edit/approve/send. The conversation is fixed by the run, so the agent
// supplies only the content and never needs (or is given) a conversation id.
func HandleDraftReply(ctx context.Context, input json.RawMessage, runCtx *domain.HandlerRunContext) (string, error) {
	var params struct {
		Body    string `json:"body"`
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid draft_reply input: %w", err)
	}
	if strings.TrimSpace(params.Body) == "" {
		return "", fmt.Errorf("draft_reply requires a body")
	}
	if runCtx.NotificationClient == nil {
		return "", fmt.Errorf("drafting is not available in this environment")
	}
	if runCtx.ConversationID == "" {
		return "", fmt.Errorf("draft_reply can only be used on a conversation")
	}

	_, err := runCtx.NotificationClient.PostReplyDraft(ctx, domain.PostReplyDraftRequest{
		ConversationID: runCtx.ConversationID,
		Body:           params.Body,
		Subject:        params.Subject,
		AgentConfigID:  emailAgentConfigID(runCtx),
		AgentRunID:     runCtx.RunID,
		Identity:       runCtx.Identity,
	})
	if err != nil {
		return "", fmt.Errorf("failed to post reply draft: %w", err)
	}
	return "Draft reply proposed for human review. It is held for a teammate to approve and send — NOT yet sent to the recipient.", nil
}

// requireEmailContext checks the run can actually send mail: a notification client and an email-bridged conversation.
func requireEmailContext(runCtx *domain.HandlerRunContext) error {
	if runCtx.NotificationClient == nil {
		return fmt.Errorf("email sending is not available in this environment")
	}
	if runCtx.ConversationID == "" {
		return fmt.Errorf("email tools can only be used on an email-bridged conversation")
	}
	return nil
}

// emailAgentConfigID resolves the agent id the reply is attributed to. The conversation participant is keyed by the agent definition id, matching how chat replies attribute the agent.
func emailAgentConfigID(runCtx *domain.HandlerRunContext) string {
	if runCtx.Definition != nil {
		return runCtx.Definition.ID
	}
	return ""
}
