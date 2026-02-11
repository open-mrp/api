package service

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/email"
	"github.com/augno/api/services/notification-service/internal/infrastructure/aws"
	"github.com/augno/api/services/notification-service/internal/infrastructure/repository"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/tracing"
)

var notificationSvcTracer = tracing.GetTracer("notification-service.notification_service")

type notificationSvcImpl struct {
	emailLogRepo     domain.EmailLogRepo
	emailSender      domain.EmailSender
	templateRenderer email.TemplateRenderer
}

type NotificationSvcConfig struct {
	EmailLogRepo     domain.EmailLogRepo
	EmailSender      domain.EmailSender
	TemplateRenderer email.TemplateRenderer
}

func NewNotificationSvc(config NotificationSvcConfig) domain.NotificationSvc {
	return &notificationSvcImpl{
		emailLogRepo:     config.EmailLogRepo,
		emailSender:      config.EmailSender,
		templateRenderer: config.TemplateRenderer,
	}
}

func DefaultNotificationSvcConfig(queries *sqlc.Queries, awsRegion string, templateRenderer email.TemplateRenderer) (NotificationSvcConfig, *apierror.APIError) {
	emailSender, apiErr := aws.NewSESEmailSender(context.Background(), constants.PlatformModeProduction, awsRegion)
	if apiErr != nil {
		return NotificationSvcConfig{}, apiErr
	}

	return NotificationSvcConfig{
		EmailLogRepo:     repository.NewEmailLogRepo(queries),
		EmailSender:      emailSender,
		TemplateRenderer: templateRenderer,
	}, nil
}

func NewDefaultNotificationSvc(queries *sqlc.Queries, awsRegion string, templateRenderer email.TemplateRenderer) (domain.NotificationSvc, *apierror.APIError) {
	config, apiErr := DefaultNotificationSvcConfig(queries, awsRegion, templateRenderer)
	if apiErr != nil {
		return nil, apiErr
	}
	return NewNotificationSvc(config), nil
}

func (s *notificationSvcImpl) SendEmail(ctx context.Context, data domain.EmailSendData) (*string, *apierror.APIError) {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.send_email")
	defer span.End()

	sesMessageID, apiErr := s.emailSender.Send(ctx, domain.EmailData{
		To:      data.To,
		Subject: data.Subject,
		Body:    data.Body,
		SendAs:  data.SendAs,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	return sesMessageID, nil
}

func (s *notificationSvcImpl) LogEmail(ctx context.Context, data domain.EmailLogData) *apierror.APIError {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.log_email")
	defer span.End()

	existing, apiErr := s.emailLogRepo.FindBySesMessageID(ctx, data.SesMessageID)
	if apiErr != nil && apiErr.Code != apierror.ErrorCodeResourceNotFound {
		return apiErr
	}

	if existing != nil {
		return nil
	}

	id, apiErr := id.GenID(id.EmailLogIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	accountID := ""
	if data.AccountID != nil {
		accountID = *data.AccountID
	}

	emailLog := &domain.EmailLog{
		ID:           id,
		HasSent:      true,
		AccountID:    accountID,
		SentByID:     data.SentByID,
		Subject:      ptrutil.Ptr(data.Subject),
		Filename:     data.Filename,
		SesMessageID: ptrutil.Ptr(data.SesMessageID),
	}

	return s.emailLogRepo.Create(ctx, emailLog)
}

// SendEnterpriseRequest sends an enterprise upgrade request email to sales
func (s *notificationSvcImpl) SendEnterpriseRequest(ctx context.Context, req *domain.EnterpriseRequestData) *apierror.APIError {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.send_enterprise_request")
	defer span.End()

	subject := "Enterprise Upgrade Request: " + req.AccountName

	body, apiErr := s.templateRenderer.RenderTemplate(ctx, constants.EmailTemplateEnterpriseRequest, map[string]any{
		"AccountID":       req.AccountID,
		"AccountName":     req.AccountName,
		"CurrentPlanName": req.CurrentPlanName,
		"RequesterName":   req.RequesterName,
		"RequesterEmail":  req.RequesterEmail,
	})
	if apiErr != nil {
		return apiErr
	}

	// Send to sales team
	salesEmail := "sales@augno.com"
	_, apiErr = s.emailSender.Send(ctx, domain.EmailData{
		To:      []string{salesEmail},
		Subject: subject,
		Body:    body,
		SendAs:  nil,
	})
	if apiErr != nil {
		return apiErr
	}

	return nil
}
