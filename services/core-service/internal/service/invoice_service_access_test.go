package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	invoiceSellerID   = "ac_seller"
	invoiceCustomerID = "ac_customer"
)

// invoiceActorCtx is a user of actorAccount reaching target as relation, holding exactly perms.
func invoiceActorCtx(relation types.IdentityRelationType, actorAccount, target string, perms ...string) context.Context {
	granted := make(map[string]bool, len(perms))
	for _, p := range perms {
		granted[p] = true
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: target},
		Actor: &types.IdentityActor{
			RelationType: relation,
			ID:           "us_actor",
			AccountID:    &actorAccount,
			Permissions:  granted,
		},
	})
}

// newGatedInvoiceSvc builds an invoice service whose repos and mediators fail the test if touched, so a
// refused caller is proven to be refused before anything is read.
func newGatedInvoiceSvc(t *testing.T) (domain.InvoiceSvc, *repositorymock.MockInvoiceRepo) {
	t.Helper()
	ctrl := gomock.NewController(t)
	invoiceRepo := repositorymock.NewMockInvoiceRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewInvoiceRepo().Return(invoiceRepo).AnyTimes()
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: mediatormock.NewMockIdempotencyMed(ctrl),
		ReadAccess:  mediatormock.NewMockReadAccessMed(ctrl),
		EditAccess:  mediatormock.NewMockEditAccessMed(ctrl),
	}).AnyTimes()
	return NewInvoiceSvc(&InvoiceSvcConfig{Repos: repos, MediatorFactory: meds, TxManager: &stubTxManager{factory: repos}}), invoiceRepo
}

type invoiceCall struct {
	name string
	call func(context.Context, domain.InvoiceSvc) *apierror.APIError
}

func invoiceCalls() []invoiceCall {
	return []invoiceCall{
		{"list", func(ctx context.Context, svc domain.InvoiceSvc) *apierror.APIError {
			_, apiErr := svc.ListInvoices(ctx, domain.ListInvoicesParams{Limit: 10})
			return apiErr
		}},
		{"get", func(ctx context.Context, svc domain.InvoiceSvc) *apierror.APIError {
			_, apiErr := svc.GetInvoice(ctx, domain.GetInvoiceParams{InvoiceID: "iv_1"})
			return apiErr
		}},
		{"update", func(ctx context.Context, svc domain.InvoiceSvc) *apierror.APIError {
			paid := true
			_, apiErr := svc.UpdateInvoice(ctx, domain.UpdateInvoiceParams{InvoiceID: "iv_1", IsPaidInFull: &paid})
			return apiErr
		}},
		{"list customer invoices", func(ctx context.Context, svc domain.InvoiceSvc) *apierror.APIError {
			_, apiErr := svc.ListCustomerInvoices(ctx, domain.ListCustomerInvoicesParams{CustomerAccountID: invoiceCustomerID, Limit: 10})
			return apiErr
		}},
	}
}

// An invoice is the seller's ledger: a customer's or supplier's user reaching the seller, or the seller's own staff
// reaching a customer's account, is refused whatever permissions it holds, before anything is read.
func TestInvoiceSvc_RefusesAnyoneButTheSellersOwnUsers(t *testing.T) {
	t.Parallel()

	all := []string{"invoices:read", "invoices:update", "customers:read", "customers:update", "suppliers:read", "suppliers:update"}
	actors := map[string]context.Context{
		"customer portal user":    invoiceActorCtx(types.IdentityRelationTypeCustomer, invoiceCustomerID, invoiceSellerID, all...),
		"supplier portal user":    invoiceActorCtx(types.IdentityRelationTypeSupplier, invoiceCustomerID, invoiceSellerID, all...),
		"seller staff at a buyer": invoiceActorCtx(types.IdentityRelationTypeInternal, invoiceSellerID, invoiceCustomerID, all...),
	}
	for name, ctx := range actors {
		for _, c := range invoiceCalls() {
			svc, _ := newGatedInvoiceSvc(t)
			apiErr := c.call(ctx, svc)
			if assert.NotNil(t, apiErr, "%s: %s", name, c.name) {
				assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, "%s: %s", name, c.name)
			}
		}
	}
}

// The seller's own users need the invoices permission for the action, not a customers or suppliers one.
func TestInvoiceSvc_RequiresTheInvoicesPermission(t *testing.T) {
	t.Parallel()

	for _, c := range invoiceCalls() {
		svc, _ := newGatedInvoiceSvc(t)
		ctx := invoiceActorCtx(types.IdentityRelationTypeInternal, invoiceSellerID, invoiceSellerID, "customers:read", "customers:update")
		apiErr := c.call(ctx, svc)
		if assert.NotNil(t, apiErr, c.name) {
			assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, c.name)
		}
	}

	svc, _ := newGatedInvoiceSvc(t)
	readOnly := invoiceActorCtx(types.IdentityRelationTypeInternal, invoiceSellerID, invoiceSellerID, "invoices:read")
	paid := true
	_, apiErr := svc.UpdateInvoice(readOnly, domain.UpdateInvoiceParams{InvoiceID: "iv_1", IsPaidInFull: &paid})
	require.NotNil(t, apiErr, "reading does not grant updating")
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

// A seller's user with invoices:read reads its own account's invoices, scoped to that account.
func TestInvoiceSvc_ScopesTheSellersReadsToItsAccount(t *testing.T) {
	t.Parallel()

	ctx := invoiceActorCtx(types.IdentityRelationTypeInternal, invoiceSellerID, invoiceSellerID, "invoices:read")

	svc, repo := newGatedInvoiceSvc(t)
	repo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p domain.ListInvoicesParams) (*domain.ListInvoicesResult, *apierror.APIError) {
			assert.Equal(t, invoiceSellerID, p.AccountID)
			return &domain.ListInvoicesResult{}, nil
		})
	_, apiErr := svc.ListInvoices(ctx, domain.ListInvoicesParams{Limit: 10})
	require.Nil(t, apiErr)

	repo.EXPECT().Get(gomock.Any(), domain.GetInvoiceParams{AccountID: invoiceSellerID, InvoiceID: "iv_1"}).Return(&domain.Invoice{ID: "iv_1"}, nil)
	got, apiErr := svc.GetInvoice(ctx, domain.GetInvoiceParams{InvoiceID: "iv_1"})
	require.Nil(t, apiErr)
	assert.Equal(t, "iv_1", got.ID)

	repo.EXPECT().ListByCustomer(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p domain.ListCustomerInvoicesParams) (*domain.ListCustomerInvoicesResult, *apierror.APIError) {
			assert.Equal(t, invoiceSellerID, p.AccountID)
			assert.Equal(t, invoiceCustomerID, p.CustomerAccountID)
			return &domain.ListCustomerInvoicesResult{}, nil
		})
	_, apiErr = svc.ListCustomerInvoices(ctx, domain.ListCustomerInvoicesParams{CustomerAccountID: invoiceCustomerID, Limit: 10})
	require.Nil(t, apiErr)
}

// Emailing an invoice outside the account is a 404: it has no recipients there either, and must not
// succeed as a no-op or be flagged sent.
func TestEmailRecord_InvoiceOutsideTheAccountIsNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	invoiceRepo := repositorymock.NewMockInvoiceRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewInvoiceRepo().Return(invoiceRepo).AnyTimes()
	idempotency := mediatormock.NewMockIdempotencyMed(ctrl)
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idempotency}).AnyTimes()
	svc := NewUtilsSvc(&UtilsSvcConfig{
		Repos:                 repos,
		MediatorFactory:       meds,
		TxManager:             &stubTxManager{factory: repos},
		NotificationPublisher: publishermock.NewMockNotificationPublisher(ctrl),
	})

	idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_1", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	invoiceRepo.EXPECT().Get(gomock.Any(), domain.GetInvoiceParams{AccountID: invoiceSellerID, InvoiceID: "iv_elsewhere"}).
		Return(nil, apierror.NewResourceNotFoundError("Invoice not found."))
	idempotency.EXPECT().CacheErrorResponse(gomock.Any(), "idk_1", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr })

	ctx := invoiceActorCtx(types.IdentityRelationTypeInternal, invoiceSellerID, invoiceSellerID, "invoices:read")
	apiErr := svc.EmailRecord(ctx, domain.EmailRecordParams{Type: domain.EmailRecordTypeInvoice, ID: "iv_elsewhere"})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
}
