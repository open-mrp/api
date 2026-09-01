package service

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
)

var utilsSvcTracer = tracing.GetTracer("core-service.service.utils")

type utilsSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	txManager             TransactionManager
	notificationPublisher domain.NotificationPublisher
	frontendURL           string
	branding              BrandingAssets
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

	// FrontendURL (optional; default: "") is the dashboard base URL used in links. It is not validated at construction.
	FrontendURL string

	// Branding (optional) resolves the merchant logo for the acknowledgement email and PDF letterhead. Omitted, both fall back to a text-only letterhead.
	Branding BrandingAssets
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
		frontendURL:           config.FrontendURL,
		branding:              config.Branding,
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
			frontendURL:           s.frontendURL,
			branding:              s.branding,
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
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
	recipients, apiErr := s.repos.NewInvoiceRepo().GetEmailRecipients(ctx, invoiceID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// No recipients still flags the invoice sent: the flag records that delivery was attempted and
	// settled, and leaving it unset offers the invoice to the resend sweep forever.
	var emailData *messaging.EmailSendData
	if len(recipients) > 0 {
		// Built by the same assembler the automatic send-on-ship uses, so a manual resend delivers an
		// identical invoice (line items, letterhead, PDF attachment).
		built, apiErr := buildInvoiceEmail(ctx, s.repos, s.branding, accountID, invoiceID)
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}
		addressed := built.addressedTo(recipients)
		emailData = &addressed
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if emailData != nil {
			if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, *emailData); apiErr != nil {
				return apiErr
			}
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
	// Built by the same assembler the automatic send-on-issue uses, so a manual resend delivers an identical acknowledgement (line items, letterhead, PDF attachment).
	emailData, apiErr := buildOrderAcknowledgementEmail(ctx, s.repos, s.branding, s.frontendURL, accountID, salesOrderID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// If no recipients, return early.
	if emailData == nil {
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil
	}

	// Publish email and mark as sent inside a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, *emailData); apiErr != nil {
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
	// Built by the same assembler the automatic send-on-issue uses, so a manual resend delivers an
	// identical submission.
	emailData, apiErr := buildPurchaseOrderSubmissionEmail(ctx, s.repos, s.branding, accountID, purchaseOrderID)
	if apiErr != nil {
		return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
	}

	// If no recipients, return early.
	if emailData == nil {
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil
	}

	// Publish email and mark as sent inside a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, *emailData); apiErr != nil {
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

const (
	// demoRequestRecipient is where marketing-site demo requests land.
	demoRequestRecipient = "dane@augno.com"
	// internalAlertRecipient is where operator-facing notices go, matching the registration alert.
	internalAlertRecipient = "dev@augno.com"
)

// RequestDemo is intentionally anonymous for public OpenAPI; POST uses idempotency keys.
func (s *utilsSvcImpl) RequestDemo(ctx context.Context, params domain.RequestDemoParams) *apierror.APIError {
	ctx, span := utilsSvcTracer.Start(ctx, "service.utils.request_demo")
	defer span.End()

	slog.InfoContext(ctx, "demo request received",
		"name", params.Name,
		"email", params.Email,
		"company", params.Company,
	)

	// The log line alone lost every demo request that arrived at this endpoint rather than the
	// dashboard's, which is the whole point of the lead form.
	emailData := messaging.EmailSendData{
		To:         []string{demoRequestRecipient},
		Subject:    "Demo Request",
		TemplateID: constants.EmailTemplateDemoRequest,
		Params: map[string]any{
			"Name":        params.Name,
			"Email":       params.Email,
			"Company":     params.Company,
			"PhoneNumber": ptrutil.Deref(params.PhoneNumber),
			"Message":     ptrutil.Deref(params.Message),
		},
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		return txSvc.notificationPublisher.PublishSendEmail(event.WithRepos(txCtx, txSvc.repos), emailData)
	})
}

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

	accountID := identity.Target.AccountID
	emailData := messaging.EmailSendData{
		To:         []string{internalAlertRecipient},
		Subject:    "Dashboard Feedback",
		TemplateID: constants.EmailTemplateDashboardFeedback,
		Params: map[string]any{
			"UserName":  ptrutil.Deref(identity.Actor.Name),
			"ActorType": string(identity.Actor.RelationType),
			// The actor's address is not on the identity, so the feedback names the account and
			// actor id and leaves the reply route to a lookup on receipt.
			"UserEmail": "",
			"ActorID":   identity.Actor.ID,
			"AccountID": accountID,
			"PageURL":   ptrutil.Deref(params.PageURL),
			"Question":  params.Question,
			"Answer":    params.Answer,
		},
		AccountID: &accountID,
		SentByID:  &identity.Actor.ID,
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *utilsSvcImpl) *apierror.APIError {
		return txSvc.notificationPublisher.PublishSendEmail(event.WithRepos(txCtx, txSvc.repos), emailData)
	})
}
