package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/event"
	"github.com/augno/api/services/core-service/internal/hubspotsync"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var hubspotSyncSvcTracer = tracing.GetTracer("core-service.hubspot_sync_service")

// HubspotSyncSvcConfig configures the HubSpot backfill application service.
type HubspotSyncSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
	// MediatorFactory (required) builds the idempotency mediator that gives mutations recovery-point semantics.
	MediatorFactory domain.MediatorFactory
	// TxManager (required) opens the transactions that commit a job/review mutation atomically with its outbox command and audit event.
	TxManager TransactionManager
	// Publisher (required) writes the preview/execute commands to the outbox.
	Publisher domain.HubspotSyncPublisher
}

func (c *HubspotSyncSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("hubspot sync service: Repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("hubspot sync service: MediatorFactory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("hubspot sync service: TxManager is required")
	}
	if c.Publisher == nil {
		return fmt.Errorf("hubspot sync service: Publisher is required")
	}
	return nil
}

type hubspotSyncSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	publisher       domain.HubspotSyncPublisher
}

func NewHubspotSyncSvc(config *HubspotSyncSvcConfig) domain.HubspotSyncSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &hubspotSyncSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		publisher:       config.Publisher,
	}
}

func (s *hubspotSyncSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

// StartBackfill creates a backfill job and dispatches the preview command in the same transaction, under idempotency-key recovery points so a retried request returns the original job rather than starting a second run. It also rejects a new run while one is already in flight or awaiting review.
func (s *hubspotSyncSvcImpl) StartBackfill(ctx context.Context, params domain.StartHubspotBackfillParams) (*domain.HubspotSyncJob, *apierror.APIError) {
	ctx, span := hubspotSyncSvcTracer.Start(ctx, "service.hubspot_sync.start_backfill")
	defer span.End()

	identity, apiErr := s.authorize(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.HubspotSyncJob](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var job *domain.HubspotSyncJob
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, txRepos domain.RepoFactory) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txRepos)
			syncRepo := txRepos.NewHubspotSyncRepo()

			latest, e := syncRepo.GetLatestJobForAccount(txCtx, accountID)
			if e != nil {
				return e
			}
			if latest != nil && isJobInFlight(latest.Status) {
				return apierror.NewValidationError("A HubSpot sync is already in progress or awaiting review.")
			}

			created, e := syncRepo.CreateJob(txCtx, domain.CreateHubspotSyncJobParams{
				AccountID:      accountID,
				Status:         hubspotsync.StatusPreviewing,
				DryRun:         params.DryRun,
				GoLiveCutoffAt: params.GoLiveCutoffAt,
			})
			if e != nil {
				return e
			}
			job = created

			if e := audit.NewPublisher().Publish(txCtx, txRepos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeHubspotSyncJob,
				ResourceID:   created.ID,
				Changes:      audit.ComputeChanges(nil, created),
			}); e != nil {
				return e
			}
			if e := s.publisher.PublishPreview(txCtx, messaging.HubspotSyncCommandData{JobID: created.ID, AccountID: accountID}); e != nil {
				return e
			}
			return meds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, job)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return job, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// GetCurrentJob returns the account's most recent backfill job, or a not-found error when none exists.
func (s *hubspotSyncSvcImpl) GetCurrentJob(ctx context.Context) (*domain.HubspotSyncJob, *apierror.APIError) {
	ctx, span := hubspotSyncSvcTracer.Start(ctx, "service.hubspot_sync.get_current_job")
	defer span.End()
	identity, apiErr := s.authorize(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	job, apiErr := s.repos.NewHubspotSyncRepo().GetLatestJobForAccount(ctx, identity.Target.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if job == nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("No HubSpot sync has been started for this account."))
	}
	return job, nil
}

// GetJob returns a backfill job for the caller's account.
func (s *hubspotSyncSvcImpl) GetJob(ctx context.Context, jobID string) (*domain.HubspotSyncJob, *apierror.APIError) {
	ctx, span := hubspotSyncSvcTracer.Start(ctx, "service.hubspot_sync.get_job")
	defer span.End()
	identity, apiErr := s.authorize(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	job, apiErr := s.repos.NewHubspotSyncRepo().GetJob(ctx, identity.Target.AccountID, jobID)
	return job, tracing.Trace(span, apiErr)
}

// ListReviews returns a job's company-review queue for the caller's account.
func (s *hubspotSyncSvcImpl) ListReviews(ctx context.Context, jobID string, status *string) ([]*domain.HubspotCompanyReview, *apierror.APIError) {
	ctx, span := hubspotSyncSvcTracer.Start(ctx, "service.hubspot_sync.list_reviews")
	defer span.End()
	identity, apiErr := s.authorize(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Confirm the job belongs to the account before exposing its reviews.
	if _, apiErr := s.repos.NewHubspotSyncRepo().GetJob(ctx, identity.Target.AccountID, jobID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	reviews, apiErr := s.repos.NewHubspotSyncRepo().ListReviewsForJob(ctx, jobID, status)
	return reviews, tracing.Trace(span, apiErr)
}

// ResolveReview records a human resolution for one ambiguous company review, under idempotency-key recovery points.
func (s *hubspotSyncSvcImpl) ResolveReview(ctx context.Context, params domain.ResolveHubspotReviewParams) (*domain.HubspotCompanyReview, *apierror.APIError) {
	ctx, span := hubspotSyncSvcTracer.Start(ctx, "service.hubspot_sync.resolve_review")
	defer span.End()
	identity, apiErr := s.authorize(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	// Deterministic input validation runs before idempotency, mirroring the create flows.
	resolveParams := domain.ResolveHubspotCompanyReviewParams{ID: params.ReviewID, AccountID: accountID}
	switch params.Action {
	case hubspotsync.ReviewResolutionLink:
		if params.ResolvedHubspotID == nil || *params.ResolvedHubspotID == "" {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("A HubSpot company id is required when linking.", "resolved_hubspot_id"))
		}
		resolveParams.Status = hubspotsync.ReviewStatusResolved
		resolution := hubspotsync.ReviewResolutionLink
		resolveParams.Resolution = &resolution
		resolveParams.ResolvedHubspotID = params.ResolvedHubspotID
	case hubspotsync.ReviewResolutionCreateNew:
		resolveParams.Status = hubspotsync.ReviewStatusResolved
		resolution := hubspotsync.ReviewResolutionCreateNew
		resolveParams.Resolution = &resolution
	case hubspotsync.ReviewActionSkip:
		resolveParams.Status = hubspotsync.ReviewStatusSkipped
	default:
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Action must be one of: link, create_new, skip.", "action"))
	}

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.HubspotCompanyReview](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var review *domain.HubspotCompanyReview
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, txRepos domain.RepoFactory) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txRepos)
			repo := txRepos.NewHubspotSyncRepo()

			old, e := repo.GetReview(txCtx, accountID, params.ReviewID)
			if e != nil {
				return e
			}
			if e := repo.ResolveReview(txCtx, resolveParams); e != nil {
				return e
			}
			updated, e := repo.GetReview(txCtx, accountID, params.ReviewID)
			if e != nil {
				return e
			}
			review = updated

			if e := audit.NewPublisher().Publish(txCtx, txRepos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeHubspotCompanyReview,
				ResourceID:   updated.ID,
				Changes:      audit.ComputeChanges(old, updated),
			}); e != nil {
				return e
			}
			return meds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, review)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return review, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// StartExecute dispatches the execute command for a reviewed job, under idempotency-key recovery points. The job must be awaiting review (or a prior failed run being retried) with no pending company reviews.
func (s *hubspotSyncSvcImpl) StartExecute(ctx context.Context, jobID string) (*domain.HubspotSyncJob, *apierror.APIError) {
	ctx, span := hubspotSyncSvcTracer.Start(ctx, "service.hubspot_sync.start_execute")
	defer span.End()
	identity, apiErr := s.authorize(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.HubspotSyncJob](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var job *domain.HubspotSyncJob
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, txRepos domain.RepoFactory) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txRepos)
			repo := txRepos.NewHubspotSyncRepo()

			current, e := repo.GetJob(txCtx, accountID, jobID)
			if e != nil {
				return e
			}
			if current.Status != hubspotsync.StatusReviewPending && current.Status != hubspotsync.StatusFailed {
				return apierror.NewValidationError("The sync must be awaiting review (or a failed run) before it can be executed.")
			}
			pending, e := repo.CountPendingReviews(txCtx, jobID)
			if e != nil {
				return e
			}
			if pending > 0 {
				return apierror.NewValidationError(fmt.Sprintf("Resolve all %d pending company reviews before executing the sync.", pending))
			}
			job = current

			if e := s.publisher.PublishExecute(txCtx, messaging.HubspotSyncCommandData{JobID: jobID, AccountID: accountID}); e != nil {
				return e
			}
			return meds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, job)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return job, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// authorize verifies the caller is an internal actor with the requested integrations permission and returns the request identity. HubSpot sync is an integration operation, so it is gated on the integrations domain.
func (s *hubspotSyncSvcImpl) authorize(ctx context.Context, action types.Action) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainIntegrations, action); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

// isJobInFlight reports whether a job's status blocks starting a new backfill.
func isJobInFlight(status string) bool {
	switch status {
	case hubspotsync.StatusPreviewing, hubspotsync.StatusReviewPending, hubspotsync.StatusExecuting:
		return true
	default:
		return false
	}
}
