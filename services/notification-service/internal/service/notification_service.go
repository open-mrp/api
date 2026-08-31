package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/email"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/aws"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/stub"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var notificationSvcTracer = tracing.GetTracer("notification-service.notification_service")

type notificationSvcImpl struct {
	emailLogRepo     domain.EmailLogRepo
	emailSender      domain.EmailSender
	templateRenderer email.TemplateRenderer
	// senderRepo resolves an account's own outbound identity. Nil leaves every send on the platform address.
	senderRepo domain.AccountEmailSenderRepo
	// merchantSender posts mail addressed from a customer's own domain. Those domains are DKIM-verified in the inbound region, and SES rejects an identity it does not hold in the region the request lands in, so a merchant-addressed message cannot go out over emailSender (the platform's send region). Nil leaves every send on the platform address.
	merchantSender domain.EmailSender
}

type NotificationSvcConfig struct {
	// EmailLogRepo (required) persists email send/delivery logs.
	EmailLogRepo domain.EmailLogRepo

	// EmailSender (required) sends outbound email.
	EmailSender domain.EmailSender

	// TemplateRenderer (required) renders email templates into message bodies.
	TemplateRenderer email.TemplateRenderer

	// SenderRepo (optional; default: nil) resolves an account's configured outbound identity. When nil, all mail sends from the platform address.
	SenderRepo domain.AccountEmailSenderRepo

	// MerchantSender (optional; default: nil) sends mail addressed from a customer's own verified domain, in the region those domains are verified in. When nil, all mail sends from the platform address.
	MerchantSender domain.EmailSender
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
		senderRepo:       config.SenderRepo,
		merchantSender:   config.MerchantSender,
	}
}

// BuildNotificationSvcConfig wires the notification service. merchantSender posts mail from accounts' own verified domains and must be an SES client in the region those domains are verified in — not awsRegion, where only the platform identity exists. Nil is allowed (test mode, or a deployment without the bridge) and leaves every send on the platform address.
func BuildNotificationSvcConfig(repoFactory domain.RepoFactory, platformMode constants.PlatformMode, awsRegion string, templateRenderer email.TemplateRenderer, merchantSender domain.EmailSender) (*NotificationSvcConfig, *apierror.APIError) {
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
		SenderRepo:       repoFactory.NewAccountEmailSenderRepo(),
		MerchantSender:   merchantSender,
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

	emailData := domain.EmailData{
		To:         data.To,
		Subject:    data.Subject,
		Body:       data.Body,
		SendAs:     data.SendAs,
		Attachment: data.Attachment,
		Filename:   data.Filename,
	}

	sender := s.emailSender
	if merchant := s.resolveMerchantSender(ctx, data); merchant != nil {
		from := merchant.FromHeader()
		emailData.From = &from
		if replyTo := merchantReplyTo(merchant); replyTo != "" {
			emailData.SendAs = &replyTo
		}
		sender = s.merchantSender
	}

	sesMessageID, apiErr := sender.Send(ctx, emailData)
	if apiErr != nil {
		return nil, apiErr
	}

	return sesMessageID, nil
}

// resolveMerchantSender returns the account's own outbound identity when this message may use one, and nil to leave it on the platform address — which is the common case and never an error.
//
// Three things all have to hold. The template must be the merchant's own correspondence with their counterparty (EmailTemplate.SendsAsMerchant), because a password reset arriving from a tenant's domain reads as a spoof of the very mail it authenticates. The account must have configured a sender on a domain that is still verified, since SES rejects an unverified identity and sending under one would bounce a merchant's invoices rather than degrade them. And the region-correct sender must be wired, or there is nothing that can post the message.
//
// A lookup failure is swallowed deliberately: the platform address still delivers the mail, and losing an invoice because a branding lookup errored is the worse outcome.
func (s *notificationSvcImpl) resolveMerchantSender(ctx context.Context, data domain.EmailSendData) *domain.AccountEmailSender {
	if s.senderRepo == nil || s.merchantSender == nil {
		return nil
	}
	if data.AccountID == nil || *data.AccountID == "" || !data.TemplateID.SendsAsMerchant() {
		return nil
	}

	sender, apiErr := s.senderRepo.GetByAccount(ctx, *data.AccountID)
	if apiErr != nil || !sender.Usable() {
		return nil
	}
	return sender
}

// merchantReplyTo picks where a reply lands: the account's configured reply-to, else the sending address itself. Either beats the platform's no-reply address, which every merchant template invites the reader to write to and none of them can receive at.
func merchantReplyTo(sender *domain.AccountEmailSender) string {
	if sender.ReplyTo != nil && *sender.ReplyTo != "" {
		return *sender.ReplyTo
	}
	return sender.Address()
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

// SendEnterpriseRequest sends an enterprise upgrade request email to support.
//
// 1. If the request originates from a sandbox, skip sending and return immediately.
// 2. Render the enterprise request email template with account and requester details.
// 3. Send the rendered email to the support address.
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

	supportEmail := "support@openmrp.ai"
	_, apiErr = s.emailSender.Send(ctx, domain.EmailData{
		To:      []string{supportEmail},
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
