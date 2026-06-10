package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var childAccountSvcTracer = tracing.GetTracer("core-service.child_account_service")

type childAccountSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ChildAccountSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ChildAccountSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("child account service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("child account service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("child account service: tx manager is required")
	}
	return nil
}

func NewChildAccountSvc(config *ChildAccountSvcConfig) domain.ChildAccountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &childAccountSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *childAccountSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *childAccountSvcImpl) withTx(ctx context.Context, fn func(context.Context, *childAccountSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &childAccountSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// BatchGetChildAccountsByIDs returns child account relations matching the
// input relation IDs that the caller's account is authorized to read.
// Used by the api-gateway resourcekit include resolver.
func (s *childAccountSvcImpl) BatchGetChildAccountsByIDs(ctx context.Context, relationIDs []string) ([]*domain.ChildAccount, *apierror.APIError) {
	ctx, span := childAccountSvcTracer.Start(ctx, "service.child_account.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if len(relationIDs) == 0 {
		return nil, nil
	}
	return s.repos.NewAccountRelationRepo().GetChildAccountsByRelationIDs(ctx, identity.Target.AccountID, relationIDs)
}

// ListChildAccounts returns a paginated list of child accounts for the target account (parent).
func (s *childAccountSvcImpl) ListChildAccounts(ctx context.Context, cursor *string, limit int32, query *string) (*domain.ListChildAccountsResult, *apierror.APIError) {
	ctx, span := childAccountSvcTracer.Start(ctx, "service.child_account.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAccountRelationRepo().ListChildAccounts(ctx, domain.ListChildAccountsParams{
		OwnerAccountID:  *identity.ActorAccountID(),
		ParentAccountID: identity.Target.AccountID,
		Cursor:          cursor,
		Limit:           limit,
		Query:           query,
	})
}

// AddChildAccount adds a child account relationship. The target account is the parent.
// PUT semantics: idempotent — if already set to this parent, return success.
func (s *childAccountSvcImpl) AddChildAccount(ctx context.Context, childAccountID string) (*domain.ChildAccount, *apierror.APIError) {
	ctx, span := childAccountSvcTracer.Start(ctx, "service.child_account.add")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ownerAccountID := *identity.ActorAccountID()
	parentAccountID := identity.Target.AccountID

	// Verify the actor has edit access to the child account.
	meds := s.mediators()
	if apiErr := meds.EditAccess.CheckEditAccess(ctx, ownerAccountID, childAccountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewAccountRelationRepo()

	// Resolve parent counterparty account ID to relation ID.
	parentRelationID, apiErr := repo.FindRelationByOwnerAndCounterparty(ctx, ownerAccountID, parentAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Parent account not found."))
	}

	// Resolve child counterparty account ID to relation ID.
	childRelationID, apiErr := repo.FindRelationByOwnerAndCounterparty(ctx, ownerAccountID, childAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Child account not found."))
	}

	// Check circular relationship: parent's parent cannot be the child being added.
	parentOfParentID, apiErr := repo.GetParentRelationID(ctx, parentRelationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if parentOfParentID != nil && *parentOfParentID == childRelationID {
		return nil, tracing.Trace(span, apierror.NewResourceConflictError("You cannot create a circular relationship."))
	}

	// Set parent_account_relation_id on the child (idempotent via UPDATE).
	var result *domain.ChildAccount
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *childAccountSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewAccountRelationRepo().SetParentRelation(txCtx, ownerAccountID, childRelationID, parentRelationID); apiErr != nil {
			return apiErr
		}

		created, apiErr := txSvc.repos.NewAccountRelationRepo().GetChildAccountDetail(txCtx, ownerAccountID, childAccountID)
		if apiErr != nil {
			return apiErr
		}
		result = created

		changes := audit.ComputeChanges(nil, created)

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeChildAccount,
			ResourceID:   created.RelationID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// RemoveChildAccount removes a child account relationship. Idempotent — if already cleared, return success.
func (s *childAccountSvcImpl) RemoveChildAccount(ctx context.Context, childAccountID string) *apierror.APIError {
	ctx, span := childAccountSvcTracer.Start(ctx, "service.child_account.remove")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	ownerAccountID := *identity.ActorAccountID()
	parentAccountID := identity.Target.AccountID

	// Verify the actor has edit access to the child account.
	meds := s.mediators()
	if apiErr := meds.EditAccess.CheckEditAccess(ctx, ownerAccountID, childAccountID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewAccountRelationRepo()

	// Resolve parent counterparty account ID to relation ID.
	parentRelationID, apiErr := repo.FindRelationByOwnerAndCounterparty(ctx, ownerAccountID, parentAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Parent account not found."))
	}

	// Resolve child counterparty account ID to relation ID.
	childRelationID, apiErr := repo.FindRelationByOwnerAndCounterparty(ctx, ownerAccountID, childAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Child account not found."))
	}

	// Fetch the child account detail before removal for audit diff.
	childAccount, apiErr := repo.GetChildAccountDetail(ctx, ownerAccountID, childAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Clear parent_account_relation_id only if it points to this parent (idempotent).
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *childAccountSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewAccountRelationRepo().ClearParentRelation(txCtx, ownerAccountID, childRelationID, parentRelationID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(childAccount, (*domain.ChildAccount)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeChildAccount,
			ResourceID:   childAccount.RelationID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
