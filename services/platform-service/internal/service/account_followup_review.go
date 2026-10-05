package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

const (
	maxFollowupSubjectChars = 200
	maxFollowupBodyChars    = 10000
)

// GetReview returns the follow-up a review token was issued for. It never changes it: mail scanners open links before people do.
func (s *accountFollowupSvcImpl) GetReview(ctx context.Context, token string) (*domain.AccountFollowupReview, *apierror.APIError) {
	ctx, span := accountFollowupSvcTracer.Start(ctx, "service.account_followup.get_review")
	defer span.End()

	f, apiErr := s.findByReviewToken(ctx, token)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := s.checkReviewOpen(f); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.review(f), nil
}

// Approve sends the follow-up as the reviewer left it.
//
// 1. Find the follow-up by token. One already reviewed is returned unchanged, so a double submit sends once.
// 2. Validate the subject and body.
// 3. In one transaction, mark it sent with the final text and enqueue the email, with the reviewer on BCC and the context note threaded under it.
func (s *accountFollowupSvcImpl) Approve(ctx context.Context, token, subject, body string) (*domain.AccountFollowupReview, *apierror.APIError) {
	ctx, span := accountFollowupSvcTracer.Start(ctx, "service.account_followup.approve")
	defer span.End()

	f, apiErr := s.findByReviewToken(ctx, token)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if f.Status != domain.AccountFollowupStatusPendingReview {
		return s.review(f), nil
	}
	if apiErr := s.checkReviewOpen(f); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	switch {
	case subject == "" || strings.ContainsAny(subject, "\r\n") || utf8.RuneCountInString(subject) > maxFollowupSubjectChars:
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Subject must be a single line of at most 200 characters.", "subject"))
	case body == "" || utf8.RuneCountInString(body) > maxFollowupBodyChars:
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Body must be between 1 and 10,000 characters.", "body"))
	}

	email, apiErr := s.followupEmail(f, subject, body)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	now := s.now().UTC()
	if apiErr := s.tx.WithTx(ctx, func(ctx context.Context, repos domain.RepoFactory) *apierror.APIError {
		approved, apiErr := repos.NewAccountFollowupRepo().Approve(ctx, f.ID, subject, body, now)
		if apiErr != nil {
			return apiErr
		}
		// Another submit got there first; it sent the email.
		if !approved {
			return nil
		}
		if _, err := repos.NewOutboxRepo().Create(ctx, email); err != nil {
			return apierror.NewInternalError(err, "Failed to enqueue account follow-up email.")
		}
		return nil
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.reloadReview(ctx, f.ID)
}

// Skip records that the reviewer chose not to send the follow-up.
func (s *accountFollowupSvcImpl) Skip(ctx context.Context, token string) (*domain.AccountFollowupReview, *apierror.APIError) {
	ctx, span := accountFollowupSvcTracer.Start(ctx, "service.account_followup.skip")
	defer span.End()

	f, apiErr := s.findByReviewToken(ctx, token)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if f.Status != domain.AccountFollowupStatusPendingReview {
		return s.review(f), nil
	}
	if apiErr := s.checkReviewOpen(f); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if _, apiErr := s.repos.NewAccountFollowupRepo().SkipReview(ctx, f.ID, s.now().UTC()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.reloadReview(ctx, f.ID)
}

func (s *accountFollowupSvcImpl) findByReviewToken(ctx context.Context, token string) (*domain.AccountFollowup, *apierror.APIError) {
	if token == "" {
		return nil, apierror.NewResourceNotFoundError("This review link is invalid.")
	}
	hash := sha256.Sum256([]byte(token))
	f, apiErr := s.repos.NewAccountFollowupRepo().FindByReviewTokenHash(ctx, hash[:])
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apierror.NewResourceNotFoundError("This review link is invalid.")
		}
		return nil, apiErr
	}
	return f, nil
}

// checkReviewOpen rejects a link that is past its window while the follow-up still awaits review. A reviewed follow-up stays readable through its link, so the reviewer can see what was sent.
func (s *accountFollowupSvcImpl) checkReviewOpen(f *domain.AccountFollowup) *apierror.APIError {
	if f.Status == domain.AccountFollowupStatusPendingReview && (f.ReviewTokenExpiresAt == nil || !s.now().Before(*f.ReviewTokenExpiresAt)) {
		return apierror.NewExpiredTokenError("This review link has expired.")
	}
	return nil
}

func (s *accountFollowupSvcImpl) reloadReview(ctx context.Context, id string) (*domain.AccountFollowupReview, *apierror.APIError) {
	f, apiErr := s.repos.NewAccountFollowupRepo().FindByID(ctx, id)
	if apiErr != nil {
		return nil, apiErr
	}
	return s.review(f), nil
}

func (s *accountFollowupSvcImpl) review(f *domain.AccountFollowup) *domain.AccountFollowupReview {
	var timeline []string
	if f.Activity != nil {
		timeline = activityTimeline(*f.Activity)
	}
	return &domain.AccountFollowupReview{Followup: f, Timeline: timeline}
}

// followupEmail is the registrant's email: plain text from the configured sender, BCC to the reviewer, with the activity context threaded beneath it in the reviewer's inbox so a reply arrives next to it.
func (s *accountFollowupSvcImpl) followupEmail(f *domain.AccountFollowup, subject, body string) (messaging.OutboxMessageInput, *apierror.APIError) {
	from := s.fromAddress
	var timeline []string
	if f.Activity != nil {
		timeline = activityTimeline(*f.Activity)
	}
	// No AccountID, as with the review email: this is the platform's own correspondence, not the account's.
	data, err := json.Marshal(messaging.EmailSendData{
		To:         []string{f.RegistrantEmail},
		Subject:    subject,
		TemplateID: constants.EmailTemplateAccountFollowup,
		Params:     map[string]any{"Body": body},
		From:       &from,
		Bcc:        []string{s.reviewerEmail},
		ThreadNote: &messaging.EmailThreadNote{
			To:         []string{s.reviewerEmail},
			TemplateID: constants.EmailTemplateAccountFollowupContext,
			Params: map[string]any{
				"RegistrantName":  f.RegistrantName,
				"RegistrantEmail": f.RegistrantEmail,
				"AccountName":     f.AccountName,
				"AccountID":       f.AccountID,
				"RegisteredAt":    f.RegisteredAt.Format("Jan 2, 2006 15:04 MST"),
				"Engagement":      string(f.Engagement),
				"Summary":         f.InternalSummary,
				"Timeline":        timeline,
			},
		},
	})
	if err != nil {
		return messaging.OutboxMessageInput{}, apierror.NewInternalError(err, "Failed to encode account follow-up email.")
	}
	return messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdSendEmail),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdSendEmail),
		Payload:     contracts.AmqpMessage{Data: data},
	}, nil
}
