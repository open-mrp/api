package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/email"
	"github.com/augno/api/services/notification-service/internal/infrastructure/aws"
	"github.com/augno/api/services/notification-service/internal/infrastructure/stub"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var notificationSvcTracer = tracing.GetTracer("notification-service.notification_service")

type notificationSvcImpl struct {
	emailLogRepo     domain.EmailLogRepo
	emailSender      domain.EmailSender
	templateRenderer email.TemplateRenderer
}

type NotificationSvcConfig struct {
	// EmailLogRepo (required) persists email send/delivery logs.
	EmailLogRepo domain.EmailLogRepo

	// EmailSender (required) sends outbound email.
	EmailSender domain.EmailSender

	// TemplateRenderer (required) renders email templates into message bodies.
	TemplateRenderer email.TemplateRenderer
}

func (c *NotificationSvcConfig) validate() error {
	if c.EmailLogRepo == nil {
		return fmt.Errorf("notification service: email log repo is required")
	}
	if c.EmailSender == nil {
		return fmt.Errorf("notification service: email sender is required")
	}
	if c.TemplateRenderer == nil {
		return fmt.Errorf("notification service: template renderer is required")
	}
	return nil
}

func NewNotificationSvc(config *NotificationSvcConfig) domain.NotificationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &notificationSvcImpl{
		emailLogRepo:     config.EmailLogRepo,
		emailSender:      config.EmailSender,
		templateRenderer: config.TemplateRenderer,
	}
}

func BuildNotificationSvcConfig(repoFactory domain.RepoFactory, platformMode constants.PlatformMode, awsRegion string, templateRenderer email.TemplateRenderer) (*NotificationSvcConfig, *apierror.APIError) {
	var emailSender domain.EmailSender
	if platformMode.IsTest() {
		emailSender = &stub.EmailSender{}
	} else {
		var apiErr *apierror.APIError
		emailSender, apiErr = aws.NewSESEmailSender(context.Background(), platformMode, awsRegion)
		if apiErr != nil {
			return nil, apiErr
		}
	}

	return &NotificationSvcConfig{
		EmailLogRepo:     repoFactory.NewEmailLogRepo(),
		EmailSender:      emailSender,
		TemplateRenderer: templateRenderer,
	}, nil
}

// SendEmail sends an email via SES, or logs a suppressed entry if the request originates from a sandbox.
//
// 1. If the request is from a sandbox account, log a suppressed email record and return without sending.
// 2. Otherwise, send the email through the configured email sender (SES).
// 3. Return the SES message ID on success.
func (s *notificationSvcImpl) SendEmail(ctx context.Context, data domain.EmailSendData) (*string, *apierror.APIError) {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.send_email")
	defer span.End()

	if s.isSandboxRequest(ctx) {
		return s.logSuppressedEmail(ctx, data)
	}

	sesMessageID, apiErr := s.emailSender.Send(ctx, domain.EmailData{
		To:         data.To,
		Subject:    data.Subject,
		Body:       data.Body,
		SendAs:     data.SendAs,
		Attachment: data.Attachment,
		Filename:   data.Filename,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	return sesMessageID, nil
}

// LogEmail records a sent email in the email log, deduplicating by SES message ID.
//
// 1. Check if an email log entry already exists for the given SES message ID.
// 2. If a duplicate is found, return early (idempotent).
// 3. Generate a new email log ID and persist the log entry.
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
		Subject:      new(data.Subject),
		Filename:     data.Filename,
		SesMessageID: new(data.SesMessageID),
		Recipients:   data.To,
	}

	return s.emailLogRepo.Create(ctx, emailLog)
}

// LogFailedEmail records an email that could not be sent, so a failed send is visible in the email log rather than vanishing. It writes HasSent=false and no SES message ID, because the send never reached SES.
//
// Delivery is retried, so this deduplicates on the outbox message ID: every attempt for the same message resolves to the same placeholder and only the first attempt writes a row.
func (s *notificationSvcImpl) LogFailedEmail(ctx context.Context, messageID string, data domain.EmailSendData) *apierror.APIError {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.log_failed_email")
	defer span.End()

	// Without a message ID there is no stable key, and each retry would append another row for the same email.
	if messageID == "" {
		return nil
	}

	placeholderID := "failed_" + messageID

	existing, apiErr := s.emailLogRepo.FindBySesMessageID(ctx, placeholderID)
	if apiErr != nil && apiErr.Code != apierror.ErrorCodeResourceNotFound {
		return apiErr
	}

	if existing != nil {
		return nil
	}

	logID, apiErr := id.GenID(id.EmailLogIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	accountID := ""
	if data.AccountID != nil {
		accountID = *data.AccountID
	}

	emailLog := &domain.EmailLog{
		ID:           logID,
		HasSent:      false,
		AccountID:    accountID,
		SentByID:     data.SentByID,
		Subject:      new(data.Subject),
		Filename:     data.Filename,
		SesMessageID: &placeholderID,
		Recipients:   data.To,
	}

	return s.emailLogRepo.Create(ctx, emailLog)
}

// SendEnterpriseRequest sends an enterprise upgrade request email to the sales team.
//
// 1. If the request originates from a sandbox, skip sending and return immediately.
// 2. Render the enterprise request email template with account and requester details.
// 3. Send the rendered email to the sales team address.
func (s *notificationSvcImpl) SendEnterpriseRequest(ctx context.Context, req *domain.EnterpriseRequestData) *apierror.APIError {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.send_enterprise_request")
	defer span.End()

	if s.isSandboxRequest(ctx) {
		return nil
	}

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

func (s *notificationSvcImpl) isSandboxRequest(ctx context.Context) bool {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	return ok && identity.AccountMode == constants.AccountModeSandbox
}

// logSuppressedEmail creates an email log entry for a sandbox-suppressed email without actually sending it. The log records HasSent=false so it is clear the email was never delivered.
func (s *notificationSvcImpl) logSuppressedEmail(ctx context.Context, data domain.EmailSendData) (*string, *apierror.APIError) {
	ctx, span := notificationSvcTracer.Start(ctx, "service.notification.sandbox_email_suppressed")
	defer span.End()

	logID, apiErr := id.GenID(id.EmailLogIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	placeholderID := "sandbox_" + logID

	accountID := ""
	if data.AccountID != nil {
		accountID = *data.AccountID
	}

	emailLog := &domain.EmailLog{
		ID:           logID,
		HasSent:      false,
		AccountID:    accountID,
		SentByID:     data.SentByID,
		Subject:      &data.Subject,
		SesMessageID: &placeholderID,
		Recipients:   data.To,
	}

	if apiErr := s.emailLogRepo.Create(ctx, emailLog); apiErr != nil {
		return nil, apiErr
	}

	return &placeholderID, nil
}
