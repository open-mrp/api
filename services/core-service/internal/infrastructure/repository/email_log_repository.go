package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
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
	sentByAPIKeyID, sentByAPIKeyName gosql.NullString,
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
		// Resolve the actor's type from which reference table the sender id
		// matched: an api_key type_id join takes precedence, otherwise the
		// sender is a user. The type must reflect the true actor kind — it was
		// previously hardcoded to "user", mislabeling api-key-sent emails.
		actor := &domain.EmailLogActor{
			ID:        sentByID.String,
			ActorType: string(constants.ActorTypeUser),
			Name:      resolveSentByName(sentByName, sentByUsername, sentByEmail),
		}
		if sentByEmail.Valid && sentByEmail.String != "" {
			handle := sentByEmail.String
			actor.Handle = &handle
		}
		if sentByAPIKeyID.Valid {
			actor.ActorType = string(constants.ActorTypeAPIKey)
			actor.Name = nil
			actor.Handle = nil
			if sentByAPIKeyName.Valid && sentByAPIKeyName.String != "" {
				actor.Name = &sentByAPIKeyName.String
			}
		}
		el.SentBy = actor
	}

	return el
}

func mapGetEmailLogRow(row sqlc.GetEmailLogRow) *domain.EmailLog {
	el := mapEmailLogRow(
		row.ID, row.HasSent,
		row.Subject, row.Filename, row.SesMessageID, row.SentByID,
		row.SentByName, row.SentByUsername, row.SentByEmail,
		row.SentByApiKeyID, row.SentByApiKeyName,
		row.CreatedAt, row.UpdatedAt,
	)
	el.RelatedType, el.RelatedID = db.StringFromNullString(row.RelatedType), db.StringFromNullString(row.RelatedID)
	return el
}

func (r *emailLogRepoImpl) List(ctx context.Context, params domain.ListEmailLogsParams) (*domain.ListEmailLogsResult, *apierror.APIError) {
	ctx, span := emailLogRepoTracer.Start(ctx, "repository.email_log.list")
	defer span.End()

	var cursorDir *pagination.Direction
	var cursorCreatedAt gosql.NullTime
	var cursorID gosql.NullString
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction
		cursorCreatedAt = gosql.NullTime{Time: cur.OccurredAt, Valid: true}
		cursorID = gosql.NullString{String: cur.ID, Valid: true}
	}
	dir := pagination.DirectionForward
	if cursorDir != nil {
		dir = *cursorDir
	}

	page := buildEmailLogListPage(params, dir, cursorCreatedAt, cursorID, params.Limit+1)
	query, args := buildEmailLogListQuery(page, dir)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer rows.Close()
	emailLogs, err := scanEmailLogListRows(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, pageInfo := pagination.BuildPageString(emailLogs, params.Limit, cursorDir, emailLogCreatedAt, emailLogID)
	if apiErr := r.enrichWithRecipients(ctx, result); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListEmailLogsResult{EmailLogs: result, PageInfo: pageInfo}, nil
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
