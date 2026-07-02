package service

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/event"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var utilsSvcTracer = tracing.GetTracer("core-service.service.utils")

type utilsSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	txManager             TransactionManager
	notificationPublisher domain.NotificationPublisher
}

// UtilsSvcConfig holds the dependencies for the utils service.
type UtilsSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// NotificationPublisher (required) publishes notification messages to the outbox.
	NotificationPublisher domain.NotificationPublisher
}

func (c *UtilsSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("utils service: Repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("utils service: MediatorFactory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("utils service: TxManager is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("utils service: NotificationPublisher is required")
	}
	return nil
}

// NewUtilsSvc creates a new utils service with the given configuration.
func NewUtilsSvc(config *UtilsSvcConfig) domain.UtilsSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &utilsSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		txManager:             config.TxManager,
		notificationPublisher: config.NotificationPublisher,
	}
}

func (s *utilsSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *utilsSvcImpl) withTx(ctx context.Context, fn func(context.Context, *utilsSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &utilsSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			txManager:             s.txManager,
			notificationPublisher: s.notificationPublisher,
		}
		return fn(txCtx, txSvc)
	})
}

// CheckDuplicate checks whether a record number already exists.
// Allows both internal and customer actors (CheckIsAssignedActor).
// For internal actors, verifies appropriate read permissions per type.
// PUT endpoint — idempotent by design, no idempotency keys needed.
func (s *utilsSvcImpl) CheckDuplicate(ctx context.Context, params domain.CheckDuplicateParams) (*domain.CheckDuplicateResult, *apierror.APIError) {
	ctx, span := utilsSvcTracer.Start(ctx, "service.utils.check_duplicate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.IsInternalActor() {
		if identity.IsTargetCustomerAccount() {
			if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		} else if identity.IsTargetSupplierAccount() {
			if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		} else {
			switch params.Type {
			case domain.DuplicateCheckTypeInvoiceNumber:
				if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			case domain.DuplicateCheckTypeOrderNumber, domain.DuplicateCheckTypeCustomerPO:
				if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			default:
				return nil, tracing.Trace(span, apierror.NewValidationError("Unsupported duplicate check type."))
			}
		}
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

	accountID := identity.Target.AccountID
	recordNumber := strings.TrimSpace(params.RecordNumber)

	var isDuplicate bool
	var message *string

	switch params.Type {
	case domain.DuplicateCheckTypeInvoiceNumber:
		dup, apiErr := s.repos.NewInvoiceRepo().IsDuplicateNumber(ctx, accountID, recordNumber)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		isDuplicate = dup
		if isDuplicate {
			msg := fmt.Sprintf("This invoice number %s already exists", recordNumber)
			message = &msg
		}

	case domain.DuplicateCheckTypeOrderNumber:
		dup, apiErr := s.repos.NewSalesOrderRepo().IsDuplicateOrderNumber(ctx, accountID, recordNumber, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		isDuplicate = dup
		if isDuplicate {
			msg := fmt.Sprintf("This sales order number %s already exists", recordNumber)
			message = &msg
		}

	case domain.DuplicateCheckTypeCustomerPO:
		if params.CustomerID == nil {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Customer ID is required for customer PO number duplicate check.", "customer_id"))
		}
		dup, apiErr := s.repos.NewSalesOrderRepo().IsDuplicateCustomerPO(ctx, accountID, *params.CustomerID, recordNumber, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		isDuplicate = dup
		if isDuplicate {
			msg := fmt.Sprintf("This customer PO number %s already exists", recordNumber)
			message = &msg
		}

	default:
		return nil, tracing.Trace(span, apierror.NewValidationError("Unsupported duplicate check type."))
	}

	return &domain.CheckDuplicateResult{
		IsDuplicate: isDuplicate,
		Message:     message,
	}, nil
}

// EmailRecord emails a record (invoice, sales order, or purchase order) to the configured recipients. POST endpoint — uses idempotency keys.
func (s *utilsSvcImpl) EmailRecord(ctx context.Context, params domain.EmailRecordParams) *apierror.APIError {
	ctx, span := utilsSvcTracer.Start(ctx, "service.utils.email_record")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch params.Type {
	case domain.EmailRecordTypeInvoice:
		if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	case domain.EmailRecordTypeSalesOrder:
		if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	case domain.EmailRecordTypePurchaseOrder:
		if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionRead); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	default:
		return tracing.Trace(span, apierror.NewValidationError("Unsupported email record type."))
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
		return s.emailRecordStarted(ctx, span, params, accountID, meds, idempotencyKey)

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *utilsSvcImpl) emailRecordStarted(ctx context.Context, span trace.Span, params domain.EmailRecordParams, accountID string, meds domain.Mediators, idempotencyKey *domain.IdempotencyKey) *apierror.APIError {
	switch params.Type {
	case domain.EmailRecordTypeInvoice:
		return s.emailInvoice(ctx, span, params.ID, accountID, meds, idempotencyKey)
	case domain.EmailRecordTypeSalesOrder:
		return s.emailSalesOrder(ctx, span, params.ID, accountID, meds, idempotencyKey)
	case domain.EmailRecordTypePurchaseOrder:
		return s.emailPurchaseOrder(ctx, span, params.ID, accountID, meds, idempotencyKey)
	default:
		return tracing.Trace(span, apierror.NewValidationError("Unsupported email record type."))
	}
}

func (s *utilsSvcImpl) emailInvoice(ctx context.Context, span trace.Span, invoiceID, accountID string, meds domain.Mediators, idempotencyKey *domain.IdempotencyKey) *apierror.APIError {
	invoiceRepo := s.repos.NewInvoiceRepo()

	// Fetch the invoice.
	invoice, apiErr := invoiceRepo.Get(ctx, domain.GetInvoiceParams{
		AccountID: accountID,
		InvoiceID: invoiceID,
	})
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// Fetch recipient emails.
	recipients, apiErr := invoiceRepo.GetEmailRecipients(ctx, invoiceID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// If no recipients, just mark as sent and return.
	if len(recipients) == 0 {
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
			if apiErr := txSvc.repos.NewInvoiceRepo().MarkEmailSent(txCtx, accountID, invoiceID); apiErr != nil {
				return apiErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil
	}

	// Fetch account name for email.
	accountName, apiErr := s.repos.NewAccountRepo().GetName(ctx, accountID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	subject := fmt.Sprintf("Invoice %s", invoice.Number)

	emailData := messaging.EmailSendData{
		To:         recipients,
		Subject:    subject,
		TemplateID: constants.EmailTemplateInvoice,
		Params: map[string]any{
			"invoice_number": invoice.Number,
			"account_name":   accountName,
		},
		AccountID: &accountID,
	}

	// Publish email and mark as sent inside a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, emailData); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewInvoiceRepo().MarkEmailSent(txCtx, accountID, invoiceID); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
	})

	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return nil
}

func (s *utilsSvcImpl) emailSalesOrder(ctx context.Context, span trace.Span, salesOrderID, accountID string, meds domain.Mediators, idempotencyKey *domain.IdempotencyKey) *apierror.APIError {
	salesOrderRepo := s.repos.NewSalesOrderRepo()

	// Fetch the sales order.
	order, apiErr := salesOrderRepo.Get(ctx, accountID, salesOrderID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// Fetch recipient emails.
	recipients, apiErr := salesOrderRepo.GetAcknowledgementRecipients(ctx, salesOrderID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// If no recipients, return early.
	if len(recipients) == 0 {
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil
	}

	// Fetch account name for email.
	accountName, apiErr := s.repos.NewAccountRepo().GetName(ctx, accountID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	subject := fmt.Sprintf("Sales Order %s", order.Number)

	emailData := messaging.EmailSendData{
		To:         recipients,
		Subject:    subject,
		TemplateID: constants.EmailTemplateOrderAcknowledgement,
		Params: map[string]any{
			"order_number": order.Number,
			"account_name": accountName,
		},
		AccountID: &accountID,
	}

	// Publish email and mark as sent inside a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, emailData); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewSalesOrderRepo().MarkAcknowledgementSent(txCtx, accountID, salesOrderID); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
	})

	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return nil
}

func (s *utilsSvcImpl) emailPurchaseOrder(ctx context.Context, span trace.Span, purchaseOrderID, accountID string, meds domain.Mediators, idempotencyKey *domain.IdempotencyKey) *apierror.APIError {
	purchaseOrderRepo := s.repos.NewPurchaseOrderRepo()

	// Fetch the purchase order.
	po, apiErr := purchaseOrderRepo.Get(ctx, accountID, purchaseOrderID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// Fetch recipient emails.
	recipients, apiErr := purchaseOrderRepo.GetSubmissionRecipients(ctx, purchaseOrderID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// If no recipients, return early.
	if len(recipients) == 0 {
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil
	}

	// Fetch account name for email.
	accountName, apiErr := s.repos.NewAccountRepo().GetName(ctx, accountID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	subject := fmt.Sprintf("Purchase Order %s", po.Number)

	emailData := messaging.EmailSendData{
		To:         recipients,
		Subject:    subject,
		TemplateID: constants.EmailTemplatePurchaseOrderSubmission,
		Params: map[string]any{
			"order_number": po.Number,
			"account_name": accountName,
		},
		AccountID: &accountID,
	}

	// Publish email and mark as sent inside a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, emailData); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewPurchaseOrderRepo().MarkSubmissionSent(txCtx, accountID, purchaseOrderID); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
	})

	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return nil
}

// RequestDemo is intentionally anonymous for public OpenAPI; POST uses idempotency keys.
func (s *utilsSvcImpl) RequestDemo(ctx context.Context, params domain.RequestDemoParams) *apierror.APIError {
	ctx, span := utilsSvcTracer.Start(ctx, "service.utils.request_demo")
	defer span.End()

	slog.InfoContext(ctx, "demo request received",
		"name", params.Name,
		"email", params.Email,
		"company", params.Company,
	)

	return nil
}

// SubmitFeedback logs user feedback. Requires an authenticated actor.
// Email sending will be wired later via the notification service.
// POST endpoint — uses idempotency keys.
func (s *utilsSvcImpl) SubmitFeedback(ctx context.Context, params domain.SubmitFeedbackParams) *apierror.APIError {
	ctx, span := utilsSvcTracer.Start(ctx, "service.utils.submit_feedback")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	slog.InfoContext(ctx, "feedback submitted",
		"question", params.Question,
		"answer", params.Answer,
		"page_url", params.PageURL,
	)

	return nil
}
