package service

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/aws"
	"github.com/augno/api/services/notification-service/internal/infrastructure/repository"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/tracing"
)

var notificationSvcTracer = tracing.GetTracer("notification-service.notification_service")

type notificationSvcImpl struct {
	emailLogRepo domain.EmailLogRepo
	emailSender  domain.EmailSender
}

type NotificationSvcConfig struct {
	EmailLogRepo domain.EmailLogRepo
	EmailSender  domain.EmailSender
}

func NewNotificationSvc(config NotificationSvcConfig) domain.NotificationSvc {
	return &notificationSvcImpl{
		emailLogRepo: config.EmailLogRepo,
		emailSender:  config.EmailSender,
	}
}

func DefaultNotificationSvcConfig(queries *sqlc.Queries, awsRegion string) (NotificationSvcConfig, *contracts.APIError) {
	emailSender, apiErr := aws.NewSESEmailSender(context.Background(), constants.PlatformModeProduction, awsRegion)
	if apiErr != nil {
		return NotificationSvcConfig{}, apiErr
	}

	return NotificationSvcConfig{
		EmailLogRepo: repository.NewEmailLogRepo(queries),
		EmailSender:  emailSender,
	}, nil
}

func NewDefaultNotificationSvc(queries *sqlc.Queries, awsRegion string) (domain.NotificationSvc, *contracts.APIError) {
	config, apiErr := DefaultNotificationSvcConfig(queries, awsRegion)
	if apiErr != nil {
		return nil, apiErr
	}
	return NewNotificationSvc(config), nil
}

func (s *notificationSvcImpl) SendEmail(ctx context.Context, to []string, subject, body string, isBodyHtml bool, sendAs *string, accountID string, sentByID *string) (*string, *contracts.APIError) {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.send_email")
	defer span.End()

	sesMessageID, apiErr := s.emailSender.Send(ctx, to, subject, body, isBodyHtml, sendAs)
	if apiErr != nil {
		return nil, apiErr
	}

	return sesMessageID, nil
}

func (s *notificationSvcImpl) LogEmail(ctx context.Context, sesMessageID string, accountID string, sentByID *string, subject string, filename *string) *contracts.APIError {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.log_email")
	defer span.End()

	existing, apiErr := s.emailLogRepo.FindBySesMessageID(ctx, sesMessageID)
	if apiErr != nil && apiErr.Code != contracts.ErrorCodeResourceNotFound {
		return apiErr
	}

	if existing != nil {
		return nil
	}

	id, apiErr := id.GenID(id.EmailLogIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	emailLog := &domain.EmailLog{
		ID:           id,
		HasSent:      true,
		AccountID:    accountID,
		SentByID:     sentByID,
		Subject:      ptrutil.String(subject),
		Filename:     filename,
		SesMessageID: ptrutil.String(sesMessageID),
	}

	return s.emailLogRepo.Create(ctx, emailLog)
}
