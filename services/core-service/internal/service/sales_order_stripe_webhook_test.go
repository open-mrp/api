package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Tests for ProcessAccountStripeWebhook and its per-event handlers. These are methods on
// SalesOrderSvcTestSuite (sales_order_service_test.go) so they share its mock wiring; the
// webhook path is unauthenticated, so plain context.Background() stands in for the request
// context. A paid order checkout links its payment. The rest mirrors the retired legacy Express
// webhook: succeeded links the payment and records the receivables transaction, failed/canceled
// unwind it, payout.paid stamps funds arrival.

// expectStripeWebhookCreds stubs the integration-credential fetch with a sealed blob the
// service can round-trip decrypt, and routes the checkout-client factory to the suite's
// client mock.
func (suite *SalesOrderSvcTestSuite) expectStripeWebhookCreds(accountID string) {
	credsJSON, _ := json.Marshal(domain.StripeCredentials{PrivateKey: "sk_test_xxx", WebhookSecret: "whsec_test"})
	encrypted, err := crypto.EncryptAESGCM(credsJSON, suite.encryptionKey, []byte(accountID), "k1")
	suite.Require().NoError(err)

	suite.accountIntegrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), accountID, constants.IntegrationCodeStripe).
		Return(encrypted, true, nil).Times(1)
	suite.checkoutFactory.EXPECT().Build("sk_test_xxx").Return(suite.checkoutClient).Times(1)
}

// expectWebhookEvent stubs signature verification to yield the given event and payment intent.
func (suite *SalesOrderSvcTestSuite) expectWebhookEvent(event *domain.StripeWebhookEvent, pi *domain.StripePaymentIntent) {
	suite.checkoutClient.EXPECT().
		ConstructWebhookEvent(gomock.Any(), "sig_test", "whsec_test").
		Return(event, pi, nil).Times(1)
}

func webhookPaymentIntent(orderID, customerID string) *domain.StripePaymentIntent {
	metadata := map[string]string{}
	if orderID != "" {
		metadata["orderID"] = orderID
	}
	if customerID != "" {
		metadata["customerID"] = customerID
	}
	return &domain.StripePaymentIntent{
		ID:                 "pi_test",
		Amount:             2000,
		PaymentMethodTypes: []string{"card"},
		Metadata:           metadata,
	}
}

// stripeAudit is an audit event the Stripe paths published, with the identity it is attributed to.
type stripeAudit struct {
	identity *types.Identity
	event    audit.PublishedEvent
}

// requireStripeAudits asserts the published audit events are exactly want, each written as
// "resource_type action resource_id", and that every one is attributed to Stripe acting on ac_test.
func (suite *SalesOrderSvcTestSuite) requireStripeAudits(want ...string) []stripeAudit {
	var got []string
	var audits []stripeAudit
	for _, msg := range suite.outbox.published(contracts.PlatformEventAuditLogged) {
		var event audit.PublishedEvent
		suite.Require().NoError(json.Unmarshal(msg.Payload.Data, &event))
		identity := msg.Payload.Identity
		suite.Require().True(identity.IsSystem(), "a Stripe-driven write must be attributed to the system, got %+v", identity)
		suite.Equal(constants.SystemActorStripe.ActorID(), identity.Actor.ID)
		suite.Equal("ac_test", identity.Target.AccountID)
		got = append(got, string(event.ResourceType)+" "+string(event.Action)+" "+event.ResourceID)
		audits = append(audits, stripeAudit{identity: identity, event: event})
	}
	suite.ElementsMatch(want, got)
	return audits
}

// expectOrderPaymentIntents stubs the order's linked payment intents read before and after a link change.
func (suite *SalesOrderSvcTestSuite) expectOrderPaymentIntents(orderID string, before, after []string) {
	suite.orderRepo.EXPECT().GetPaymentIntentIDs(gomock.Any(), "ac_test", []string{orderID}).
		Return(map[string][]string{orderID: before}, nil).Times(1)
	suite.orderRepo.EXPECT().GetPaymentIntentIDs(gomock.Any(), "ac_test", []string{orderID}).
		Return(map[string][]string{orderID: after}, nil).Times(1)
}

// expectStripeTransaction stubs the lookup of the recorded transaction tr_1 and its allocations.
func (suite *SalesOrderSvcTestSuite) expectStripeTransaction(existing *domain.Transaction, allocations []*domain.TransactionAllocation) {
	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").
		Return(&domain.TransactionRecord{ID: "tr_1", Number: "1001", AmountID: "qt_1"}, nil).Times(1)
	suite.transactionRepo.EXPECT().Get(gomock.Any(), "ac_test", "tr_1").Return(existing, nil).Times(1)
	suite.transactionRepo.EXPECT().GetAllocations(gomock.Any(), "tr_1").Return(allocations, nil).Times(1)
}

func stripeTransactionFixture(note string) *domain.Transaction {
	return &domain.Transaction{ID: "tr_1", Number: "1001", AmountID: "qt_1", AmountValue: "20", Note: &note, StripePaymentID: new("pi_test")}
}

// --- Plumbing ---

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_IntegrationInactiveRejected() {
	suite.accountIntegrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), "ac_test", constants.IntegrationCodeStripe).
		Return("", false, nil).Times(1)

	apiErr := suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test")
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_MissingWebhookSecretRejected() {
	// Credentials sealed without a webhook secret: verification is impossible, so the
	// request must be rejected rather than acknowledged.
	credsJSON, _ := json.Marshal(domain.StripeCredentials{PrivateKey: "sk_test_xxx"})
	encrypted, err := crypto.EncryptAESGCM(credsJSON, suite.encryptionKey, []byte("ac_test"), "k1")
	suite.Require().NoError(err)
	suite.accountIntegrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), "ac_test", constants.IntegrationCodeStripe).
		Return(encrypted, true, nil).Times(1)

	apiErr := suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test")
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_BadSignatureRejected() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.checkoutClient.EXPECT().
		ConstructWebhookEvent(gomock.Any(), "sig_test", "whsec_test").
		Return(nil, nil, apierror.NewValidationError("Failed to verify webhook signature.")).Times(1)

	apiErr := suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test")
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_UnhandledEventTypeAcked() {
	suite.expectStripeWebhookCreds("ac_test")
	// No repo expectations beyond credential fetch: gomock fails the test if an
	// unhandled event type touches orders, transactions, or payment-intent links.
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "charge.updated"}, nil)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_SucceededWithoutPaymentIntentAcked() {
	suite.expectStripeWebhookCreds("ac_test")
	// A succeeded event whose payload didn't parse into a payment intent is acknowledged
	// without side effects rather than erroring into Stripe retries.
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, nil)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
}

// --- payment_intent.succeeded ---

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_LinksPaymentAndRecordsTransaction() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", BuyerAccountID: "ac_buyer"}, nil).Times(1)

	// Order link: not yet linked, so a link row is created.
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.opiRepo.EXPECT().Create(gomock.Any(), gomock.Any(), "pi_test", "or_1").Return(nil).Times(1)
	suite.expectOrderPaymentIntents("or_1", nil, []string{"pi_test"})

	// Receivables transaction: none recorded yet, so one is created with the legacy shape.
	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.transactionRepo.EXPECT().FetchAndIncrementTransactionNumber(gomock.Any(), "ac_test").Return("1001", nil).Times(1)
	suite.transactionRepo.EXPECT().GetDollarUnitID(gomock.Any()).Return("un_dollar", nil).Times(1)
	suite.transactionRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), "1001", string(constants.TransactionTypePayment), "ac_test", "ac_buyer", gomock.Any(), gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Any(), "20", "un_dollar", gomock.Nil(), gomock.Nil()).
		DoAndReturn(func(_ context.Context, _, _, _, _, _ string, stripePaymentID, methodCode, _, _, note *string, _, _ string, _, _ *time.Time) *apierror.APIError {
			suite.Require().NotNil(stripePaymentID)
			suite.Equal("pi_test", *stripePaymentID)
			suite.Require().NotNil(methodCode)
			suite.Equal(string(domain.TransactionMethodCreditCard), *methodCode)
			suite.Require().NotNil(note)
			suite.Equal("Payment captured by Stripe", *note)
			return nil
		}).Times(1)
	suite.transactionRepo.EXPECT().Get(gomock.Any(), "ac_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, txID string) (*domain.Transaction, *apierror.APIError) {
			return &domain.Transaction{ID: txID, Number: "1001", AmountValue: "20"}, nil
		}).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))

	audits := suite.outbox.published(contracts.PlatformEventAuditLogged)
	suite.Require().Len(audits, 2)
	var created audit.PublishedEvent
	suite.Require().NoError(json.Unmarshal(audits[1].Payload.Data, &created))
	suite.requireStripeAudits("sales_order update or_1", "transaction create "+created.ResourceID)
	var linked audit.PublishedEvent
	suite.Require().NoError(json.Unmarshal(audits[0].Payload.Data, &linked))
	suite.Require().Len(linked.Changes, 1)
	suite.Equal("payment_intent_ids", linked.Changes[0].Field)
	suite.JSONEq(`[]`, string(linked.Changes[0].OldValue))
	suite.JSONEq(`["pi_test"]`, string(linked.Changes[0].NewValue))
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_NoOrderMetadataAcked() {
	suite.expectStripeWebhookCreds("ac_test")
	// No orderID in metadata (e.g. a payment created outside OpenMRP on the same Stripe
	// account): acknowledged with zero side effects.
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("", "ac_buyer"))

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_UnknownOrderAcked() {
	suite.expectStripeWebhookCreds("ac_test")
	// The metadata is vendor-controlled: an orderID that doesn't resolve within this
	// account (another tenant's order, or a bogus value) must be ignored, not linked
	// and not turned into an error that would make Stripe retry.
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_other_tenant", "ac_buyer"))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_other_tenant").
		Return(nil, apierror.NewResourceNotFoundError("Resource not found.")).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_OrderLookupErrorPropagates() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(nil, apierror.NewInternalError(nil, "db down")).Times(1)

	apiErr := suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test")
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_RedeliveryBackfillsMissingTransaction() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", BuyerAccountID: "ac_buyer"}, nil).Times(1)

	// Already linked: the link is skipped, but the transaction is still recorded if
	// missing (covers events replayed after the transaction path shipped).
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").
		Return(&domain.OrderPaymentIntent{ID: "orpyie_1", PaymentIntentID: "pi_test", SalesOrderID: "or_1"}, nil).Times(1)

	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.transactionRepo.EXPECT().FetchAndIncrementTransactionNumber(gomock.Any(), "ac_test").Return("1002", nil).Times(1)
	suite.transactionRepo.EXPECT().GetDollarUnitID(gomock.Any()).Return("un_dollar", nil).Times(1)
	suite.transactionRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), "1002", string(constants.TransactionTypePayment), "ac_test", "ac_buyer", gomock.Any(), gomock.Any(), gomock.Nil(), gomock.Nil(), gomock.Any(), "20", "un_dollar", gomock.Nil(), gomock.Nil()).
		Return(nil).Times(1)
	suite.transactionRepo.EXPECT().Get(gomock.Any(), "ac_test", gomock.Any()).
		Return(&domain.Transaction{ID: "tr_new"}, nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits("transaction create tr_new")
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_FullRedeliveryIsIdempotent() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", BuyerAccountID: "ac_buyer"}, nil).Times(1)

	// Both the link and the transaction already exist: a duplicate delivery must write nothing.
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").
		Return(&domain.OrderPaymentIntent{ID: "orpyie_1", PaymentIntentID: "pi_test", SalesOrderID: "or_1"}, nil).Times(1)
	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").
		Return(&domain.TransactionRecord{ID: "tr_1", Number: "1001", AmountID: "qt_1"}, nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits()
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_CustomerMismatchLinksWithoutTransaction() {
	suite.expectStripeWebhookCreds("ac_test")
	// Metadata claims a customer that is not the order's buyer: the payment link is still
	// recorded (the order genuinely got paid) but no receivables transaction is booked
	// against the mismatched customer.
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_1", "ac_impostor"))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", BuyerAccountID: "ac_buyer"}, nil).Times(1)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.opiRepo.EXPECT().Create(gomock.Any(), gomock.Any(), "pi_test", "or_1").Return(nil).Times(1)
	suite.expectOrderPaymentIntents("or_1", nil, []string{"pi_test"})

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits("sales_order update or_1")
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_NoCustomerMetadataLinksWithoutTransaction() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_1", ""))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", BuyerAccountID: "ac_buyer"}, nil).Times(1)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.opiRepo.EXPECT().Create(gomock.Any(), gomock.Any(), "pi_test", "or_1").Return(nil).Times(1)
	suite.expectOrderPaymentIntents("or_1", []string{"pi_earlier"}, []string{"pi_earlier", "pi_test"})

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Succeeded_TransactionCreateErrorPropagates() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.succeeded"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", BuyerAccountID: "ac_buyer"}, nil).Times(1)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.opiRepo.EXPECT().Create(gomock.Any(), gomock.Any(), "pi_test", "or_1").Return(nil).Times(1)
	suite.expectOrderPaymentIntents("or_1", nil, []string{"pi_test"})
	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.transactionRepo.EXPECT().FetchAndIncrementTransactionNumber(gomock.Any(), "ac_test").
		Return("", apierror.NewInternalError(nil, "lock timeout")).Times(1)

	apiErr := suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test")
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

// --- payment_intent.payment_failed ---

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_PaymentFailed_MarksTransactionAndUnlinksOrder() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.payment_failed"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.expectStripeTransaction(stripeTransactionFixture("Payment captured by Stripe"), []*domain.TransactionAllocation{{ID: "ta_1", TransactionID: "tr_1", InvoiceID: "in_1", AmountValue: "20"}})
	// The transaction survives (with a failure note) but its allocations and the order
	// link are unwound — mirroring the legacy webhook exactly.
	suite.transactionRepo.EXPECT().UpdateNote(gomock.Any(), "tr_1", "Payment failed in Stripe").Return(nil).Times(1)
	suite.transactionRepo.EXPECT().DeleteAllocations(gomock.Any(), "tr_1").Return(nil).Times(1)
	suite.transactionRepo.EXPECT().Get(gomock.Any(), "ac_test", "tr_1").Return(stripeTransactionFixture("Payment failed in Stripe"), nil).Times(1)
	suite.deletedRecordRepo.EXPECT().
		CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeTransactionAllocation, "ta_1", "ac_test", gomock.Any()).Return(nil).Times(1)
	suite.paymentFlagsMed.EXPECT().Enqueue(gomock.Any(), "ac_test", []string{"tr_1"}, []string{"in_1"}).Return(nil).Times(1)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").
		Return(&domain.OrderPaymentIntent{ID: "orpyie_1", PaymentIntentID: "pi_test", SalesOrderID: "or_1"}, nil).Times(1)
	suite.expectOrderPaymentIntents("or_1", []string{"pi_test"}, nil)
	suite.opiRepo.EXPECT().Delete(gomock.Any(), "orpyie_1").Return(nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))

	audits := suite.requireStripeAudits("transaction update tr_1", "transaction_allocation delete ta_1", "sales_order update or_1")
	for _, a := range audits {
		if a.event.ResourceType == constants.ObjectTypeTransaction {
			suite.Require().Len(a.event.Changes, 1)
			suite.Equal("note", a.event.Changes[0].Field)
			suite.JSONEq(`"Payment failed in Stripe"`, string(a.event.Changes[0].NewValue))
		}
	}
}

// A transaction the payment intent points at in another account is left alone: the event can only speak for its own account.
func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_PaymentFailed_OtherAccountsTransactionIsSkipped() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.payment_failed"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").
		Return(&domain.TransactionRecord{ID: "tr_1", Number: "1001", AmountID: "qt_1"}, nil).Times(1)
	suite.transactionRepo.EXPECT().Get(gomock.Any(), "ac_test", "tr_1").
		Return(nil, apierror.NewResourceNotFoundError("Transaction not found.")).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits()
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_PaymentFailed_NoTransactionIsNoop() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.payment_failed"}, webhookPaymentIntent("or_1", "ac_buyer"))

	// No recorded transaction: nothing is touched — including the order link, matching
	// the legacy early return.
	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_PaymentFailed_NoLinkStillMarksTransaction() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.payment_failed"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.expectStripeTransaction(stripeTransactionFixture("Payment captured by Stripe"), nil)
	suite.transactionRepo.EXPECT().UpdateNote(gomock.Any(), "tr_1", "Payment failed in Stripe").Return(nil).Times(1)
	suite.transactionRepo.EXPECT().DeleteAllocations(gomock.Any(), "tr_1").Return(nil).Times(1)
	suite.transactionRepo.EXPECT().Get(gomock.Any(), "ac_test", "tr_1").Return(stripeTransactionFixture("Payment failed in Stripe"), nil).Times(1)
	suite.paymentFlagsMed.EXPECT().Enqueue(gomock.Any(), "ac_test", []string{"tr_1"}, []string{}).Return(nil).Times(1)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits("transaction update tr_1")
}

// --- payment_intent.canceled ---

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Canceled_DeletesTransactionChainAndUnlinks() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.canceled"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.expectStripeTransaction(stripeTransactionFixture("Payment captured by Stripe"), []*domain.TransactionAllocation{{ID: "ta_1", TransactionID: "tr_1", InvoiceID: "in_1", AmountValue: "20"}})
	// Unlike a failure, a cancellation removes the transaction entirely: allocations
	// first (FK), then the transaction, then its amount quantity, then the order link.
	suite.deletedRecordRepo.EXPECT().
		CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeTransaction, "tr_1", "ac_test", gomock.Any()).Return(nil).Times(1)
	suite.deletedRecordRepo.EXPECT().
		CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeTransactionAllocation, "ta_1", "ac_test", gomock.Any()).Return(nil).Times(1)
	suite.paymentFlagsMed.EXPECT().Enqueue(gomock.Any(), "ac_test", []string{"tr_1"}, []string{"in_1"}).Return(nil).Times(1)
	deleteAllocations := suite.transactionRepo.EXPECT().DeleteAllocations(gomock.Any(), "tr_1").Return(nil).Times(1)
	deleteTx := suite.transactionRepo.EXPECT().Delete(gomock.Any(), "tr_1").Return(nil).Times(1).After(deleteAllocations)
	suite.transactionRepo.EXPECT().DeleteQuantity(gomock.Any(), "qt_1").Return(nil).Times(1).After(deleteTx)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").
		Return(&domain.OrderPaymentIntent{ID: "orpyie_1", PaymentIntentID: "pi_test", SalesOrderID: "or_1"}, nil).Times(1)
	suite.expectOrderPaymentIntents("or_1", []string{"pi_test"}, nil)
	suite.opiRepo.EXPECT().Delete(gomock.Any(), "orpyie_1").Return(nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits("transaction_allocation delete ta_1", "transaction delete tr_1", "sales_order update or_1")
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_Canceled_NoTransactionIsNoop() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payment_intent.canceled"}, webhookPaymentIntent("or_1", "ac_buyer"))

	suite.transactionRepo.EXPECT().FindByStripePaymentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
}

// --- payout.paid ---

// Reconciling a payout reads every charge in it from Stripe, so the webhook only queues it.
func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_PayoutPaid_IsQueuedNotProcessed() {
	suite.expectStripeWebhookCreds("ac_test")
	rawPayout, _ := json.Marshal(map[string]any{"id": "po_test", "arrival_date": 1783500000})
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "payout.paid", RawJSON: rawPayout}, nil)
	// No ListPayoutPaymentIntentIDs expectation: a Stripe read here fails the test.

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))

	queued := suite.outbox.published(contracts.CoreEventAccountStripePayoutPaid)
	suite.Require().Len(queued, 1)
	var data messaging.AccountStripePayoutPaidData
	suite.Require().NoError(json.Unmarshal(queued[0].Payload.Data, &data))
	suite.Equal("ac_test", data.AccountID)
	suite.Equal("evt_1", data.EventID)
	suite.JSONEq(string(rawPayout), string(data.Event))
}

func (suite *SalesOrderSvcTestSuite) TestReconcileAccountStripePayout_StampsFundsReceived() {
	suite.expectStripeWebhookCreds("ac_test")
	arrival := int64(1783500000)
	rawPayout, _ := json.Marshal(map[string]any{"id": "po_test", "arrival_date": arrival})

	suite.checkoutClient.EXPECT().ListPayoutPaymentIntentIDs(gomock.Any(), "po_test").
		Return([]string{"pi_a", "pi_b"}, nil).Times(1)
	suite.transactionRepo.EXPECT().LockIDsByStripePaymentIDs(gomock.Any(), "ac_test", []string{"pi_a", "pi_b"}).
		Return([]string{"tr_a", "tr_b"}, nil).Times(1)
	landed := time.Unix(arrival, 0)
	suite.transactionRepo.EXPECT().GetByIDs(gomock.Any(), "ac_test", []string{"tr_a", "tr_b"}).
		Return([]*domain.Transaction{{ID: "tr_a"}, {ID: "tr_b", FundsReceivedAt: &landed}}, nil).Times(1)
	suite.transactionRepo.EXPECT().
		UpdateFundsReceivedByStripePaymentIDs(gomock.Any(), "ac_test", []string{"pi_a", "pi_b"}, landed).
		Return(nil).Times(1)
	suite.transactionRepo.EXPECT().GetByIDs(gomock.Any(), "ac_test", []string{"tr_a", "tr_b"}).
		Return([]*domain.Transaction{{ID: "tr_a", FundsReceivedAt: &landed}, {ID: "tr_b", FundsReceivedAt: &landed}}, nil).Times(1)

	suite.Nil(suite.svc.ReconcileAccountStripePayout(context.Background(), "ac_test", rawPayout))

	// tr_b already carried this payout's date (a redelivery), so only tr_a changed.
	audits := suite.requireStripeAudits("transaction update tr_a")
	suite.Require().Len(audits[0].event.Changes, 1)
	suite.Equal("funds_received_at", audits[0].event.Changes[0].Field)
}

func (suite *SalesOrderSvcTestSuite) TestReconcileAccountStripePayout_NoRecordedTransactionsIsNoop() {
	suite.expectStripeWebhookCreds("ac_test")
	rawPayout, _ := json.Marshal(map[string]any{"id": "po_test", "arrival_date": 1783500000})

	suite.checkoutClient.EXPECT().ListPayoutPaymentIntentIDs(gomock.Any(), "po_test").
		Return([]string{"pi_elsewhere"}, nil).Times(1)
	suite.transactionRepo.EXPECT().LockIDsByStripePaymentIDs(gomock.Any(), "ac_test", []string{"pi_elsewhere"}).
		Return(nil, nil).Times(1)

	suite.Nil(suite.svc.ReconcileAccountStripePayout(context.Background(), "ac_test", rawPayout))
	suite.requireStripeAudits()
}

// --- checkout.session.completed ---

// checkoutSessionJSON is a completed order checkout as Stripe reports it, with the overrides applied on top.
func checkoutSessionJSON(overrides map[string]any) []byte {
	sess := map[string]any{
		"id":             "cs_test",
		"payment_intent": "pi_test",
		"payment_status": "paid",
		"amount_total":   2000,
		"currency":       "usd",
		"customer":       "cus_buyer",
		"metadata": map[string]string{
			"orderID":             "or_1",
			"customerID":          "ac_buyer",
			"stripeCustomerID":    "cus_buyer",
			"expectedAmountCents": "2000",
			"expectedCurrency":    "usd",
		},
	}
	for k, v := range overrides {
		sess[k] = v
	}
	raw, _ := json.Marshal(sess)
	return raw
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_CheckoutCompleted_LinksPayment() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "checkout.session.completed", RawJSON: checkoutSessionJSON(nil)}, nil)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1", BuyerAccountID: "ac_buyer"}, nil).Times(1)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").Return(nil, nil).Times(1)
	suite.opiRepo.EXPECT().Create(gomock.Any(), gomock.Any(), "pi_test", "or_1").Return(nil).Times(1)
	suite.expectOrderPaymentIntents("or_1", nil, []string{"pi_test"})

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits("sales_order update or_1")
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_CheckoutCompleted_AlreadyLinkedWritesNothing() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "checkout.session.completed", RawJSON: checkoutSessionJSON(nil)}, nil)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(&domain.SalesOrder{ID: "or_1"}, nil).Times(1)
	suite.opiRepo.EXPECT().FindByPaymentIntentID(gomock.Any(), "pi_test").
		Return(&domain.OrderPaymentIntent{ID: "orpyie_1", PaymentIntentID: "pi_test", SalesOrderID: "or_1"}, nil).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits()
}

// The metadata is vendor-controlled, so an order outside this account is acknowledged and not linked.
func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_CheckoutCompleted_UnknownOrderAcked() {
	suite.expectStripeWebhookCreds("ac_test")
	suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "checkout.session.completed", RawJSON: checkoutSessionJSON(nil)}, nil)

	suite.orderRepo.EXPECT().Get(gomock.Any(), "ac_test", "or_1").
		Return(nil, apierror.NewResourceNotFoundError("Resource not found.")).Times(1)

	suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
	suite.requireStripeAudits()
}

func (suite *SalesOrderSvcTestSuite) TestAccountWebhook_CheckoutCompleted_GuardsSkipWithoutLinking() {
	metadata := func(drop string, set map[string]string) map[string]string {
		m := map[string]string{
			"orderID":             "or_1",
			"stripeCustomerID":    "cus_buyer",
			"expectedAmountCents": "2000",
			"expectedCurrency":    "usd",
		}
		delete(m, drop)
		for k, v := range set {
			m[k] = v
		}
		return m
	}
	cases := map[string]map[string]any{
		"no order metadata":       {"metadata": metadata("orderID", nil)},
		"not paid":                {"payment_status": "unpaid"},
		"no payment intent":       {"payment_intent": ""},
		"invalid expected amount": {"metadata": metadata("", map[string]string{"expectedAmountCents": "abc"})},
		"amount mismatch":         {"amount_total": 1999},
		"currency mismatch":       {"currency": "eur"},
		"no expected currency":    {"metadata": metadata("expectedCurrency", nil)},
		"customer mismatch":       {"customer": "cus_other"},
		"no expected customer":    {"metadata": metadata("stripeCustomerID", nil)},
	}
	for name, overrides := range cases {
		suite.Run(name, func() {
			suite.SetupTest()
			suite.expectStripeWebhookCreds("ac_test")
			// No order, link or transaction expectations: gomock fails the case if a skipped session touches them.
			suite.expectWebhookEvent(&domain.StripeWebhookEvent{ID: "evt_1", Type: "checkout.session.completed", RawJSON: checkoutSessionJSON(overrides)}, nil)

			suite.Nil(suite.svc.ProcessAccountStripeWebhook(context.Background(), "ac_test", []byte("{}"), "sig_test"))
			suite.requireStripeAudits()
		})
	}
}

func (suite *SalesOrderSvcTestSuite) TestReconcileAccountStripePayout_NoFundedPaymentsIsNoop() {
	suite.expectStripeWebhookCreds("ac_test")
	rawPayout, _ := json.Marshal(map[string]any{"id": "po_test", "arrival_date": 1783500000})

	suite.checkoutClient.EXPECT().ListPayoutPaymentIntentIDs(gomock.Any(), "po_test").
		Return(nil, nil).Times(1)

	suite.Nil(suite.svc.ReconcileAccountStripePayout(context.Background(), "ac_test", rawPayout))
}

func (suite *SalesOrderSvcTestSuite) TestReconcileAccountStripePayout_UnparseablePayloadAcked() {
	suite.expectStripeWebhookCreds("ac_test")
	// A payout object we can't parse is acknowledged (logged, no retry storm) rather
	// than failed.
	rawPayout := []byte("not-json")

	suite.Nil(suite.svc.ReconcileAccountStripePayout(context.Background(), "ac_test", rawPayout))
}

func (suite *SalesOrderSvcTestSuite) TestReconcileAccountStripePayout_StripeErrorPropagates() {
	suite.expectStripeWebhookCreds("ac_test")
	rawPayout, _ := json.Marshal(map[string]any{"id": "po_test", "arrival_date": 1783500000})

	suite.checkoutClient.EXPECT().ListPayoutPaymentIntentIDs(gomock.Any(), "po_test").
		Return(nil, apierror.NewInternalError(nil, "stripe unavailable")).Times(1)

	apiErr := suite.svc.ReconcileAccountStripePayout(context.Background(), "ac_test", rawPayout)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

// --- method mapping ---

func TestStripeTransactionMethodCode(t *testing.T) {
	t.Parallel()

	creditCard := string(domain.TransactionMethodCreditCard)
	ach := string(domain.TransactionMethodACH)

	cases := []struct {
		name    string
		methods []string
		want    *string
	}{
		{"card maps to credit card", []string{"card"}, &creditCard},
		{"link maps to credit card", []string{"link"}, &creditCard},
		{"us bank account maps to ach", []string{"us_bank_account"}, &ach},
		{"card wins over us bank account regardless of order", []string{"us_bank_account", "card"}, &creditCard},
		{"us bank account wins over link", []string{"link", "us_bank_account"}, &ach},
		{"unknown methods map to nil", []string{"paypal"}, nil},
		{"empty maps to nil", nil, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripeTransactionMethodCode(tc.methods)
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			if assert.NotNil(t, got) {
				assert.Equal(t, *tc.want, *got)
			}
		})
	}
}
