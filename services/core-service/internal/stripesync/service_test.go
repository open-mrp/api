package stripesync

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	clientmock "github.com/open-mrp/api/services/core-service/internal/domain/mock/client"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	testOwnerAccountID    = "ac_owner"
	testCustomerAccountID = "ac_buyer"
)

type harness struct {
	svc             Service
	customerRepo    *repositorymock.MockCustomerRepo
	integrationRepo *repositorymock.MockAccountIntegrationRepo
	client          *clientmock.MockStripeCheckoutClient
	factory         *clientmock.MockStripeCheckoutClientFactory
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctrl := gomock.NewController(t)

	customerRepo := repositorymock.NewMockCustomerRepo(ctrl)
	integrationRepo := repositorymock.NewMockAccountIntegrationRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewCustomerRepo().Return(customerRepo).AnyTimes()
	repos.EXPECT().NewAccountIntegrationRepo().Return(integrationRepo).AnyTimes()

	client := clientmock.NewMockStripeCheckoutClient(ctrl)
	factory := clientmock.NewMockStripeCheckoutClientFactory(ctrl)

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	return &harness{
		svc:             NewService(repos, factory, key),
		customerRepo:    customerRepo,
		integrationRepo: integrationRepo,
		client:          client,
		factory:         factory,
	}
}

// expectConnected stubs an active Stripe integration whose credentials round-trip through the real encryption, so a change to the sealing scheme fails here rather than in production.
func (h *harness) expectConnected(t *testing.T) {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	credsJSON, err := json.Marshal(domain.StripeCredentials{PrivateKey: "sk_test_xxx"})
	require.NoError(t, err)
	encrypted, err := crypto.EncryptAESGCM(credsJSON, key, []byte(testOwnerAccountID), "k1")
	require.NoError(t, err)

	h.integrationRepo.EXPECT().
		HasIntegration(gomock.Any(), testOwnerAccountID, constants.IntegrationCodeStripe).
		Return(true, nil).Times(1)
	h.integrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), testOwnerAccountID, constants.IntegrationCodeStripe).
		Return(encrypted, true, nil).Times(1)
	h.factory.EXPECT().Build("sk_test_xxx").Return(h.client).Times(1)
}

func strptr(s string) *string { return &s }

// An account that never connected Stripe must not be a failure: every customer edit
// on such an account publishes this command, and erroring would dead-letter all of them.
func TestSyncCustomer_NoIntegrationIsNoOp(t *testing.T) {
	h := newHarness(t)
	h.integrationRepo.EXPECT().
		HasIntegration(gomock.Any(), testOwnerAccountID, constants.IntegrationCodeStripe).
		Return(false, nil).Times(1)

	require.Nil(t, h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID))
}

// A disconnected-but-present integration is the same no-op as never having one.
func TestSyncCustomer_InactiveIntegrationIsNoOp(t *testing.T) {
	h := newHarness(t)
	h.integrationRepo.EXPECT().
		HasIntegration(gomock.Any(), testOwnerAccountID, constants.IntegrationCodeStripe).
		Return(true, nil).Times(1)
	h.integrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), testOwnerAccountID, constants.IntegrationCodeStripe).
		Return("encrypted", false, nil).Times(1)

	require.Nil(t, h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID))
}

func TestSyncCustomer_CreatesAndLinksWhenUnlinked(t *testing.T) {
	h := newHarness(t)
	h.expectConnected(t)

	h.customerRepo.EXPECT().Get(gomock.Any(), testOwnerAccountID, testCustomerAccountID, nil).
		Return(&domain.Customer{
			ID:     testCustomerAccountID,
			Name:   "Buyer Co",
			Number: "301064",
			Email:  strptr("billing@buyer.example.com"),
		}, nil).Times(1)
	h.customerRepo.EXPECT().GetStripeCustomerID(gomock.Any(), testOwnerAccountID, testCustomerAccountID).
		Return(nil, nil, nil).Times(1)

	h.client.EXPECT().CreateStripeCustomer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateStripeCustomerParams) (*domain.StripeCustomer, *apierror.APIError) {
			require.Equal(t, "billing@buyer.example.com", params.Email)
			require.Equal(t, "Buyer Co", params.Name)
			require.Equal(t, "301064", params.Number)
			require.Equal(t, testCustomerAccountID, params.CustomerID)
			return &domain.StripeCustomer{ID: "cus_new"}, nil
		}).Times(1)

	// The link must be persisted, or the next sync creates a second Stripe customer.
	h.customerRepo.EXPECT().
		SetStripeCustomerID(gomock.Any(), testOwnerAccountID, testCustomerAccountID, "cus_new", "billing@buyer.example.com").
		Return(nil).Times(1)

	require.Nil(t, h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID))
}

// Stripe keys a customer on its email; there is nothing to create one with yet. The
// update that adds an email publishes another command, so this must not error.
func TestSyncCustomer_UnlinkedWithoutEmailIsNoOp(t *testing.T) {
	h := newHarness(t)
	h.expectConnected(t)

	h.customerRepo.EXPECT().Get(gomock.Any(), testOwnerAccountID, testCustomerAccountID, nil).
		Return(&domain.Customer{ID: testCustomerAccountID, Name: "Buyer Co", Number: "301064"}, nil).Times(1)
	h.customerRepo.EXPECT().GetStripeCustomerID(gomock.Any(), testOwnerAccountID, testCustomerAccountID).
		Return(nil, nil, nil).Times(1)

	require.Nil(t, h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID))
}

func TestSyncCustomer_UpdatesExistingAndRemirrorsEmail(t *testing.T) {
	h := newHarness(t)
	h.expectConnected(t)

	h.customerRepo.EXPECT().Get(gomock.Any(), testOwnerAccountID, testCustomerAccountID, nil).
		Return(&domain.Customer{
			ID:     testCustomerAccountID,
			Name:   "Buyer Co Renamed",
			Number: "301064",
			Email:  strptr("new@buyer.example.com"),
		}, nil).Times(1)
	h.customerRepo.EXPECT().GetStripeCustomerID(gomock.Any(), testOwnerAccountID, testCustomerAccountID).
		Return(strptr("cus_existing"), strptr("old@buyer.example.com"), nil).Times(1)

	h.client.EXPECT().UpdateStripeCustomer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateStripeCustomerParams) *apierror.APIError {
			require.Equal(t, "cus_existing", params.StripeCustomerID)
			require.NotNil(t, params.Email)
			require.Equal(t, "new@buyer.example.com", *params.Email)
			require.NotNil(t, params.Name)
			require.Equal(t, "Buyer Co Renamed", *params.Name)
			return nil
		}).Times(1)

	// The mirrored email is re-pointed at what Stripe now holds, keeping the existing link.
	h.customerRepo.EXPECT().
		SetStripeCustomerID(gomock.Any(), testOwnerAccountID, testCustomerAccountID, "cus_existing", "new@buyer.example.com").
		Return(nil).Times(1)

	require.Nil(t, h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID))
}

// A rename with an unchanged email still reaches Stripe, and costs no extra write to
// the relation — the mirrored email is already correct.
func TestSyncCustomer_UnchangedEmailSkipsRelationWrite(t *testing.T) {
	h := newHarness(t)
	h.expectConnected(t)

	h.customerRepo.EXPECT().Get(gomock.Any(), testOwnerAccountID, testCustomerAccountID, nil).
		Return(&domain.Customer{
			ID:     testCustomerAccountID,
			Name:   "Buyer Co Renamed",
			Number: "301064",
			Email:  strptr("same@buyer.example.com"),
		}, nil).Times(1)
	h.customerRepo.EXPECT().GetStripeCustomerID(gomock.Any(), testOwnerAccountID, testCustomerAccountID).
		Return(strptr("cus_existing"), strptr("same@buyer.example.com"), nil).Times(1)

	h.client.EXPECT().UpdateStripeCustomer(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	// No SetStripeCustomerID expectation: the controller fails the test if one is called.

	require.Nil(t, h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID))
}

// Clearing the email in OpenMRP must not strip Stripe's copy: live receipts and
// subscriptions point at it, and a cleared branding field is not that request.
func TestSyncCustomer_ClearedEmailIsNotPushedToStripe(t *testing.T) {
	h := newHarness(t)
	h.expectConnected(t)

	h.customerRepo.EXPECT().Get(gomock.Any(), testOwnerAccountID, testCustomerAccountID, nil).
		Return(&domain.Customer{ID: testCustomerAccountID, Name: "Buyer Co", Number: "301064"}, nil).Times(1)
	h.customerRepo.EXPECT().GetStripeCustomerID(gomock.Any(), testOwnerAccountID, testCustomerAccountID).
		Return(strptr("cus_existing"), strptr("old@buyer.example.com"), nil).Times(1)

	h.client.EXPECT().UpdateStripeCustomer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateStripeCustomerParams) *apierror.APIError {
			require.Nil(t, params.Email, "a cleared OpenMRP email must not blank the Stripe customer's")
			return nil
		}).Times(1)

	require.Nil(t, h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID))
}

// Credentials that no longer decrypt fail identically on every retry. Classifying them
// as transient would redeliver every customer edit on the account forever.
func TestSyncCustomer_UndecryptableCredentialsAreNotTransient(t *testing.T) {
	h := newHarness(t)
	h.integrationRepo.EXPECT().
		HasIntegration(gomock.Any(), testOwnerAccountID, constants.IntegrationCodeStripe).
		Return(true, nil).Times(1)
	h.integrationRepo.EXPECT().
		GetEncryptedCredentials(gomock.Any(), testOwnerAccountID, constants.IntegrationCodeStripe).
		Return("not-a-valid-sealed-blob", true, nil).Times(1)

	apiErr := h.svc.SyncCustomer(context.Background(), testOwnerAccountID, testCustomerAccountID)
	require.NotNil(t, apiErr)
	require.False(t, apiErr.IsTransient, "the consumer retries transient errors; this one can never succeed")
}
