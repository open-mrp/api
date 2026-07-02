package repository

import (
	"context"
	"database/sql"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var messageReportRepoTracer = tracing.GetTracer("notification-service.message_report_repository")

type messageReportRepoImpl struct {
	db *sqlc.Queries
}

func NewMessageReportRepo(db *sqlc.Queries) domain.MessageReportRepo {
	return &messageReportRepoImpl{db: db}
}

func (r *messageReportRepoImpl) Create(ctx context.Context, report *domain.MessageReport) *apierror.APIError {
	ctx, span := messageReportRepoTracer.Start(ctx, "repository.message_report.create")
	defer span.End()

	var messageID sql.NullString
	if report.MessageID != "" {
		messageID = sql.NullString{String: report.MessageID, Valid: true}
	}
	err := r.db.CreateMessagingReport(ctx, sqlc.CreateMessagingReportParams{
		ID:                    report.ID,
		AccountID:             report.AccountID,
		ConversationID:        report.ConversationID,
		MessageID:             messageID,
		ReporterAccountUserID: report.ReporterAccountUserID,
		Reason:                report.Reason,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
