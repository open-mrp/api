package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db"
)

// conversationListColumns is the SELECT list of the caller's conversation list, in the order
// scanConversationListRow reads it.
const conversationListColumns = `c.id, c.account_id, c.type, c.audience, c.title, c.group_id, c.topic_resource_type,` +
	` c.topic_resource_id, c.created_by_participant_id, c.next_sequence, c.last_message_id, c.last_message_at,` +
	` c.is_archived, c.legal_hold, c.workflow_status, c.assignee_resource_type, c.assignee_resource_id,` +
	` c.email_inbox_id, c.email_external_address, c.metadata, c.created_at, c.updated_at, c.continues_conversation_id,` +
	` p.id, p.last_read_sequence, p.hidden_at`

// conversationListRow is one conversation of the caller's list with the caller's participant row.
type conversationListRow struct {
	sqlc.Conversation
	ParticipantID               string
	ParticipantLastReadSequence int64
	ParticipantHiddenAt         sql.NullTime
}

// buildConversationListQuery assembles one page of the caller's active internal conversations, most
// recently active first. Only the predicates the filter sets are emitted: an `(? IS NULL OR ...)` guard
// is not sargable, and kept the type filter off conversation_account_type_lastmsg_idx.
//
// The membership scope lives on conversation_participant while the sort key lives on conversation, so
// with no type filter the page is sorted from the caller's conversations (an accepted residual; see
// docs/patterns/performant-list-endpoint-patterns.md).
func buildConversationListQuery(f domain.ConversationListFilter) (string, []any) {
	var b strings.Builder
	b.WriteString("SELECT " + conversationListColumns + " FROM conversation c" +
		" JOIN conversation_participant p ON p.conversation_id = c.id AND p.account_user_id = ?" +
		" WHERE c.account_id = ? AND p.membership = 'active'" +
		// Internal conversations only: customer-facing cases (portal support, email-bridged threads)
		// live in the support inbox, not a staff member's Messages list.
		" AND c.audience = 'internal'")
	args := []any{db.NullStringPtr(&f.AccountUserID), f.AccountID}

	switch f.Status {
	case "", "active":
		b.WriteString(" AND p.hidden_at IS NULL")
	case "hidden":
		b.WriteString(" AND p.hidden_at IS NOT NULL")
	case "all":
	default:
		b.WriteString(" AND FALSE")
	}
	if f.Type != nil && *f.Type != "" {
		b.WriteString(" AND c.type = ?")
		args = append(args, *f.Type)
	}
	if n := len(f.ParticipantAccountUserIDs); n > 0 {
		// Conversations that also seat any of the given account users, served by cvpt_conv_user_uq.
		b.WriteString(" AND EXISTS (SELECT 1 FROM conversation_participant op" +
			" WHERE op.conversation_id = c.id AND op.membership = 'active'" +
			" AND op.account_user_id IN (" + strings.TrimSuffix(strings.Repeat("?,", n), ",") + "))")
		for _, id := range f.ParticipantAccountUserIDs {
			args = append(args, id)
		}
	}
	// Conversations with no messages have no last_message_at and sort after every other row, so a
	// cursor taken in that tail pages them by id alone.
	if cursorID := db.NullStringPtr(f.CursorID); cursorID.Valid {
		if f.CursorInNullTail {
			b.WriteString(" AND c.last_message_at IS NULL AND c.id < ?")
			args = append(args, cursorID)
		} else {
			at := db.NullTimePtr(f.CursorLastMessageAt)
			b.WriteString(" AND (c.last_message_at < ? OR (c.last_message_at = ? AND c.id < ?) OR c.last_message_at IS NULL)")
			args = append(args, at, at, cursorID)
		}
	}
	b.WriteString(" ORDER BY c.last_message_at DESC, c.id DESC LIMIT ?")
	return b.String(), append(args, f.Limit)
}

func listConversationRows(ctx context.Context, q *sqlc.Queries, f domain.ConversationListFilter) ([]conversationListRow, error) {
	query, args := buildConversationListQuery(f)
	rows, err := q.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []conversationListRow
	for rows.Next() {
		var i conversationListRow
		if err := rows.Scan(
			&i.ID, &i.AccountID, &i.Type, &i.Audience, &i.Title, &i.GroupID, &i.TopicResourceType,
			&i.TopicResourceID, &i.CreatedByParticipantID, &i.NextSequence, &i.LastMessageID, &i.LastMessageAt,
			&i.IsArchived, &i.LegalHold, &i.WorkflowStatus, &i.AssigneeResourceType, &i.AssigneeResourceID,
			&i.EmailInboxID, &i.EmailExternalAddress, &i.Metadata, &i.CreatedAt, &i.UpdatedAt, &i.ContinuesConversationID,
			&i.ParticipantID, &i.ParticipantLastReadSequence, &i.ParticipantHiddenAt,
		); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
