package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
)

// recallNoteLimit is how many notes a run reads.
const recallNoteLimit = 50

// withRecalledNotes appends the notes the run should know to its system prompt: the account's standing preferences,
// facts and instructions, and the notes on the run's conversation. Recall is best-effort: a run without notes still runs.
func (s *runnerSvc) withRecalledNotes(ctx context.Context, run *sqlc.AgentRun, accountID, systemPrompt string) string {
	var recordIDs []string
	if run != nil && run.ConversationID.Valid && run.ConversationID.String != "" {
		recordIDs = append(recordIDs, run.ConversationID.String)
	}
	notes, err := s.coreClient.RecallNotes(ctx, accountID, recordIDs, recallNoteLimit)
	if err != nil {
		slog.WarnContext(ctx, "Recalling notes failed; running without them", "account_id", accountID, "error", err)
		return systemPrompt
	}
	return appendRecalledNotes(systemPrompt, notes)
}

func appendRecalledNotes(systemPrompt string, notes []domain.RecalledNote) string {
	if len(notes) == 0 {
		return systemPrompt
	}
	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\nNotes from people and agents in this account, most important first:\n")
	for _, n := range notes {
		if n.RecordID != "" {
			fmt.Fprintf(&sb, "- [%s, on %s %s] %s\n", n.Type, n.RecordType, n.RecordID, n.Body)
		} else {
			fmt.Fprintf(&sb, "- [%s] %s\n", n.Type, n.Body)
		}
	}
	return sb.String()
}
