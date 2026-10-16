package repository

import (
	"context"
	"slices"
	"strings"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/db"
)

const notificationListColumns = `id, account_id, recipient_account_user_id, category, source_message_id, conversation_id,` +
	` title, body, template_key, template_params, link_resource_type, link_resource_id, priority, seen_at, read_at,` +
	` dismissed_at, metadata, created_at, updated_at, sender_type, sender_id, sender_name`

// notificationStatusPredicates are the timestamp states each status reads from.
var notificationStatusPredicates = map[string]string{
	string(constants.NotificationStatusUnseen):    "(seen_at IS NULL AND dismissed_at IS NULL)",
	string(constants.NotificationStatusSeen):      "(seen_at IS NOT NULL AND read_at IS NULL AND dismissed_at IS NULL)",
	string(constants.NotificationStatusRead):      "(read_at IS NOT NULL AND dismissed_at IS NULL)",
	string(constants.NotificationStatusDismissed): "(dismissed_at IS NOT NULL)",
}

// buildNotificationListQuery assembles one page of a person's notifications, newest first. Only the filters the caller
// set are emitted. Every status but dismissed reads as `dismissed_at IS NULL`, served by nf_recipient_feed_idx; a type
// filter by nf_recipient_category_created_idx and a sender filter by nf_recipient_sender_created_idx. A finer status
// filter (unseen alone, say) can still sort the person's rows (accepted residual, bounded by the person's feed).
func buildNotificationListQuery(f domain.NotificationListFilter) (string, []any) {
	var b strings.Builder
	b.WriteString("SELECT " + notificationListColumns + " FROM notification WHERE recipient_account_user_id = ?")
	args := []any{f.RecipientAccountUserID}
	in := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		b.WriteString(" AND " + column + " IN (" + strings.TrimSuffix(strings.Repeat("?,", len(values)), ",") + ")")
		for _, v := range values {
			args = append(args, v)
		}
	}
	in("category", f.Types)

	statuses := slices.Clone(f.Statuses)
	slices.Sort(statuses)
	statuses = slices.Compact(statuses)
	active := []string{string(constants.NotificationStatusRead), string(constants.NotificationStatusSeen), string(constants.NotificationStatusUnseen)}
	switch {
	case len(statuses) == 0, len(statuses) == len(notificationStatusPredicates):
	case slices.Equal(statuses, active):
		b.WriteString(" AND dismissed_at IS NULL")
	default:
		var preds []string
		for _, st := range statuses {
			if p, ok := notificationStatusPredicates[st]; ok {
				preds = append(preds, p)
			}
		}
		if len(preds) == 0 {
			b.WriteString(" AND FALSE")
		} else {
			b.WriteString(" AND (" + strings.Join(preds, " OR ") + ")")
		}
	}

	if f.Search != nil && *f.Search != "" {
		term := "%" + *f.Search + "%"
		b.WriteString(" AND (title LIKE ? OR body LIKE ?)")
		args = append(args, term, term)
	}
	in("sender_id", f.SenderIDs)
	in("sender_type", f.SenderTypes)
	if cursorID := db.NullStringPtr(f.CursorID); cursorID.Valid {
		at := db.NullTimePtr(f.CursorCreatedAt)
		b.WriteString(" AND (created_at < ? OR (created_at = ? AND id < ?))")
		args = append(args, at, at, cursorID)
	}
	b.WriteString(" ORDER BY created_at DESC, id DESC LIMIT ?")
	return b.String(), append(args, f.Limit)
}

func listNotificationRows(ctx context.Context, q *sqlc.Queries, f domain.NotificationListFilter) ([]sqlc.Notification, error) {
	query, args := buildNotificationListQuery(f)
	rows, err := q.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []sqlc.Notification
	for rows.Next() {
		var i sqlc.Notification
		if err := rows.Scan(
			&i.ID, &i.AccountID, &i.RecipientAccountUserID, &i.Category, &i.SourceMessageID, &i.ConversationID,
			&i.Title, &i.Body, &i.TemplateKey, &i.TemplateParams, &i.LinkResourceType, &i.LinkResourceID, &i.Priority,
			&i.SeenAt, &i.ReadAt, &i.DismissedAt, &i.Metadata, &i.CreatedAt, &i.UpdatedAt, &i.SenderType, &i.SenderID,
			&i.SenderName,
		); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
