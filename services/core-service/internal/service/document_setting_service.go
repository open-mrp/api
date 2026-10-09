package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var documentSettingSvcTracer = tracing.GetTracer("core-service.document_setting_service")

type documentSettingSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type DocumentSettingSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *DocumentSettingSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("document setting service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("document setting service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("document setting service: tx manager is required")
	}
	return nil
}

func NewDocumentSettingSvc(config *DocumentSettingSvcConfig) domain.DocumentSettingSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &documentSettingSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *documentSettingSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *documentSettingSvcImpl) withTx(ctx context.Context, fn func(context.Context, *documentSettingSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &documentSettingSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// documentSettingOrDefault is the account's saved setting for the type, or the unsaved default when it has none.
func documentSettingOrDefault(saved *domain.DocumentSetting, accountID string, documentType constants.DocumentType) *domain.DocumentSetting {
	if saved != nil {
		return saved
	}
	return &domain.DocumentSetting{AccountID: accountID, DocumentType: documentType}
}

func (s *documentSettingSvcImpl) ListDocumentSettings(ctx context.Context) ([]*domain.DocumentSetting, *apierror.APIError) {
	ctx, span := documentSettingSvcTracer.Start(ctx, "service.document_setting.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	saved, apiErr := s.repos.NewDocumentSettingRepo().List(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	byType := make(map[constants.DocumentType]*domain.DocumentSetting, len(saved))
	for _, setting := range saved {
		byType[setting.DocumentType] = setting
	}

	documentTypes := constants.DocumentTypes()
	out := make([]*domain.DocumentSetting, len(documentTypes))
	for i, documentType := range documentTypes {
		out[i] = documentSettingOrDefault(byType[documentType], accountID, documentType)
	}
	return out, nil
}

func (s *documentSettingSvcImpl) GetDocumentSetting(ctx context.Context, documentType constants.DocumentType) (*domain.DocumentSetting, *apierror.APIError) {
	ctx, span := documentSettingSvcTracer.Start(ctx, "service.document_setting.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !documentType.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown document type.", "document_type"))
	}

	accountID := identity.Target.AccountID
	saved, apiErr := s.repos.NewDocumentSettingRepo().Get(ctx, accountID, documentType)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return documentSettingOrDefault(saved, accountID, documentType), nil
}

func (s *documentSettingSvcImpl) UpdateDocumentSetting(ctx context.Context, params domain.UpdateDocumentSettingParams) (*domain.DocumentSetting, *apierror.APIError) {
	ctx, span := documentSettingSvcTracer.Start(ctx, "service.document_setting.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !params.DocumentType.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown document type.", "document_type"))
	}

	accountID := identity.Target.AccountID
	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.DocumentSetting](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.DocumentSetting
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *documentSettingSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewDocumentSettingRepo()

			saved, apiErr := txRepo.Get(txCtx, accountID, params.DocumentType)
			if apiErr != nil {
				return apiErr
			}
			old := documentSettingOrDefault(saved, accountID, params.DocumentType)

			settingID, genErr := id.GenID(id.DocumentSettingIDPrefix, nil)
			if genErr != nil {
				return genErr
			}
			if apiErr := txRepo.Upsert(txCtx, settingID, params.After(*old)); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Get(txCtx, accountID, params.DocumentType)
			if apiErr != nil {
				return apiErr
			}
			if updated == nil {
				return apierror.NewInvariantViolationError("Document setting missing after upsert.")
			}
			result = updated

			action := constants.AuditActionUpdate
			if saved == nil {
				action = constants.AuditActionCreate
			}
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       action,
				ResourceType: constants.ObjectTypeDocumentSetting,
				ResourceID:   updated.ID,
				Changes:      audit.ComputeChanges(old, updated),
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
