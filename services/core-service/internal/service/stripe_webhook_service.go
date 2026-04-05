package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var stripeWebhookSvcTracer = tracing.GetTracer("core-service.service.stripe_webhook")

type stripeWebhookSvcImpl struct {
	repos                 domain.RepoFactory
	txManager             TransactionManager
	checkoutClientFactory domain.StripeCheckoutClientFactory
	encryptionKey         []byte
}

type StripeWebhookSvcConfig struct {
	Repos                 domain.RepoFactory
	TxManager             TransactionManager
	CheckoutClientFactory domain.StripeCheckoutClientFactory
	EncryptionKey         []byte
}

func (c *StripeWebhookSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("stripe webhook service: Repos is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("stripe webhook service: TxManager is required")
	}
	if c.CheckoutClientFactory == nil {
		return fmt.Errorf("stripe webhook service: CheckoutClientFactory is required")
	}
	if len(c.EncryptionKey) == 0 {
		return fmt.Errorf("stripe webhook service: EncryptionKey is required")
	}
	return nil
}

func NewStripeWebhookSvc(config *StripeWebhookSvcConfig) domain.StripeWebhookSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &stripeWebhookSvcImpl{
		repos:                 config.Repos,
		txManager:             config.TxManager,
		checkoutClientFactory: config.CheckoutClientFactory,
		encryptionKey:         config.EncryptionKey,
	}
}

func (s *stripeWebhookSvcImpl) withTx(ctx context.Context, fn func(context.Context, *stripeWebhookSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &stripeWebhookSvcImpl{
			repos:     f,
			txManager: s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// HandleAccountStripeWebhook processes a Stripe webhook event for a specific account.
// Always returns nil to ensure Stripe receives a 200 response. Errors are logged.
func (s *stripeWebhookSvcImpl) HandleAccountStripeWebhook(ctx context.Context, params domain.HandleStripeWebhookParams) *apierror.APIError {
	ctx, span := stripeWebhookSvcTracer.Start(ctx, "service.stripe_webhook.handle")
	defer span.End()

	// Fetch and decrypt the account's Stripe webhook secret.
	integrationRepo := s.repos.NewAccountIntegrationRepo()
	encryptedCreds, isActive, apiErr := integrationRepo.GetEncryptedCredentials(ctx, params.AccountID, constants.IntegrationCodeStripe)
	if apiErr != nil {
		slog.Error("stripe webhook: failed to fetch integration credentials", "account_id", params.AccountID, "error", apiErr.PublicMessage)
		return nil
	}
	if !isActive {
		slog.Warn("stripe webhook: integration not active", "account_id", params.AccountID)
		return nil
	}

	decrypted, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, nil)
	if err != nil {
		slog.Error("stripe webhook: failed to decrypt credentials", "account_id", params.AccountID, "error", err.Error())
		return nil
	}
	var stripeCreds domain.StripeCredentials
	if err := json.Unmarshal(decrypted, &stripeCreds); err != nil {
		slog.Error("stripe webhook: failed to parse credentials", "account_id", params.AccountID, "error", err.Error())
		return nil
	}

	// Verify webhook signature and parse event.
	checkoutClient := s.checkoutClientFactory.Build(stripeCreds.PrivateKey)
	event, paymentIntent, apiErr := checkoutClient.ConstructWebhookEvent(params.RawPayload, params.StripeSignature, stripeCreds.WebhookSecret)
	if apiErr != nil {
		slog.Error("stripe webhook: signature verification failed", "account_id", params.AccountID, "error", apiErr.PublicMessage)
		return nil
	}

	if paymentIntent == nil {
		slog.Warn("stripe webhook: no payment intent in event", "account_id", params.AccountID, "event_type", event.Type)
		return nil
	}

	// Deduplicate via stripe_event_log.
	eventLogRepo := s.repos.NewStripeEventLogRepo()
	exists, apiErr := eventLogRepo.Exists(ctx, event.ID, paymentIntent.ID)
	if apiErr != nil {
		slog.Error("stripe webhook: dedup check failed", "account_id", params.AccountID, "error", apiErr.PublicMessage)
		return nil
	}
	if exists {
		slog.Info("stripe webhook: duplicate event, skipping", "event_id", event.ID, "object_id", paymentIntent.ID)
		return nil
	}

	// Log the event for deduplication.
	eventLogID, apiErr := id.GenID(id.StripeEventLogIDPrefix, nil)
	if apiErr != nil {
		slog.Error("stripe webhook: failed to generate event log ID", "error", apiErr.PublicMessage)
		return nil
	}
	if apiErr := eventLogRepo.Create(ctx, eventLogID, event.ID, paymentIntent.ID, event.Type); apiErr != nil {
		slog.Error("stripe webhook: failed to log event", "account_id", params.AccountID, "error", apiErr.PublicMessage)
		return nil
	}

	// Dispatch by event type.
	switch event.Type {
	case "payment_intent.succeeded":
		s.handlePaymentIntentSucceeded(ctx, params.AccountID, paymentIntent)
	case "payment_intent.payment_failed":
		s.handlePaymentIntentFailed(ctx, paymentIntent)
	case "payment_intent.canceled":
		s.handlePaymentIntentCanceled(ctx, paymentIntent)
	default:
		slog.Info("stripe webhook: unhandled event type", "event_type", event.Type)
	}

	return nil
}

func (s *stripeWebhookSvcImpl) handlePaymentIntentSucceeded(ctx context.Context, accountID string, pi *domain.StripePaymentIntent) {
	orderID := pi.Metadata["orderID"]
	customerID := pi.Metadata["customerID"]

	if orderID == "" || customerID == "" {
		slog.Warn("stripe webhook: missing metadata in payment intent", "payment_intent_id", pi.ID)
		return
	}

	// Verify order exists and buyer matches.
	orderRepo := s.repos.NewSalesOrderRepo()
	isForCustomer, apiErr := orderRepo.IsOrderForCustomer(ctx, orderID, customerID)
	if apiErr != nil {
		slog.Error("stripe webhook: failed to verify order", "order_id", orderID, "error", apiErr.PublicMessage)
		return
	}
	if !isForCustomer {
		slog.Warn("stripe webhook: order/buyer mismatch", "order_id", orderID, "customer_id", customerID)
		return
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *stripeWebhookSvcImpl) *apierror.APIError {
		// Create order_payment_intent record.
		opiID, apiErr := id.GenID(id.OrderPaymentIntentIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		opiRepo := txSvc.repos.NewOrderPaymentIntentRepo()
		if apiErr := opiRepo.Create(txCtx, opiID, pi.ID, orderID); apiErr != nil {
			return apiErr
		}

		// Fetch and increment transaction number.
		txRepo := txSvc.repos.NewTransactionRepo()
		number, apiErr := txRepo.FetchAndIncrementTransactionNumber(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}

		// Map payment method — check all types, matching Dashboard behavior.
		methodCode := mapPaymentMethodFromTypes(pi.PaymentMethodTypes)

		// Look up the dollar unit ID for the quantity record.
		dollarUnitID, apiErr := txRepo.GetDollarUnitID(txCtx)
		if apiErr != nil {
			return apiErr
		}

		// Create transaction with quantity.
		txID, apiErr := id.GenID(id.TransactionIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		note := "Payment captured by Stripe"
		// Convert from cents to dollars to match Dashboard behavior.
		amountDollars := float64(pi.Amount) / 100.0
		amountValue := fmt.Sprintf("%g", amountDollars)

		return txRepo.Create(txCtx, txID, number, "payment", accountID, customerID,
			&pi.ID, methodCode, nil, nil, &note, amountValue, dollarUnitID)
	})

	if apiErr != nil {
		slog.Error("stripe webhook: failed to process payment_intent.succeeded", "payment_intent_id", pi.ID, "error", apiErr.PublicMessage)
	}
}

// mapPaymentMethodFromTypes checks all payment method types (matching Dashboard's includes behavior)
// and returns the first matching internal transaction method code.
func mapPaymentMethodFromTypes(types []string) *string {
	for _, t := range types {
		mapped := domain.MapStripePaymentMethodToTransactionMethod(t)
		if mapped != nil {
			s := string(*mapped)
			return &s
		}
	}
	return nil
}

func (s *stripeWebhookSvcImpl) handlePaymentIntentFailed(ctx context.Context, pi *domain.StripePaymentIntent) {
	txRepo := s.repos.NewTransactionRepo()

	txRecord, apiErr := txRepo.FindByStripePaymentID(ctx, pi.ID)
	if apiErr != nil {
		// No-op if not found.
		slog.Info("stripe webhook: no transaction found for failed payment intent", "payment_intent_id", pi.ID)
		return
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *stripeWebhookSvcImpl) *apierror.APIError {
		txRepoInner := txSvc.repos.NewTransactionRepo()

		// Update note.
		if apiErr := txRepoInner.UpdateNote(txCtx, txRecord.ID, "Payment failed in Stripe"); apiErr != nil {
			return apiErr
		}

		// Delete allocations.
		if apiErr := txRepoInner.DeleteAllocations(txCtx, txRecord.ID); apiErr != nil {
			return apiErr
		}

		// Delete order_payment_intent.
		opiRepo := txSvc.repos.NewOrderPaymentIntentRepo()
		opi, apiErr := opiRepo.FindByPaymentIntentID(txCtx, pi.ID)
		if apiErr != nil {
			return nil // No-op if not found.
		}
		return opiRepo.Delete(txCtx, opi.ID)
	})

	if apiErr != nil {
		slog.Error("stripe webhook: failed to process payment_intent.payment_failed", "payment_intent_id", pi.ID, "error", apiErr.PublicMessage)
	}
}

func (s *stripeWebhookSvcImpl) handlePaymentIntentCanceled(ctx context.Context, pi *domain.StripePaymentIntent) {
	txRepo := s.repos.NewTransactionRepo()

	txRecord, apiErr := txRepo.FindByStripePaymentID(ctx, pi.ID)
	if apiErr != nil {
		// No-op if not found.
		slog.Info("stripe webhook: no transaction found for canceled payment intent", "payment_intent_id", pi.ID)
		return
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *stripeWebhookSvcImpl) *apierror.APIError {
		txRepoInner := txSvc.repos.NewTransactionRepo()

		// Delete allocations first.
		if apiErr := txRepoInner.DeleteAllocations(txCtx, txRecord.ID); apiErr != nil {
			return apiErr
		}

		// Delete the transaction.
		if apiErr := txRepoInner.Delete(txCtx, txRecord.ID); apiErr != nil {
			return apiErr
		}

		// Delete the quantity.
		if apiErr := txRepoInner.DeleteQuantity(txCtx, txRecord.AmountID); apiErr != nil {
			return apiErr
		}

		// Delete order_payment_intent.
		opiRepo := txSvc.repos.NewOrderPaymentIntentRepo()
		opi, apiErr := opiRepo.FindByPaymentIntentID(txCtx, pi.ID)
		if apiErr != nil {
			return nil // No-op if not found.
		}
		return opiRepo.Delete(txCtx, opi.ID)
	})

	if apiErr != nil {
		slog.Error("stripe webhook: failed to process payment_intent.canceled", "payment_intent_id", pi.ID, "error", apiErr.PublicMessage)
	}
}
