package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var emailLogRepoTracer = tracing.GetTracer("core-service.email_log_repository")

type emailLogRepoImpl struct {
	queries *sqlc.Queries
}

func NewEmailLogRepo(queries *sqlc.Queries) domain.EmailLogRepo {
	return &emailLogRepoImpl{queries: queries}
}

func emailLogCreatedAt(e *domain.EmailLog) time.Time { return e.CreatedAt }
func emailLogID(e *domain.EmailLog) string           { return e.ID }

func resolveSentByName(name, username, email gosql.NullString) *string {
	if name.Valid && name.String != "" {
		return &name.String
	}
	if username.Valid && username.String != "" {
		return &username.String
	}
	if email.Valid && email.String != "" {
		return &email.String
	}
	return nil
}

func mapEmailLogRow(
	id string,
	hasSent bool,
	subject, filename, sesMessageID, sentByID gosql.NullString,
	sentByName, sentByUsername, sentByEmail gosql.NullString,
	createdAt, updatedAt time.Time,
) *domain.EmailLog {
	el := &domain.EmailLog{
		ID:        id,
		HasSent:   hasSent,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if subject.Valid {
		el.Subject = &subject.String
	}
	if filename.Valid {
		el.Filename = &filename.String
	}
	if sesMessageID.Valid {
		el.SESMessageID = &sesMessageID.String
	}
	if sentByID.Valid {
		el.SentByID = &sentByID.String
	}
	el.SentByName = resolveSentByName(sentByName, sentByUsername, sentByEmail)

	return el
}

func mapForwardEmailLogRow(row sqlc.ListEmailLogsForwardRow) *domain.EmailLog {
	return mapEmailLogRow(
		row.ID, row.HasSent,
		row.Subject, row.Filename, row.SesMessageID, row.SentByID,
		row.SentByName, row.SentByUsername, row.SentByEmail,
		row.CreatedAt, row.UpdatedAt,
	)
}

func mapBackwardEmailLogRow(row sqlc.ListEmailLogsBackwardRow) *domain.EmailLog {
	return mapEmailLogRow(
		row.ID, row.HasSent,
		row.Subject, row.Filename, row.SesMessageID, row.SentByID,
		row.SentByName, row.SentByUsername, row.SentByEmail,
		row.CreatedAt, row.UpdatedAt,
	)
}

func mapGetEmailLogRow(row sqlc.GetEmailLogRow) *domain.EmailLog {
	return mapEmailLogRow(
		row.ID, row.HasSent,
		row.Subject, row.Filename, row.SesMessageID, row.SentByID,
		row.SentByName, row.SentByUsername, row.SentByEmail,
		row.CreatedAt, row.UpdatedAt,
	)
}

func buildEmailLogSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func (r *emailLogRepoImpl) fetchRecipients(ctx context.Context, emailLogID string) ([]string, *apierror.APIError) {
	rows, err := r.queries.GetEmailRecipientsByEmailLogID(ctx, gosql.NullString{String: emailLogID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	recipients := make([]string, len(rows))
	copy(recipients, rows)
	return recipients, nil
}

func (r *emailLogRepoImpl) enrichWithRecipients(ctx context.Context, emailLogs []*domain.EmailLog) *apierror.APIError {
	for _, el := range emailLogs {
		recipients, apiErr := r.fetchRecipients(ctx, el.ID)
		if apiErr != nil {
			return apiErr
		}
		el.Recipients = recipients
	}
	return nil
}

func (r *emailLogRepoImpl) List(ctx context.Context, params domain.ListEmailLogsParams) (*domain.ListEmailLogsResult, *apierror.APIError) {
	ctx, span := emailLogRepoTracer.Start(ctx, "repository.email_log.list")
	defer span.End()

	searchQuery := buildEmailLogSearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListEmailLogsBackward(ctx, sqlc.ListEmailLogsBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			emailLogs := make([]*domain.EmailLog, len(rows))
			for i, row := range rows {
				emailLogs[i] = mapBackwardEmailLogRow(row)
			}
			result, pageInfo := pagination.BuildPageString(emailLogs, params.Limit, cursorDir, emailLogCreatedAt, emailLogID)
			if apiErr := r.enrichWithRecipients(ctx, result); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListEmailLogsResult{EmailLogs: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListEmailLogsForward(ctx, sqlc.ListEmailLogsForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		emailLogs := make([]*domain.EmailLog, len(rows))
		for i, row := range rows {
			emailLogs[i] = mapForwardEmailLogRow(row)
		}
		result, pageInfo := pagination.BuildPageString(emailLogs, params.Limit, cursorDir, emailLogCreatedAt, emailLogID)
		if apiErr := r.enrichWithRecipients(ctx, result); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListEmailLogsResult{EmailLogs: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListEmailLogsForward(ctx, sqlc.ListEmailLogsForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	emailLogs := make([]*domain.EmailLog, len(rows))
	for i, row := range rows {
		emailLogs[i] = mapForwardEmailLogRow(row)
	}
	result, pageInfo := pagination.BuildPageString(emailLogs, params.Limit, cursorDir, emailLogCreatedAt, emailLogID)
	if apiErr := r.enrichWithRecipients(ctx, result); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListEmailLogsResult{EmailLogs: result, PageInfo: pageInfo}, nil
}

func (r *emailLogRepoImpl) Get(ctx context.Context, params domain.GetEmailLogParams) (*domain.EmailLog, *apierror.APIError) {
	ctx, span := emailLogRepoTracer.Start(ctx, "repository.email_log.get")
	defer span.End()

	row, err := r.queries.GetEmailLog(ctx, sqlc.GetEmailLogParams{
		ID:        params.EmailLogID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	emailLog := mapGetEmailLogRow(row)

	recipients, apiErr := r.fetchRecipients(ctx, emailLog.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	emailLog.Recipients = recipients

	return emailLog, nil
}
