package repository

import (
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
)

// emailLogListColumns is the SELECT list for the email log listing, in the order scanEmailLogListRows reads it.
const emailLogListColumns = `el.id, el.has_sent, el.subject, el.filename, el.ses_message_id, el.sent_by_id, ` +
	`u.name, u.username, u.email, ak.type_id, ak.name, el.related_type, el.related_id, el.created_at, el.updated_at`

// buildEmailLogListPage chooses a page of the account's email logs. Only the filters the caller sent are emitted.
// Forward pages older (DESC), backward newer (ASC).
func buildEmailLogListPage(
	params domain.ListEmailLogsParams,
	dir pagination.Direction,
	cursorCreatedAt gosql.NullTime,
	cursorID gosql.NullString,
	limit int32,
) keysetPage {
	page := keysetPage{
		table: "email_log", alias: "el", sortColumn: "created_at",
		createdIndex: "email_log_account_created_idx",
		filters: []keysetFilter{
			{column: "el.related_id", index: "email_log_account_related_created_idx", values: params.RelatedIDs},
		},
		where: []string{"el.account_id = ?"},
		args:  []any{params.AccountID},
		desc:  dir != pagination.DirectionBackward,
		limit: limit,
	}
	if params.HasSent != nil {
		page.where, page.args = append(page.where, "el.has_sent = ?"), append(page.args, *params.HasSent)
	}
	if params.CreatedAfter != nil {
		page.where, page.args = append(page.where, "el.created_at >= ?"), append(page.args, *params.CreatedAfter)
	}
	if params.CreatedBefore != nil {
		page.where, page.args = append(page.where, "el.created_at < ?"), append(page.args, *params.CreatedBefore)
	}
	if params.Query != nil && *params.Query != "" {
		like := "%" + db.EscapeLike(*params.Query) + "%"
		page.where = append(page.where, "(el.subject LIKE ? OR EXISTS (SELECT 1 FROM email_recipient er WHERE er.email_log_id = el.id AND er.email LIKE ?))")
		page.args = append(page.args, like, like)
	}
	if cursorCreatedAt.Valid {
		if dir == pagination.DirectionBackward {
			page.where = append(page.where, "(el.created_at > ? OR (el.created_at = ? AND el.id > ?))")
		} else {
			page.where = append(page.where, "(el.created_at < ? OR (el.created_at = ? AND el.id < ?))")
		}
		page.args = append(page.args, cursorCreatedAt.Time, cursorCreatedAt.Time, cursorID.String)
	}
	return page
}

// buildEmailLogListQuery wraps a settled page in the listing's sender joins.
func buildEmailLogListQuery(page keysetPage, dir pagination.Direction) (string, []any) {
	orderBy := " ORDER BY el.created_at DESC, el.id DESC"
	if dir == pagination.DirectionBackward {
		orderBy = " ORDER BY el.created_at ASC, el.id ASC"
	}
	pageSQL, args := page.sql()
	return "SELECT STRAIGHT_JOIN " + emailLogListColumns + " FROM (" + pageSQL + ") page" +
		" JOIN email_log el ON el.id = page.id" +
		" LEFT JOIN `user` u ON el.sent_by_id = u.id" +
		" LEFT JOIN api_key ak ON el.sent_by_id = ak.type_id" + orderBy, args
}

func scanEmailLogListRows(rows *gosql.Rows) ([]*domain.EmailLog, error) {
	var out []*domain.EmailLog
	for rows.Next() {
		var (
			id                                                   string
			hasSent                                              bool
			subject, filename, sesMessageID, sentByID            gosql.NullString
			sentByName, sentByUsername, sentByEmail              gosql.NullString
			sentByAPIKeyID, sentByAPIKeyName, relType, relatedID gosql.NullString
			createdAt, updatedAt                                 gosql.NullTime
		)
		if err := rows.Scan(&id, &hasSent, &subject, &filename, &sesMessageID, &sentByID,
			&sentByName, &sentByUsername, &sentByEmail, &sentByAPIKeyID, &sentByAPIKeyName,
			&relType, &relatedID, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		el := mapEmailLogRow(id, hasSent, subject, filename, sesMessageID, sentByID,
			sentByName, sentByUsername, sentByEmail, sentByAPIKeyID, sentByAPIKeyName,
			createdAt.Time, updatedAt.Time)
		el.RelatedType, el.RelatedID = db.StringFromNullString(relType), db.StringFromNullString(relatedID)
		out = append(out, el)
	}
	return out, rows.Err()
}
