package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func (f *followupFixture) pendingReview(token string) (*domain.AccountFollowup, []byte) {
	af := f.followup("sam@customer-example.org", 1)
	af.Status = domain.AccountFollowupStatusPendingReview
	expires := f.now.Add(time.Hour)
	af.ReviewTokenExpiresAt = &expires
	af.Engagement = domain.AccountFollowupEngagementMedium
	af.InternalSummary = "Set up locations."
	af.DraftSubject = "draft subject"
	af.DraftBody = "draft body"
	af.Activity = &domain.AccountActivity{TotalRequests: 1, Sessions: []domain.AccountActivitySession{{Start: af.RegisteredAt, Requests: 1}}}
	hash := sha256.Sum256([]byte(token))
	return af, hash[:]
}

func TestAccountFollowupApproveSendsReviewedText(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)
	af, hash := f.pendingReview("tok")

	sent := *af
	sent.Status = domain.AccountFollowupStatusSent
	f.repo.EXPECT().FindByReviewTokenHash(gomock.Any(), hash).Return(af, nil)
	f.repo.EXPECT().Approve(gomock.Any(), "acfu_1", "edited subject", "Hi Sam,\n\nedited body", f.now).Return(true, nil)
	f.repo.EXPECT().FindByID(gomock.Any(), "acfu_1").Return(&sent, nil)

	review, apiErr := f.svc.Approve(context.Background(), "tok", "  edited subject ", "Hi Sam,\r\n\r\nedited body\n")
	require.Nil(t, apiErr)
	require.Equal(t, domain.AccountFollowupStatusSent, review.Followup.Status)

	require.Len(t, f.outbox.inputs, 1)
	var email messaging.EmailSendData
	require.NoError(t, json.Unmarshal(f.outbox.inputs[0].Payload.Data, &email))
	require.Equal(t, []string{"sam@customer-example.org"}, email.To)
	require.Equal(t, "edited subject", email.Subject)
	require.Equal(t, constants.EmailTemplateAccountFollowup, email.TemplateID)
	require.Equal(t, "Hi Sam,\n\nedited body", email.Params["Body"])
	require.Equal(t, "OpenMRP <dev@openmrp.ai>", *email.From)
	require.Equal(t, []string{"reviewer@example.com"}, email.Bcc)
	require.Nil(t, email.AccountID)
	require.Equal(t, []string{"reviewer@example.com"}, email.ThreadNote.To)
	require.Equal(t, constants.EmailTemplateAccountFollowupContext, email.ThreadNote.TemplateID)
	require.Equal(t, "Set up locations.", email.ThreadNote.Params["Summary"])
}

func TestAccountFollowupApproveTwiceSendsOnce(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)
	af, hash := f.pendingReview("tok")
	af.Status = domain.AccountFollowupStatusSent

	f.repo.EXPECT().FindByReviewTokenHash(gomock.Any(), hash).Return(af, nil)

	review, apiErr := f.svc.Approve(context.Background(), "tok", "s", "b")
	require.Nil(t, apiErr)
	require.Equal(t, domain.AccountFollowupStatusSent, review.Followup.Status)
	require.Empty(t, f.outbox.inputs)
}

func TestAccountFollowupApproveLostRaceSendsNothing(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)
	af, hash := f.pendingReview("tok")

	f.repo.EXPECT().FindByReviewTokenHash(gomock.Any(), hash).Return(af, nil)
	f.repo.EXPECT().Approve(gomock.Any(), "acfu_1", "s", "b", f.now).Return(false, nil)
	f.repo.EXPECT().FindByID(gomock.Any(), "acfu_1").Return(af, nil)

	_, apiErr := f.svc.Approve(context.Background(), "tok", "s", "b")
	require.Nil(t, apiErr)
	require.Empty(t, f.outbox.inputs, "the submit that won the update sent the email")
}

func TestAccountFollowupApproveRejectsBadInput(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct{ subject, body string }{
		"empty subject":     {"", "b"},
		"multiline subject": {"a\nBcc: x@example.com", "b"},
		"empty body":        {"s", "   "},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newFollowupFixture(t)
			af, hash := f.pendingReview("tok")
			f.repo.EXPECT().FindByReviewTokenHash(gomock.Any(), hash).Return(af, nil)

			_, apiErr := f.svc.Approve(context.Background(), "tok", tc.subject, tc.body)
			require.NotNil(t, apiErr)
			require.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
			require.Empty(t, f.outbox.inputs)
		})
	}
}

func TestAccountFollowupReviewExpired(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)
	af, hash := f.pendingReview("tok")
	expired := f.now.Add(-time.Minute)
	af.ReviewTokenExpiresAt = &expired

	f.repo.EXPECT().FindByReviewTokenHash(gomock.Any(), hash).Return(af, nil).Times(2)

	_, apiErr := f.svc.GetReview(context.Background(), "tok")
	require.Equal(t, apierror.ErrorCodeExpiredToken, apiErr.Code)
	_, apiErr = f.svc.Approve(context.Background(), "tok", "s", "b")
	require.Equal(t, apierror.ErrorCodeExpiredToken, apiErr.Code)
	require.Empty(t, f.outbox.inputs)
}

func TestAccountFollowupReviewUnknownToken(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)

	f.repo.EXPECT().FindByReviewTokenHash(gomock.Any(), gomock.Any()).Return(nil, apierror.NewResourceNotFoundError("Account follow-up not found."))

	_, apiErr := f.svc.GetReview(context.Background(), "nope")
	require.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	require.Equal(t, "This review link is invalid.", apiErr.PublicMessage)
}

func TestAccountFollowupSkip(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)
	af, hash := f.pendingReview("tok")
	skipped := *af
	skipped.Status = domain.AccountFollowupStatusSkipped

	f.repo.EXPECT().FindByReviewTokenHash(gomock.Any(), hash).Return(af, nil)
	f.repo.EXPECT().SkipReview(gomock.Any(), "acfu_1", f.now).Return(true, nil)
	f.repo.EXPECT().FindByID(gomock.Any(), "acfu_1").Return(&skipped, nil)

	review, apiErr := f.svc.Skip(context.Background(), "tok")
	require.Nil(t, apiErr)
	require.Equal(t, domain.AccountFollowupStatusSkipped, review.Followup.Status)
	require.Empty(t, f.outbox.inputs)
}
