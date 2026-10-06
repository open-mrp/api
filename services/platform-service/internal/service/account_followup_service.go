package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var accountFollowupSvcTracer = tracing.GetTracer("platform-service.account_followup_service")

const (
	// defaultAccountFollowupDelay is how long after registering the follow-up is drafted: long enough for a first session to finish and maybe a second to start, soon enough that the registrant still remembers signing up.
	defaultAccountFollowupDelay = 24 * time.Hour

	// accountFollowupDraftStaleAfter is how long a follow-up may sit in drafting before it counts as abandoned. A draft is one LLM call and two writes; anything still drafting after this lost its consumer.
	accountFollowupDraftStaleAfter = 15 * time.Minute

	// accountFollowupMaxAttempts caps drafting attempts before the follow-up is marked failed for a person to look at.
	accountFollowupMaxAttempts = 4

	// accountFollowupRetryDelay is the base wait before re-drafting after a failure, multiplied by the attempt count so a gateway outage is not hammered.
	accountFollowupRetryDelay = 30 * time.Minute

	// accountFollowupDueBatchSize caps how many follow-ups one tick enqueues. Signups arrive a few a day, so this only matters after an outage.
	accountFollowupDueBatchSize = 50

	// accountFollowupActivityLimit caps the requests read for one registrant. A day of real use is a few hundred; past this the summary says it stopped reading.
	accountFollowupActivityLimit = 5000

	// accountFollowupReviewTTL is how long a review link stays valid.
	accountFollowupReviewTTL = 7 * 24 * time.Hour

	defaultAccountFollowupSignature = "Dane"

	defaultAccountFollowupFromAddress = "Dane <dane@openmrp.ai>"
)

// defaultAccountFollowupExcludedDomains are registrant domains that never get a follow-up: the team's own, and the domains tests register with. Other domains to exclude (partners, existing customers) belong in ACCOUNT_FOLLOWUP_EXCLUDED_DOMAINS, not here.
var defaultAccountFollowupExcludedDomains = []string{
	"augno.com", "openmrp.ai",
	"e2e-test.openmrp.ai", "e2e.openmrp.ai", "mail.e2e.openmrp.ai", "test.openmrp.ai",
	"example.com", "test.com",
}

// AccountFollowupSvcConfig configures the account follow-up service.
type AccountFollowupSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// Tx (required) commits a draft together with the review email announcing it.
	Tx TransactionManager

	// ReviewerEmail (required) receives every draft for approval.
	ReviewerEmail string

	// ReviewBaseURL (required) is the review page; the review token is appended as ?token=.
	ReviewBaseURL string

	// Drafter (optional; default: nil) writes the personal paragraph. Without one, Draft fails and due follow-ups wait; the scheduler should not run.
	Drafter domain.AccountFollowupDrafter

	// Delay (optional; default: 24h) is how long after registering a follow-up is drafted. Zero or negative values are treated as unset.
	Delay time.Duration

	// ExcludedDomains (optional; default: the team's and test domains) are registrant email domains that are skipped.
	ExcludedDomains []string

	// Signature (optional; default: "Dane") closes every follow-up.
	Signature string

	// FromAddress (optional; default: "Dane <dane@openmrp.ai>") sends every approved follow-up. It must be on a platform domain, or notification-service sends from noreply@ instead.
	FromAddress string

	// Now (optional; default: time.Now) is the clock, for tests.
	Now func() time.Time
}

func (c *AccountFollowupSvcConfig) WithDefaults() *AccountFollowupSvcConfig {
	if c == nil {
		c = &AccountFollowupSvcConfig{}
	}
	if c.Delay <= 0 {
		c.Delay = defaultAccountFollowupDelay
	}
	if len(c.ExcludedDomains) == 0 {
		c.ExcludedDomains = defaultAccountFollowupExcludedDomains
	}
	if c.Signature == "" {
		c.Signature = defaultAccountFollowupSignature
	}
	if c.FromAddress == "" {
		c.FromAddress = defaultAccountFollowupFromAddress
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

func (c *AccountFollowupSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("account follow-up service: repos is required")
	}
	if c.Tx == nil {
		return fmt.Errorf("account follow-up service: transaction manager is required")
	}
	if c.ReviewerEmail == "" {
		return fmt.Errorf("account follow-up service: reviewer email is required")
	}
	if _, err := url.ParseRequestURI(c.ReviewBaseURL); err != nil {
		return fmt.Errorf("account follow-up service: review base URL is invalid: %w", err)
	}
	return nil
}

type accountFollowupSvcImpl struct {
	repos         domain.RepoFactory
	tx            TransactionManager
	drafter       domain.AccountFollowupDrafter
	reviewerEmail string
	reviewBaseURL string
	delay         time.Duration
	excluded      map[string]bool
	signature     string
	fromAddress   string
	now           func() time.Time
}

func NewAccountFollowupSvc(config *AccountFollowupSvcConfig) (domain.AccountFollowupSvc, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	excluded := make(map[string]bool, len(config.ExcludedDomains))
	for _, d := range config.ExcludedDomains {
		excluded[strings.ToLower(strings.TrimSpace(d))] = true
	}
	return &accountFollowupSvcImpl{
		repos:         config.Repos,
		tx:            config.Tx,
		drafter:       config.Drafter,
		reviewerEmail: config.ReviewerEmail,
		reviewBaseURL: config.ReviewBaseURL,
		delay:         config.Delay,
		excluded:      excluded,
		signature:     config.Signature,
		fromAddress:   config.FromAddress,
		now:           config.Now,
	}, nil
}

// Schedule records a new registrant's follow-up, due Delay after they registered. Excluded registrants are recorded too and skipped when due, so the log shows every registration.
func (s *accountFollowupSvcImpl) Schedule(ctx context.Context, data messaging.AccountFollowupScheduleData) *apierror.APIError {
	ctx, span := accountFollowupSvcTracer.Start(ctx, "service.account_followup.schedule")
	defer span.End()

	followupID, apiErr := id.GenID(id.AccountFollowupIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return tracing.Trace(span, s.repos.NewAccountFollowupRepo().Create(ctx, &domain.AccountFollowup{
		ID:               followupID,
		AccountID:        data.AccountID,
		SandboxAccountID: data.SandboxAccountID,
		UserID:           data.UserID,
		AccountName:      data.AccountName,
		RegistrantName:   data.RegistrantName,
		RegistrantEmail:  data.RegistrantEmail,
		RegisteredAt:     data.RegisteredAt.UTC(),
		Status:           domain.AccountFollowupStatusScheduled,
		ScheduledFor:     data.RegisteredAt.UTC().Add(s.delay),
	}))
}

// EnqueueDue enqueues a draft command for each due follow-up. It never drafts inline: drafting calls an LLM, and the scheduler's lease should not be held for that.
func (s *accountFollowupSvcImpl) EnqueueDue(ctx context.Context) *apierror.APIError {
	ctx, span := accountFollowupSvcTracer.Start(ctx, "service.account_followup.enqueue_due")
	defer span.End()

	now := s.now().UTC()
	ids, apiErr := s.repos.NewAccountFollowupRepo().ListDueIDs(ctx, now, now.Add(-accountFollowupDraftStaleAfter), accountFollowupDueBatchSize)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	outbox := s.repos.NewOutboxRepo()
	for _, followupID := range ids {
		data, err := json.Marshal(messaging.AccountFollowupDraftData{FollowupID: followupID})
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Failed to encode account follow-up draft command."))
		}
		// A follow-up enqueued twice (the tick fires before the first command is consumed) is harmless: the draft claims the row, and the second command finds it already claimed.
		if _, err := outbox.Create(ctx, messaging.OutboxMessageInput{
			ServiceName: domain.ServiceName,
			MessageType: string(contracts.PlatformCmdDraftAccountFollowup),
			Destination: messaging.ApplicationExchange,
			RoutingKey:  string(contracts.PlatformCmdDraftAccountFollowup),
			Payload:     contracts.AmqpMessage{Data: data},
		}); err != nil {
			// One failed enqueue should not abort the batch; the follow-up stays due and is picked up next tick.
			slog.ErrorContext(ctx, "Account follow-up: failed to enqueue draft", "followup_id", followupID, "error", err)
		}
	}
	return nil
}

// Draft turns a due follow-up into a draft awaiting review.
//
// 1. Claim the follow-up; if another worker holds it or it is no longer due, stop.
// 2. Skip registrants on an excluded domain.
// 3. Summarize the registrant's activity and have the drafter write the personal paragraph. A drafter failure reschedules with backoff, then fails the follow-up after the last attempt.
// 4. Save the draft and enqueue the review email in one transaction.
func (s *accountFollowupSvcImpl) Draft(ctx context.Context, followupID string) *apierror.APIError {
	ctx, span := accountFollowupSvcTracer.Start(ctx, "service.account_followup.draft")
	defer span.End()

	repo := s.repos.NewAccountFollowupRepo()
	now := s.now().UTC()

	claimed, apiErr := repo.ClaimForDraft(ctx, followupID, now.Add(-accountFollowupDraftStaleAfter))
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !claimed {
		return nil
	}

	f, apiErr := repo.FindByID(ctx, followupID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if s.isExcluded(f.RegistrantEmail) {
		return tracing.Trace(span, repo.Skip(ctx, f.ID, domain.AccountFollowupSkipReasonExcludedDomain))
	}

	requests, apiErr := repo.ListRequests(ctx, f.AccountID, f.SandboxAccountID, f.RegisteredAt, now, accountFollowupActivityLimit+1)
	if apiErr != nil {
		return tracing.Trace(span, s.retryLater(ctx, f, apiErr.Error()))
	}
	truncated := len(requests) > accountFollowupActivityLimit
	if truncated {
		requests = requests[:accountFollowupActivityLimit]
	}
	activity := summarizeAccountActivity(requests, f.SandboxAccountID, truncated)

	if s.drafter == nil {
		return tracing.Trace(span, s.retryLater(ctx, f, "no drafter is configured"))
	}
	firstName := firstName(f.RegistrantName)
	out, err := s.drafter.Draft(ctx, domain.AccountFollowupDrafterInput{
		FirstName:   firstName,
		AccountName: f.AccountName,
		Activity:    activity,
	})
	if err != nil {
		return tracing.Trace(span, s.retryLater(ctx, f, err.Error()))
	}

	token, tokenHash, apiErr := newReviewToken()
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	draft := &domain.AccountFollowupDraft{
		Activity:             activity,
		Model:                out.Model,
		PromptVersion:        out.PromptVersion,
		Engagement:           out.Engagement,
		InternalSummary:      out.InternalSummary,
		Subject:              out.Subject,
		Body:                 composeFollowupBody(firstName, out.Paragraph, s.signature),
		ReviewTokenHash:      tokenHash,
		ReviewTokenExpiresAt: now.Add(accountFollowupReviewTTL),
		DraftedAt:            now,
	}

	reviewEmail, apiErr := s.reviewEmail(f, draft, token)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return tracing.Trace(span, s.tx.WithTx(ctx, func(ctx context.Context, repos domain.RepoFactory) *apierror.APIError {
		saved, apiErr := repos.NewAccountFollowupRepo().SaveDraft(ctx, f.ID, draft)
		if apiErr != nil {
			return apiErr
		}
		// Reclaimed as stale by another worker while the LLM call ran; that worker's draft wins.
		if !saved {
			return nil
		}
		if _, err := repos.NewOutboxRepo().Create(ctx, reviewEmail); err != nil {
			return apierror.NewInternalError(err, "Failed to enqueue account follow-up review email.")
		}
		return nil
	}))
}

// retryLater puts a follow-up whose draft failed back on the schedule with growing backoff, or fails it once attempts run out. The failure is recorded on the row rather than returned, so the command is acked instead of redelivered into the same failure.
func (s *accountFollowupSvcImpl) retryLater(ctx context.Context, f *domain.AccountFollowup, reason string) *apierror.APIError {
	slog.WarnContext(ctx, "Account follow-up: draft failed", "followup_id", f.ID, "attempt", f.Attempts, "error", reason)
	repo := s.repos.NewAccountFollowupRepo()
	if f.Attempts >= accountFollowupMaxAttempts {
		return repo.Fail(ctx, f.ID, reason)
	}
	return repo.Reschedule(ctx, f.ID, s.now().UTC().Add(time.Duration(f.Attempts)*accountFollowupRetryDelay), reason)
}

func (s *accountFollowupSvcImpl) isExcluded(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return true
	}
	return s.excluded[strings.ToLower(email[at+1:])]
}

func (s *accountFollowupSvcImpl) reviewEmail(f *domain.AccountFollowup, d *domain.AccountFollowupDraft, token string) (messaging.OutboxMessageInput, *apierror.APIError) {
	reviewURL, err := url.Parse(s.reviewBaseURL)
	if err != nil {
		return messaging.OutboxMessageInput{}, apierror.NewInternalError(err, "Invalid account follow-up review URL.")
	}
	q := reviewURL.Query()
	q.Set("token", token)
	reviewURL.RawQuery = q.Encode()

	// No AccountID: this is the platform's own mail, and attributing it to the registrant's account would put it in their email activity.
	data, err := json.Marshal(messaging.EmailSendData{
		To:         []string{s.reviewerEmail},
		Subject:    fmt.Sprintf("Follow-up ready: %s (%s)", f.RegistrantName, f.AccountName),
		TemplateID: constants.EmailTemplateAccountFollowupReview,
		Params: map[string]any{
			"RegistrantName":  f.RegistrantName,
			"RegistrantEmail": f.RegistrantEmail,
			"AccountName":     f.AccountName,
			"AccountID":       f.AccountID,
			"RegisteredAt":    f.RegisteredAt.Format("Jan 2, 2006 15:04 MST"),
			"Engagement":      string(d.Engagement),
			"Summary":         d.InternalSummary,
			"Timeline":        activityTimeline(d.Activity),
			"DraftSubject":    d.Subject,
			"DraftBody":       d.Body,
			"ReviewURL":       reviewURL.String(),
			"ExpiresAt":       d.ReviewTokenExpiresAt.Format("Jan 2, 2006"),
		},
	})
	if err != nil {
		return messaging.OutboxMessageInput{}, apierror.NewInternalError(err, "Failed to encode account follow-up review email.")
	}
	return messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdSendEmail),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdSendEmail),
		Payload:     contracts.AmqpMessage{Data: data},
	}, nil
}

// newReviewToken returns a random review token and the SHA-256 hash stored in its place, so a database read alone cannot approve a send.
func newReviewToken() (string, []byte, *apierror.APIError) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, apierror.NewInternalError(err, "Failed to generate review token.")
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}

func firstName(fullName string) string {
	if fields := strings.Fields(fullName); len(fields) > 0 {
		return fields[0]
	}
	return "there"
}

// composeFollowupBody wraps the drafter's paragraph in the fixed parts of the email: the greeting, the ask, the offer of help, and the signature. Only the paragraph varies, so a draft can't wander into promises the fixed text doesn't make.
func composeFollowupBody(firstName, paragraph, signature string) string {
	return fmt.Sprintf(`Hi %s,

%s

I'd love to hear how it's going so far: what you were hoping OpenMRP would handle for you, and whether anything was confusing or missing. Just reply to this email; it comes straight to me.

If it would help, I'm also happy to get on a quick call and help you get set up.

%s`, firstName, paragraph, signature)
}
