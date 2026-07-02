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
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var invoiceSvcTracer = tracing.GetTracer("core-service.invoice_service")

type invoiceSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type InvoiceSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *InvoiceSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("invoice service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("invoice service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("invoice service: tx manager is required")
	}
	return nil
}

func NewInvoiceSvc(config *InvoiceSvcConfig) domain.InvoiceSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &invoiceSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *invoiceSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *invoiceSvcImpl) withTx(ctx context.Context, fn func(context.Context, *invoiceSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &invoiceSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *invoiceSvcImpl) ListInvoices(ctx context.Context, params domain.ListInvoicesParams) (*domain.ListInvoicesResult, *apierror.APIError) {
	ctx, span := invoiceSvcTracer.Start(ctx, "service.invoice.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkInvoiceReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewInvoiceRepo()
	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Expand lines per invoice only when requested (so the list can serve the lines.item array filter).
	for _, include := range params.Includes {
		if include == "lines" {
			for _, inv := range result.Invoices {
				lines, apiErr := repo.GetLines(ctx, inv.ID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				inv.Lines = lines
			}
			break
		}
	}

	return result, nil
}

func (s *invoiceSvcImpl) GetInvoice(ctx context.Context, params domain.GetInvoiceParams) (*domain.Invoice, *apierror.APIError) {
	ctx, span := invoiceSvcTracer.Start(ctx, "service.invoice.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkInvoiceReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewInvoiceRepo()

	invoice, apiErr := repo.Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Conditionally fetch lines and allocations based on includes
	for _, include := range params.Includes {
		switch include {
		case "lines":
			lines, apiErr := repo.GetLines(ctx, params.InvoiceID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			invoice.Lines = lines
		case "allocations":
			allocations, apiErr := repo.GetAllocations(ctx, params.InvoiceID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			invoice.Allocations = allocations
		}
	}

	return invoice, nil
}

func (s *invoiceSvcImpl) UpdateInvoice(ctx context.Context, params domain.UpdateInvoiceParams) (*domain.InvoiceSummary, *apierror.APIError) {
	ctx, span := invoiceSvcTracer.Start(ctx, "service.invoice.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkInvoiceWritePermission(identity, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.InvoiceSummary](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.InvoiceSummary
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *invoiceSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewInvoiceRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetInvoiceParams{
				AccountID: params.AccountID,
				InvoiceID: params.InvoiceID,
			})
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			// Use explicit field names because old (*Invoice) and updated (*InvoiceSummary) are different types.
			// Only the updatable fields are compared.
			changes := audit.ComputeChanges(old, updated, "Note", "HasBeenSent", "IsEdiSent", "IsPaidInFull")

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeInvoice,
				ResourceID:   updated.ID,
				Changes:      changes,
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

func (s *invoiceSvcImpl) ListCustomerInvoices(ctx context.Context, params domain.ListCustomerInvoicesParams) (*domain.ListCustomerInvoicesResult, *apierror.APIError) {
	ctx, span := invoiceSvcTracer.Start(ctx, "service.invoice.list_customer")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkInvoiceReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID
	params.IncludeChildAccounts = true

	repo := s.repos.NewInvoiceRepo()

	result, apiErr := repo.ListByCustomer(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Fetch allocations for each invoice to match Dashboard parity
	for _, inv := range result.Invoices {
		allocations, apiErr := repo.GetAllocations(ctx, inv.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		inv.Allocations = allocations
	}

	return result, nil
}

// checkInvoiceReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need invoices:read for their own account, or customers:read / suppliers:read for external accounts.
func checkInvoiceReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead)
}

// checkInvoiceWritePermission checks the appropriate write permission based on the identity context.
// Internal actors need invoices:{action} for their own account, or customers:update / suppliers:update for external accounts.
func checkInvoiceWritePermission(identity *types.Identity, action types.Action) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
	}
	return identity.CheckHasPermission(types.PermissionDomainInvoices, action)
}
