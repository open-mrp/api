package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/metadata"
	"github.com/open-mrp/api/shared/tracing"
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

	if apiErr := checkInvoiceAccess(identity, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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

	if apiErr := checkInvoiceAccess(identity, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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
			if apiErr := attachAllocationTransactions(ctx, s.repos.NewTransactionRepo(), params.AccountID, allocations); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			invoice.Allocations = allocations
		}
	}

	return invoice, nil
}

// attachAllocationTransactions reads the transactions the allocations draw on in one query and sets
// each on its allocation. They are part of the invoice's ledger, so a caller allowed to read the
// invoice sees them without also holding transactions:read, as the dashboard always showed them.
func attachAllocationTransactions(ctx context.Context, repo domain.TransactionRepo, accountID string, allocations []*domain.InvoiceAllocation) *apierror.APIError {
	ids := make([]string, 0, len(allocations))
	seen := make(map[string]bool, len(allocations))
	for _, a := range allocations {
		if a.TransactionID != "" && !seen[a.TransactionID] {
			seen[a.TransactionID] = true
			ids = append(ids, a.TransactionID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	transactions, apiErr := repo.GetByIDs(ctx, accountID, ids)
	if apiErr != nil {
		return apiErr
	}
	byID := make(map[string]*domain.Transaction, len(transactions))
	for _, t := range transactions {
		byID[t.ID] = t
	}
	for _, a := range allocations {
		a.Transaction = byID[a.TransactionID]
	}
	return nil
}

func (s *invoiceSvcImpl) UpdateInvoice(ctx context.Context, params domain.UpdateInvoiceParams) (*domain.Invoice, *apierror.APIError) {
	ctx, span := invoiceSvcTracer.Start(ctx, "service.invoice.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := checkInvoiceAccess(identity, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	// Recalculating the invoice's payments later can overturn a paid-in-full flag set here; the person who
	// set it is told. An API key or agent has no one to tell.
	params.PaidInFullMarkedByID = nil
	if params.IsPaidInFull != nil && identity.HasUserActor() {
		params.PaidInFullMarkedByID = &identity.Actor.ID
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Invoice](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Invoice
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
			if apiErr := metadata.CheckLimit(updated.Metadata, "metadata"); apiErr != nil {
				return apiErr
			}

			// Expand the same relations a read would, so a PATCH answers ?include= like a GET does.
			for _, include := range params.Includes {
				switch include {
				case "lines":
					lines, apiErr := txRepo.GetLines(txCtx, params.InvoiceID)
					if apiErr != nil {
						return apiErr
					}
					updated.Lines = lines
				case "allocations":
					allocations, apiErr := txRepo.GetAllocations(txCtx, params.InvoiceID)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := attachAllocationTransactions(txCtx, txSvc.repos.NewTransactionRepo(), params.AccountID, allocations); apiErr != nil {
						return apiErr
					}
					updated.Allocations = allocations
				}
			}
			result = updated

			// Names the fields explicitly so only the updatable ones are diffed.
			changes := audit.ComputeChanges(old, updated, "Note", "HasBeenSent", "IsEdiSent", "IsPaidInFull", "Metadata")

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeInvoice,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   updated.OrderID,
				Changes:          changes,
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

	if apiErr := checkInvoiceAccess(identity, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewInvoiceRepo()

	result, apiErr := repo.ListByCustomer(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The settle flow works each invoice's balance out from these, but they cost a query of their
	// own, so they are loaded only when the caller asked to expand them.
	if slices.Contains(params.Includes, "allocations") && len(result.Invoices) > 0 {
		invoiceIDs := make([]string, len(result.Invoices))
		for i, inv := range result.Invoices {
			invoiceIDs[i] = inv.ID
		}

		byInvoice, apiErr := repo.GetAllocationsForInvoices(ctx, invoiceIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		var all []*domain.InvoiceAllocation
		for _, inv := range result.Invoices {
			inv.Allocations = byInvoice[inv.ID]
			all = append(all, inv.Allocations...)
		}
		if apiErr := attachAllocationTransactions(ctx, s.repos.NewTransactionRepo(), params.AccountID, all); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return result, nil
}

// checkInvoiceAccess admits only the seller's own users, an invoice being its ledger, and for a read anyone loading an invoice that a request they were allowed to make includes.
func checkInvoiceAccess(identity *types.Identity, action types.Action) *apierror.APIError {
	checkActor := identity.CheckIsInternalActor
	if action == types.ActionRead {
		checkActor = identity.CheckIsInternalActorForRead
	}
	if apiErr := checkActor(); apiErr != nil {
		return apiErr
	}
	return identity.CheckHasPermission(types.PermissionDomainInvoices, action)
}

// Rejects an invoice that would exceed the account plan's per-billing-period invoice cap. Sandboxes,
// accounts on no plan, and plans with no cap are exempt.
func enforceInvoicesPerPeriodLimit(ctx context.Context, repos domain.RepoFactory, accountID string) *apierror.APIError {
	max, start, apiErr := resolveAccountPlanLimit(ctx, repos, accountID, constants.AccountPlanLimitInvoicesMaximum)
	if apiErr != nil || max == nil {
		return apiErr
	}

	count, apiErr := repos.NewInvoiceRepo().CountSince(ctx, accountID, start)
	if apiErr != nil {
		return apiErr
	}
	if count >= int64(*max) {
		return apierror.NewValidationError(fmt.Sprintf("Your plan allows a maximum of %d invoices per billing period.", *max))
	}
	return nil
}
