package repository

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/notification-service/internal/reaper"
	"github.com/augno/api/shared/tracing"
)

var messagingReaperRepoTracer = tracing.GetTracer("notification-service.messaging_reaper_repository")

type messagingReaperRepoImpl struct {
	queries *sqlc.Queries
}

// NewMessagingReaperRepo constructs the retention-worker repository.
func NewMessagingReaperRepo(queries *sqlc.Queries) reaper.MessagingReaperRepo {
	return &messagingReaperRepoImpl{queries: queries}
}

func (r *messagingReaperRepoImpl) PurgeActionedNotifications(ctx context.Context, retentionHours int, limit int32) (int64, error) {
	ctx, span := messagingReaperRepoTracer.Start(ctx, "repository.messaging_reaper.purge_actioned_notifications")
	defer span.End()

	count, err := r.queries.PurgeActionedNotifications(ctx, sqlc.PurgeActionedNotificationsParams{
		Column1: retentionHours,
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	return count, nil
}

func (r *messagingReaperRepoImpl) PurgeStaleNotifications(ctx context.Context, capHours int, limit int32) (int64, error) {
	ctx, span := messagingReaperRepoTracer.Start(ctx, "repository.messaging_reaper.purge_stale_notifications")
	defer span.End()

	count, err := r.queries.PurgeStaleNotifications(ctx, sqlc.PurgeStaleNotificationsParams{
		Column1: capHours,
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	return count, nil
}

func (r *messagingReaperRepoImpl) PurgeExpiredAnnouncements(ctx context.Context, retentionHours int, limit int32) (int64, error) {
	ctx, span := messagingReaperRepoTracer.Start(ctx, "repository.messaging_reaper.purge_expired_announcements")
	defer span.End()

	count, err := r.queries.PurgeExpiredAnnouncements(ctx, sqlc.PurgeExpiredAnnouncementsParams{
		Column1: retentionHours,
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	return count, nil
}

func (r *messagingReaperRepoImpl) PurgeOrphanedAnnouncementReceipts(ctx context.Context, limit int32) (int64, error) {
	ctx, span := messagingReaperRepoTracer.Start(ctx, "repository.messaging_reaper.purge_orphaned_announcement_receipts")
	defer span.End()

	count, err := r.queries.PurgeOrphanedAnnouncementReceipts(ctx, limit)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	return count, nil
}

func (r *messagingReaperRepoImpl) ListPurgeableMessageAttachments(ctx context.Context, retentionHours int, limit int32) ([]reaper.AttachmentPurgeRef, error) {
	ctx, span := messagingReaperRepoTracer.Start(ctx, "repository.messaging_reaper.list_purgeable_message_attachments")
	defer span.End()

	rows, err := r.queries.ListPurgeableMessageAttachments(ctx, sqlc.ListPurgeableMessageAttachmentsParams{
		Column1: retentionHours,
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	refs := make([]reaper.AttachmentPurgeRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, reaper.AttachmentPurgeRef{ID: row.ID, S3Key: row.S3Key.String})
	}
	return refs, nil
}

func (r *messagingReaperRepoImpl) DeleteMessageAttachmentByID(ctx context.Context, id string) error {
	ctx, span := messagingReaperRepoTracer.Start(ctx, "repository.messaging_reaper.delete_message_attachment_by_id")
	defer span.End()

	if err := r.queries.DeleteMessageAttachmentByID(ctx, id); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (r *messagingReaperRepoImpl) PurgeTombstonedMessages(ctx context.Context, retentionHours int, limit int32) (int64, error) {
	ctx, span := messagingReaperRepoTracer.Start(ctx, "repository.messaging_reaper.purge_tombstoned_messages")
	defer span.End()

	count, err := r.queries.PurgeTombstonedMessages(ctx, sqlc.PurgeTombstonedMessagesParams{
		Column1: retentionHours,
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	return count, nil
}
