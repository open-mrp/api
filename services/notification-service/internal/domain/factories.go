package domain

import "github.com/open-mrp/api/shared/messaging"

// RepoFactory constructs repository implementations for a single database session (typically *sqlc.Queries).
type RepoFactory interface {
	NewEmailLogRepo() EmailLogRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
	NewNotificationRepo() NotificationRepo
	NewAnnouncementRepo() AnnouncementRepo
	NewConversationRepo() ConversationRepo
	NewParticipantRepo() ParticipantRepo
	NewMessageRepo() MessageRepo
	NewBlockRepo() BlockRepo
	NewMessageReportRepo() MessageReportRepo
	NewNotificationPreferenceRepo() NotificationPreferenceRepo
	NewMessageAttachmentRepo() MessageAttachmentRepo
	NewEmailDomainRepo() EmailDomainRepo
	NewEmailInboxRepo() EmailInboxRepo
	NewAccountEmailSenderRepo() AccountEmailSenderRepo
	NewEmailMessageRepo() EmailMessageRepo
	NewSupportRouteRepo() SupportRouteRepo
	NewConversationLinkRepo() ConversationLinkRepo
	NewMessagingGroupRepo() MessagingGroupRepo
	NewDeletedRecordRepo() DeletedRecordRepo
	NewOutboxRepo() messaging.OutboxRepo
}
