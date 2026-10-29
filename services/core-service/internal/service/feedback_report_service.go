package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var feedbackReportSvcTracer = tracing.GetTracer("core-service.feedback_report_service")

type feedbackReportSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type FeedbackReportSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *FeedbackReportSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("feedback report service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("feedback report service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("feedback report service: tx manager is required")
	}
	return nil
}

func NewFeedbackReportSvc(config *FeedbackReportSvcConfig) domain.FeedbackReportSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &feedbackReportSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *feedbackReportSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *feedbackReportSvcImpl) withTx(ctx context.Context, fn func(context.Context, *feedbackReportSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &feedbackReportSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *feedbackReportSvcImpl) CreateFeedbackReport(ctx context.Context, params domain.CreateFeedbackReportParams) (*domain.FeedbackReport, *apierror.APIError) {
	ctx, span := feedbackReportSvcTracer.Start(ctx, "service.feedback_report.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsUser(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsTargetAccountSet(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.Message = strings.TrimSpace(params.Message)
	if params.Message == "" {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Message must not be blank.", "message"))
	}
	if params.Page != nil && strings.TrimSpace(*params.Page) == "" {
		params.Page = nil
	}
	params.AccountID = identity.Target.AccountID
	params.UserID = identity.Actor.ID

	feedbackReportID, apiErr := id.GenID(id.FeedbackReportIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.FeedbackReport](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.FeedbackReport
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *feedbackReportSvcImpl) *apierror.APIError {
			created, apiErr := txSvc.repos.NewFeedbackReportRepo().Create(txCtx, feedbackReportID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeFeedbackReport,
				ResourceID:   created.ID,
				Changes:      audit.ComputeChanges(nil, created),
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *feedbackReportSvcImpl) ListFeedbackReports(ctx context.Context, params domain.ListFeedbackReportsParams) (*domain.ListFeedbackReportsResult, *apierror.APIError) {
	ctx, span := feedbackReportSvcTracer.Start(ctx, "service.feedback_report.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewFeedbackReportRepo().List(ctx, params)
}

func (s *feedbackReportSvcImpl) BatchGetFeedbackReportsByIDs(ctx context.Context, ids []string) ([]*domain.FeedbackReport, *apierror.APIError) {
	ctx, span := feedbackReportSvcTracer.Start(ctx, "service.feedback_report.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return s.repos.NewFeedbackReportRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}
