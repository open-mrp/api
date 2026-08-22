package repository

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var emailLogRepoTracer = tracing.GetTracer("notification-service.email_log_repository")

type emailLogRepoImpl struct {
	db *sqlc.Queries
}

func NewEmailLogRepo(db *sqlc.Queries) domain.EmailLogRepo {
	return &emailLogRepoImpl{db: db}
}

func (r *emailLogRepoImpl) Create(ctx context.Context, emailLog *domain.EmailLog) *apierror.APIError {
	ctx, span := emailLogRepoTracer.Start(ctx, "repository.email_log.create")
	defer span.End()

	err := r.db.CreateEmailLog(ctx, sqlc.CreateEmailLogParams{
		ID:           emailLog.ID,
		HasSent:      emailLog.HasSent,
		AccountID:    emailLog.AccountID,
		SentByID:     db.NullStringPtr(emailLog.SentByID),
		Subject:      db.NullStringPtr(emailLog.Subject),
		Filename:     db.NullStringPtr(emailLog.Filename),
		SesMessageID: db.NullStringPtr(emailLog.SesMessageID),
	})
	if err != nil {
		if db.IsDuplicateEntry(err) {
			return nil // idempotent — log already exists
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	for _, email := range emailLog.Recipients {
		if email == "" {
			continue
		}

		recipientID, apiErr := id.GenID(id.EmailRecipientIDPrefix, nil)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		if err := r.db.CreateEmailRecipient(ctx, sqlc.CreateEmailRecipientParams{
			ID:         recipientID,
			Email:      email,
			EmailLogID: db.NullString(emailLog.ID),
		}); err != nil {
			if db.IsDuplicateEntry(err) {
				continue
			}
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}

	return nil
}

func (r *emailLogRepoImpl) FindBySesMessageID(ctx context.Context, sesMessageID string) (*domain.EmailLog, *apierror.APIError) {
	ctx, span := emailLogRepoTracer.Start(ctx, "repository.email_log.find_by_ses_message_id")
	defer span.End()

	row, err := r.db.FindEmailLogBySesMessageID(ctx, db.NullString(sesMessageID))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	emailLog := &domain.EmailLog{
		ID:           row.ID,
		HasSent:      row.HasSent,
		AccountID:    row.AccountID,
		SentByID:     db.StringFromNullString(row.SentByID),
		Subject:      db.StringFromNullString(row.Subject),
		Filename:     db.StringFromNullString(row.Filename),
		SesMessageID: db.StringFromNullString(row.SesMessageID),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}

	return emailLog, nil
}
