package repository

import (
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/messaging"
)

type repoFactoryImpl struct {
	db *sqlc.Queries
}

func NewRepoFactory(db *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{db: db}
}

func (f *repoFactoryImpl) NewEmailLogRepo() domain.EmailLogRepo {
	return NewEmailLogRepo(f.db)
}

func (f *repoFactoryImpl) NewDeletedRecordRepo() domain.DeletedRecordRepo {
	return NewDeletedRecordRepo(f.db)
}

func (f *repoFactoryImpl) NewIdempotencyKeyRepo() domain.IdempotencyKeyRepo {
	return NewIdempotencyKeyRepo(f.db)
}

func (f *repoFactoryImpl) NewNotificationRepo() domain.NotificationRepo {
	return NewNotificationRepo(f.db)
}

func (f *repoFactoryImpl) NewAnnouncementRepo() domain.AnnouncementRepo {
	return NewAnnouncementRepo(f.db)
}

func (f *repoFactoryImpl) NewConversationRepo() domain.ConversationRepo {
	return NewConversationRepo(f.db)
}

func (f *repoFactoryImpl) NewParticipantRepo() domain.ParticipantRepo {
	return NewParticipantRepo(f.db)
}

func (f *repoFactoryImpl) NewMessageRepo() domain.MessageRepo {
	return NewMessageRepo(f.db)
}

func (f *repoFactoryImpl) NewBlockRepo() domain.BlockRepo {
	return NewBlockRepo(f.db)
}

func (f *repoFactoryImpl) NewMessageReportRepo() domain.MessageReportRepo {
	return NewMessageReportRepo(f.db)
}

func (f *repoFactoryImpl) NewNotificationPreferenceRepo() domain.NotificationPreferenceRepo {
	return NewNotificationPreferenceRepo(f.db)
}

func (f *repoFactoryImpl) NewMessageAttachmentRepo() domain.MessageAttachmentRepo {
	return NewMessageAttachmentRepo(f.db)
}

func (f *repoFactoryImpl) NewEmailDomainRepo() domain.EmailDomainRepo {
	return NewEmailDomainRepo(f.db)
}

func (f *repoFactoryImpl) NewEmailInboxRepo() domain.EmailInboxRepo {
	return NewEmailInboxRepo(f.db)
}

func (f *repoFactoryImpl) NewEmailMessageRepo() domain.EmailMessageRepo {
	return NewEmailMessageRepo(f.db)
}

func (f *repoFactoryImpl) NewSupportRouteRepo() domain.SupportRouteRepo {
	return NewSupportRouteRepo(f.db)
}

func (f *repoFactoryImpl) NewConversationLinkRepo() domain.ConversationLinkRepo {
	return NewConversationLinkRepo(f.db)
}

func (f *repoFactoryImpl) NewMessagingGroupRepo() domain.MessagingGroupRepo {
	return NewMessagingGroupRepo(f.db)
}

func (f *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(f.db)
}
