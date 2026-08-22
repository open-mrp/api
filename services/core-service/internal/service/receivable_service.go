package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var receivableSvcTracer = tracing.GetTracer("core-service.service.receivable")

type receivableSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	txManager             TransactionManager
	notificationPublisher domain.NotificationPublisher
}

// ReceivableSvcConfig holds the dependencies for the receivable service.
type ReceivableSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// NotificationPublisher (required) publishes notification messages to the outbox.
	NotificationPublisher domain.NotificationPublisher
}

func (c *ReceivableSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("receivable service: Repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("receivable service: MediatorFactory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("receivable service: TxManager is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("receivable service: NotificationPublisher is required")
	}
	return nil
}

// NewReceivableSvc creates a new receivable service with the given configuration.
func NewReceivableSvc(config *ReceivableSvcConfig) domain.ReceivableSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &receivableSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		txManager:             config.TxManager,
		notificationPublisher: config.NotificationPublisher,
	}
}

func (s *receivableSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *receivableSvcImpl) withTx(ctx context.Context, fn func(context.Context, *receivableSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &receivableSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			txManager:             s.txManager,
			notificationPublisher: s.notificationPublisher,
		}
		return fn(txCtx, txSvc)
	})
}

// ListReceivables returns a paginated list of receivable entries for the caller's account. Internal actors only. Requires invoices:read permission.
func (s *receivableSvcImpl) ListReceivables(ctx context.Context, params domain.ListReceivablesParams) (*domain.ListReceivablesResult, *apierror.APIError) {
	ctx, span := receivableSvcTracer.Start(ctx, "service.receivable.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewReceivableRepo().List(ctx, params)
}

// ListReceivablesByCustomer returns a paginated list of receivable entries for a specific customer. Internal actors only. Requires customers:read permission.
func (s *receivableSvcImpl) ListReceivablesByCustomer(ctx context.Context, params domain.ListReceivablesByCustomerParams) (*domain.ListReceivablesByCustomerResult, *apierror.APIError) {
	ctx, span := receivableSvcTracer.Start(ctx, "service.receivable.list_by_customer")
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

	params.AccountID = identity.Target.AccountID

	return s.repos.NewReceivableRepo().ListByCustomer(ctx, params)
}

// ExportReceivablesByCustomer returns all receivable entries for a specific customer without pagination. Internal actors only. Requires customers:read permission.
func (s *receivableSvcImpl) ExportReceivablesByCustomer(ctx context.Context, params domain.ListReceivablesByCustomerParams) ([]domain.ReceivableEntry, *apierror.APIError) {
	ctx, span := receivableSvcTracer.Start(ctx, "service.receivable.export_by_customer")
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

	accountID := identity.Target.AccountID

	// A customer with nothing outstanding and a customer that does not exist both produce an
	// empty ledger. Without this read the export answers for any account ID at all, and a
	// finance team cannot tell an empty statement from one addressed to the wrong company.
	if _, apiErr := s.repos.NewCustomerRepo().Get(ctx, accountID, params.CustomerAccountID, nil); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewReceivableRepo().ListAllByCustomer(ctx, accountID, params.CustomerAccountID, params.CutoffDate)
}

// EmailReceivablesForCustomer sends a receivables statement to the specified email addresses. Internal actors only. Requires customers:read permission. Uses idempotency keys. The email publish and the idempotency cache update commit together in one transaction so a retry replays the cached result instead of re-sending.
func (s *receivableSvcImpl) EmailReceivablesForCustomer(ctx context.Context, params domain.EmailReceivablesParams) *apierror.APIError {
	ctx, span := receivableSvcTracer.Start(ctx, "service.receivable.email_for_customer")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		receivableRepo := s.repos.NewReceivableRepo()

		// Fetch all receivables for the customer (no cutoff date).
		receivables, apiErr := receivableRepo.ListAllByCustomer(ctx, accountID, params.CustomerAccountID, nil)
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}

		// Fetch open credits for the customer.
		openCredits, apiErr := receivableRepo.ListOpenCreditsByCustomer(ctx, accountID, params.CustomerAccountID)
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}

		// Fetch customer name for email subject.
		customerName, apiErr := s.repos.NewAccountRepo().GetName(ctx, params.CustomerAccountID)
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}

		// Generate the Excel statement of account.
		excelBytes, err := GenerateStatementOfAccount(receivables, openCredits)
		if err != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate statement of account.")))
		}

		// Base64 encode the Excel bytes.
		encoded := base64.StdEncoding.EncodeToString(excelBytes)

		formattedDate := time.Now().Format("01-02-2006")
		attachmentFilename := fmt.Sprintf("account-statement-%s-%s.xlsx", customerName, formattedDate)
		attachmentContentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

		emailData := messaging.EmailSendData{
			To:                    params.RecipientEmails,
			Subject:               fmt.Sprintf("Statement of Account for %s", customerName),
			TemplateID:            constants.EmailTemplateStatementOfAccount,
			Params:                map[string]any{"body": fmt.Sprintf("Please find the statement of account for %s attached.", customerName)},
			AccountID:             &accountID,
			SentByID:              &identity.Actor.ID,
			AttachmentData:        &encoded,
			AttachmentFilename:    &attachmentFilename,
			AttachmentContentType: &attachmentContentType,
		}

		// Publish the email inside a transaction so the outbox write and idempotency cache update are atomic.
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivableSvcImpl) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txSvc.repos)

			if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, emailData); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}
