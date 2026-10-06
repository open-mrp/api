package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/repository"
	servicemock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/service"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// inlineTx runs the callback against the same repos with no real transaction.
type inlineTx struct{ repos domain.RepoFactory }

func (t inlineTx) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(ctx, t.repos)
}

func (t inlineTx) WithTxSavepoint(context.Context, func(context.Context, domain.RepoFactory, db.SavepointRunner) *apierror.APIError) *apierror.APIError {
	panic("not used")
}

type followupFixture struct {
	svc     domain.AccountFollowupSvc
	repo    *repositorymock.MockAccountFollowupRepo
	drafter *servicemock.MockAccountFollowupDrafter
	outbox  *capturingOutboxRepo
	now     time.Time
}

func newFollowupFixture(t *testing.T) *followupFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	f := &followupFixture{
		repo:    repositorymock.NewMockAccountFollowupRepo(ctrl),
		drafter: servicemock.NewMockAccountFollowupDrafter(ctrl),
		outbox:  &capturingOutboxRepo{},
		now:     time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC),
	}
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAccountFollowupRepo().Return(f.repo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(f.outbox).AnyTimes()

	svc, err := NewAccountFollowupSvc(&AccountFollowupSvcConfig{
		Repos:         factory,
		Tx:            inlineTx{repos: factory},
		ReviewerEmail: "reviewer@example.com",
		ReviewBaseURL: "https://api.example.com/account-followups/review",
		Drafter:       f.drafter,
		Now:           func() time.Time { return f.now },
	})
	require.NoError(t, err)
	f.svc = svc
	return f
}

func (f *followupFixture) followup(email string, attempts int) *domain.AccountFollowup {
	sandbox := "ac_sandbox"
	return &domain.AccountFollowup{
		ID:               "acfu_1",
		AccountID:        "ac_live",
		SandboxAccountID: &sandbox,
		UserID:           "us_1",
		AccountName:      "Example Co",
		RegistrantName:   "Sam Example",
		RegistrantEmail:  email,
		RegisteredAt:     f.now.Add(-24 * time.Hour),
		Status:           domain.AccountFollowupStatusDrafting,
		Attempts:         attempts,
	}
}

func TestAccountFollowupScheduleIsDueAfterDelay(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)

	registered := time.Date(2026, 10, 3, 8, 34, 0, 0, time.UTC)
	var created *domain.AccountFollowup
	f.repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, af *domain.AccountFollowup) *apierror.APIError {
		created = af
		return nil
	})

	require.Nil(t, f.svc.Schedule(context.Background(), messaging.AccountFollowupScheduleData{
		AccountID: "ac_live", UserID: "us_1", AccountName: "Example Co",
		RegistrantName: "Sam Example", RegistrantEmail: "sam@customer-example.org", RegisteredAt: registered,
	}))
	require.Equal(t, domain.AccountFollowupStatusScheduled, created.Status)
	require.Equal(t, registered.Add(24*time.Hour), created.ScheduledFor, "due a day after registering, not a day after the message was consumed")
	require.Regexp(t, `^acfu_`, created.ID)
}

func TestAccountFollowupEnqueueDue(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)

	f.repo.EXPECT().ListDueIDs(gomock.Any(), f.now, f.now.Add(-accountFollowupDraftStaleAfter), int32(accountFollowupDueBatchSize)).Return([]string{"acfu_1", "acfu_2"}, nil)

	require.Nil(t, f.svc.EnqueueDue(context.Background()))
	require.Len(t, f.outbox.inputs, 2)
	var data messaging.AccountFollowupDraftData
	require.NoError(t, json.Unmarshal(f.outbox.inputs[1].Payload.Data, &data))
	require.Equal(t, "acfu_2", data.FollowupID)
	require.Equal(t, string(contracts.PlatformCmdDraftAccountFollowup), f.outbox.inputs[1].RoutingKey)
}

func TestAccountFollowupDraftSavesDraftAndEmailsReviewer(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)
	af := f.followup("sam@customer-example.org", 1)

	f.repo.EXPECT().ClaimForDraft(gomock.Any(), "acfu_1", f.now.Add(-accountFollowupDraftStaleAfter)).Return(true, nil)
	f.repo.EXPECT().FindByID(gomock.Any(), "acfu_1").Return(af, nil)
	f.repo.EXPECT().ListRequests(gomock.Any(), "ac_live", af.SandboxAccountID, af.RegisteredAt, f.now, int32(accountFollowupActivityLimit+1)).Return([]domain.AccountRequest{
		{TargetAccountID: "ac_live", Method: "POST", NormalizedRoute: "/v1/operations/locations", StatusCode: 201, OccurredAt: af.RegisteredAt.Add(time.Minute)},
	}, nil)
	f.drafter.EXPECT().Draft(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, in domain.AccountFollowupDrafterInput) (*domain.AccountFollowupDrafterOutput, error) {
		require.Equal(t, "Sam", in.FirstName)
		require.Equal(t, []domain.AccountActivityCount{{Resource: "operations/locations", Count: 1}}, in.Activity.Created)
		return &domain.AccountFollowupDrafterOutput{
			Model: "test-model", PromptVersion: "v1", Engagement: domain.AccountFollowupEngagementMedium,
			InternalSummary: "Set up a location.", Subject: "your openmrp setup", Paragraph: "Thanks for trying OpenMRP.",
		}, nil
	})
	var saved *domain.AccountFollowupDraft
	f.repo.EXPECT().SaveDraft(gomock.Any(), "acfu_1", gomock.Any()).DoAndReturn(func(_ context.Context, _ string, d *domain.AccountFollowupDraft) (bool, *apierror.APIError) {
		saved = d
		return true, nil
	})

	require.Nil(t, f.svc.Draft(context.Background(), "acfu_1"))

	require.Equal(t, "your openmrp setup", saved.Subject)
	require.Contains(t, saved.Body, "Hi Sam,\n\nThanks for trying OpenMRP.\n\n")
	require.True(t, saved.ReviewTokenExpiresAt.Equal(f.now.Add(7*24*time.Hour)))

	require.Len(t, f.outbox.inputs, 1)
	var email messaging.EmailSendData
	require.NoError(t, json.Unmarshal(f.outbox.inputs[0].Payload.Data, &email))
	require.Equal(t, []string{"reviewer@example.com"}, email.To)
	require.Equal(t, constants.EmailTemplateAccountFollowupReview, email.TemplateID)
	require.Nil(t, email.AccountID, "review mail must not land in the registrant's email activity")

	// The link carries the token whose hash was saved; the token itself is never stored.
	reviewURL, err := url.Parse(email.Params["ReviewURL"].(string))
	require.NoError(t, err)
	require.Equal(t, "/account-followups/review", reviewURL.Path)
	hash := sha256.Sum256([]byte(reviewURL.Query().Get("token")))
	require.Equal(t, hash[:], saved.ReviewTokenHash)
}

func TestAccountFollowupDraftSkipsExcludedDomain(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)

	f.repo.EXPECT().ClaimForDraft(gomock.Any(), "acfu_1", gomock.Any()).Return(true, nil)
	f.repo.EXPECT().FindByID(gomock.Any(), "acfu_1").Return(f.followup("someone@E2E.openmrp.ai", 1), nil)
	f.repo.EXPECT().Skip(gomock.Any(), "acfu_1", domain.AccountFollowupSkipReasonExcludedDomain).Return(nil)

	require.Nil(t, f.svc.Draft(context.Background(), "acfu_1"))
	require.Empty(t, f.outbox.inputs)
}

func TestAccountFollowupDraftNotClaimedIsNoop(t *testing.T) {
	t.Parallel()
	f := newFollowupFixture(t)

	f.repo.EXPECT().ClaimForDraft(gomock.Any(), "acfu_1", gomock.Any()).Return(false, nil)

	require.Nil(t, f.svc.Draft(context.Background(), "acfu_1"))
}

func TestAccountFollowupDraftFailureBacksOffThenFails(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		attempts int
		expect   func(f *followupFixture)
	}{
		{"reschedules with backoff", 2, func(f *followupFixture) {
			f.repo.EXPECT().Reschedule(gomock.Any(), "acfu_1", f.now.Add(2*accountFollowupRetryDelay), "gateway down").Return(nil)
		}},
		{"fails after the last attempt", accountFollowupMaxAttempts, func(f *followupFixture) {
			f.repo.EXPECT().Fail(gomock.Any(), "acfu_1", "gateway down").Return(nil)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFollowupFixture(t)

			f.repo.EXPECT().ClaimForDraft(gomock.Any(), "acfu_1", gomock.Any()).Return(true, nil)
			f.repo.EXPECT().FindByID(gomock.Any(), "acfu_1").Return(f.followup("sam@customer-example.org", tc.attempts), nil)
			f.repo.EXPECT().ListRequests(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			f.drafter.EXPECT().Draft(gomock.Any(), gomock.Any()).Return(nil, errors.New("gateway down"))
			tc.expect(f)

			require.Nil(t, f.svc.Draft(context.Background(), "acfu_1"), "the failure is recorded on the row, so the command is acked rather than redelivered")
			require.Empty(t, f.outbox.inputs)
		})
	}
}

func TestComposeFollowupBody(t *testing.T) {
	t.Parallel()
	body := composeFollowupBody("Sam", "Thanks for trying OpenMRP.", "Dane")
	require.Equal(t, `Hi Sam,

Thanks for trying OpenMRP.

I'd love to hear how it's going so far: what you were hoping OpenMRP would handle for you, and whether anything was confusing or missing. Just reply to this email; it comes straight to me.

If it would help, I'm also happy to get on a quick call and help you get set up.

Dane`, body)
}
