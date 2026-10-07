package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	inclSellerID = "ac_seller"
	inclBuyerID  = "ac_buyer"
	inclOtherID  = "ac_other_buyer"
)

// inclCtx is a user of actorAccount reaching the seller as relation, holding exactly perms; included marks it as loading what an authorized request includes.
func inclCtx(relation types.IdentityRelationType, actorAccount string, included bool, perms ...string) context.Context {
	granted := make(map[string]bool, len(perms))
	for _, p := range perms {
		granted[p] = true
	}
	identity := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: inclSellerID},
		Actor:  &types.IdentityActor{RelationType: relation, ID: "us_actor", AccountID: &actorAccount, Permissions: granted},
	}
	if included {
		identity = identity.ForIncludeReads()
	}
	return appctx.WithIdentity(context.Background(), identity)
}

func inclPortal(included bool) context.Context {
	return inclCtx(types.IdentityRelationTypeCustomer, inclBuyerID, included)
}

func inclStaff(included bool, perms ...string) context.Context {
	return inclCtx(types.IdentityRelationTypeInternal, inclSellerID, included, perms...)
}

type inclHarness struct {
	ctrl       *gomock.Controller
	repos      *factorymock.MockRepoFactory
	meds       *factorymock.MockMediatorFactory
	readAccess *mediatormock.MockReadAccessMed
}

func newInclHarness(t *testing.T) *inclHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &inclHarness{
		ctrl:       ctrl,
		repos:      factorymock.NewMockRepoFactory(ctrl),
		meds:       factorymock.NewMockMediatorFactory(ctrl),
		readAccess: mediatormock.NewMockReadAccessMed(ctrl),
	}
	h.meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		ReadAccess:  h.readAccess,
		Idempotency: mediatormock.NewMockIdempotencyMed(ctrl),
		EditAccess:  mediatormock.NewMockEditAccessMed(ctrl),
	}).AnyTimes()
	return h
}

func (h *inclHarness) departments() (domain.DepartmentSvc, *repositorymock.MockDepartmentRepo) {
	repo := repositorymock.NewMockDepartmentRepo(h.ctrl)
	h.repos.EXPECT().NewDepartmentRepo().Return(repo).AnyTimes()
	return NewDepartmentSvc(&DepartmentSvcConfig{Repos: h.repos, MediatorFactory: h.meds, JobSvcFactory: NewJobSvcFactory(), TxManager: &stubTxManager{factory: h.repos}}), repo
}

func requireRefused(t *testing.T, apiErr *apierror.APIError, msgAndArgs ...any) {
	t.Helper()
	require.NotNil(t, apiErr, msgAndArgs...)
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, msgAndArgs...)
}

// A record an authorized request includes is read for staff without its own permission and for portal users, from the seller's account; without the flag both are refused before anything is read.
func TestIncludeReads_InternalOnlyBatchReadAdmitsTheIncludingCaller(t *testing.T) {
	t.Parallel()

	for name, ctx := range map[string]context.Context{
		"staff without departments:read": inclStaff(true, "sales_orders:read"),
		"customer portal user":           inclPortal(true),
	} {
		svc, repo := newInclHarness(t).departments()
		repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, []string{"dep_1"}).Return([]*domain.Department{{ID: "dep_1"}}, nil)

		got, apiErr := svc.BatchGetDepartmentsByIDs(ctx, []string{"dep_1"})
		require.Nil(t, apiErr, name)
		require.Len(t, got, 1, name)
	}

	for name, ctx := range map[string]context.Context{
		"staff without departments:read": inclStaff(false, "sales_orders:read"),
		"customer portal user":           inclPortal(false),
	} {
		svc, _ := newInclHarness(t).departments()
		_, apiErr := svc.BatchGetDepartmentsByIDs(ctx, []string{"dep_1"})
		requireRefused(t, apiErr, name)
	}
}

// The flag never authorizes a write: the repo mock has no expectations, so any write that got past the gate fails the test.
func TestIncludeReads_WritesIgnoreTheFlag(t *testing.T) {
	t.Parallel()

	for name, ctx := range map[string]context.Context{
		"staff without departments:create": inclStaff(true, "departments:read"),
		"customer portal user":             inclPortal(true),
	} {
		svc, _ := newInclHarness(t).departments()
		_, apiErr := svc.CreateDepartment(ctx, domain.CreateDepartmentParams{AccountID: inclSellerID, Name: "Assembly"})
		require.NotNil(t, apiErr, name)
	}

	h := newInclHarness(t)
	h.repos.EXPECT().NewInvoiceRepo().Return(repositorymock.NewMockInvoiceRepo(h.ctrl)).AnyTimes()
	invoices := NewInvoiceSvc(&InvoiceSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}})
	_, apiErr := invoices.UpdateInvoice(inclStaff(true, "invoices:read"), domain.UpdateInvoiceParams{InvoiceID: "inv_1"})
	requireRefused(t, apiErr, "an invoice update on the flag")
}

// GetInvoice backs the invoice include, so a portal user loading an included invoice reads it; the same call without the flag is refused.
func TestIncludeReads_InvoiceReadAdmitsTheIncludingCaller(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	repo := repositorymock.NewMockInvoiceRepo(h.ctrl)
	h.repos.EXPECT().NewInvoiceRepo().Return(repo).AnyTimes()
	svc := NewInvoiceSvc(&InvoiceSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}})
	repo.EXPECT().Get(gomock.Any(), domain.GetInvoiceParams{AccountID: inclSellerID, InvoiceID: "inv_1"}).Return(&domain.Invoice{ID: "inv_1"}, nil)

	got, apiErr := svc.GetInvoice(inclPortal(true), domain.GetInvoiceParams{InvoiceID: "inv_1"})
	require.Nil(t, apiErr)
	assert.Equal(t, "inv_1", got.ID)

	_, apiErr = svc.GetInvoice(inclPortal(false), domain.GetInvoiceParams{InvoiceID: "inv_1"})
	requireRefused(t, apiErr)
}

// A portal user normally sees only its own customer record; one an authorized request includes (a parent account, say) is read too, still from the seller's account.
func TestIncludeReads_CustomerBatchDropsTheBuyerFilterForIncludes(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	repo := repositorymock.NewMockCustomerRepo(h.ctrl)
	h.repos.EXPECT().NewCustomerRepo().Return(repo).AnyTimes()
	svc := NewCustomerSvc(&CustomerSvcConfig{Repos: h.repos, MediatorFactory: h.meds, JobSvcFactory: factorymock.NewMockJobSvcFactory(h.ctrl), TxManager: &stubTxManager{factory: h.repos}})
	h.readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), inclBuyerID, inclSellerID).Return(nil).Times(2)

	ids := []string{inclBuyerID, inclOtherID}
	repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, []string{inclBuyerID}).Return([]*domain.Customer{{}}, nil)
	_, apiErr := svc.BatchGetCustomers(inclPortal(false), ids)
	require.Nil(t, apiErr)

	repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, ids).Return([]*domain.Customer{{}, {}}, nil)
	got, apiErr := svc.BatchGetCustomers(inclPortal(true), ids)
	require.Nil(t, apiErr)
	assert.Len(t, got, 2)
}

// The buyer filter on sales orders applies to a portal user's own reads, not to an order an authorized request includes.
func TestIncludeReads_SalesOrderBatchDropsTheBuyerFilterForIncludes(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	repo := repositorymock.NewMockSalesOrderRepo(h.ctrl)
	h.repos.EXPECT().NewSalesOrderRepo().Return(repo).AnyTimes()
	svc := NewSalesOrderSvc(&SalesOrderSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}})
	h.readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), inclBuyerID, inclSellerID).Return(nil).Times(2)

	buyer := inclBuyerID
	repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, &buyer, []string{"so_1"}).Return(nil, nil)
	_, apiErr := svc.BatchGetSalesOrders(inclPortal(false), []string{"so_1"}, nil)
	require.Nil(t, apiErr)

	repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, (*string)(nil), []string{"so_1"}).Return(nil, nil)
	_, apiErr = svc.BatchGetSalesOrders(inclPortal(true), []string{"so_1"}, nil)
	require.Nil(t, apiErr)
}

// Users load for portal includes (a sales rep, a responsible user) through the relation the portal user holds, which runs from the seller to the buyer.
func TestIncludeReads_UserBatchAdmitsAPortalUserAlongTheRelation(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	repo := repositorymock.NewMockUserRepo(h.ctrl)
	h.repos.EXPECT().NewUserRepo().Return(repo).AnyTimes()
	svc := NewUserSvc(&UserSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}, S3Client: &capturingObjectStore{}})

	_, apiErr := svc.BatchGetUsersByIDs(inclPortal(false), []string{"us_rep"})
	requireRefused(t, apiErr, "a portal user reading users directly")

	h.readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), inclBuyerID, inclSellerID).Return(nil)
	repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, []string{"us_rep"}).Return([]*domain.UserRecord{{ID: "us_rep"}}, nil)
	got, apiErr := svc.BatchGetUsersByIDs(inclPortal(true), []string{"us_rep"})
	require.Nil(t, apiErr)
	require.Len(t, got, 1)
}

// Account users take the same route as users: the counterparty relation admits the portal user, the seller's account scopes the rows.
func TestIncludeReads_AccountUserBatchAdmitsAPortalUserAlongTheRelation(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	repo := repositorymock.NewMockAccountUserRepo(h.ctrl)
	h.repos.EXPECT().NewAccountUserRepo().Return(repo).AnyTimes()
	svc := newCounterpartyAccountUserSvc(h.ctrl, h.repos, domain.Mediators{ReadAccess: h.readAccess}, nil)

	h.readAccess.EXPECT().CheckReadAccess(gomock.Any(), inclBuyerID, inclSellerID).Return(apierror.NewAuthorizationError("You cannot access this account."))
	_, apiErr := svc.BatchGetAccountUsersByIDs(inclPortal(false), []string{"acus_rep"})
	requireRefused(t, apiErr, "a portal user reading account users directly")

	h.readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), inclBuyerID, inclSellerID).Return(nil)
	repo.EXPECT().GetByIDs(gomock.Any(), inclSellerID, []string{"acus_rep"}).Return(nil, nil)
	_, apiErr = svc.BatchGetAccountUsersByIDs(inclPortal(true), []string{"acus_rep"})
	require.Nil(t, apiErr)
}

// The flag never widens the account a read is scoped to: an included read without a relation to the target is still refused.
func TestIncludeReads_StillNeedTheRelationToTheTarget(t *testing.T) {
	t.Parallel()

	h := newInclHarness(t)
	h.repos.EXPECT().NewUserRepo().Return(repositorymock.NewMockUserRepo(h.ctrl)).AnyTimes()
	svc := NewUserSvc(&UserSvcConfig{Repos: h.repos, MediatorFactory: h.meds, TxManager: &stubTxManager{factory: h.repos}, S3Client: &capturingObjectStore{}})
	h.readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), inclBuyerID, inclSellerID).Return(apierror.NewAuthorizationError("You cannot access this account."))

	_, apiErr := svc.BatchGetUsersByIDs(inclPortal(true), []string{"us_rep"})
	requireRefused(t, apiErr)
}
