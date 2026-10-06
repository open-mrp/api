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
	"github.com/open-mrp/api/shared/field"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func inlineName(name string) *domain.InlineAddressParams {
	return &domain.InlineAddressParams{Name: &name, Country: new("US")}
}

func TestSaveInlineAddresses_IdenticalAddressesAreSavedOnce(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	med := mediatormock.NewMockAddressMed(ctrl)
	med.EXPECT().Save(gomock.Any(), "ac_supplier", *inlineName("Dock"), "bill_to_address").Return(&domain.Address{ID: "ad_dock"}, nil).Times(1)

	billID, shipID, apiErr := saveInlineAddresses(context.Background(), med, "ac_supplier", inlineName("Dock"), inlineName("Dock"), "bill_to_address", "ship_to_address")

	require.Nil(t, apiErr)
	assert.Equal(t, "ad_dock", billID)
	assert.Equal(t, "ad_dock", shipID, "one address entered for both sides is shared")
}

func TestSaveInlineAddresses_DifferentAddressesAreSavedApart(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	med := mediatormock.NewMockAddressMed(ctrl)
	med.EXPECT().Save(gomock.Any(), "ac_buyer", *inlineName("Office"), "billing_address").Return(&domain.Address{ID: "ad_office"}, nil)
	med.EXPECT().Save(gomock.Any(), "ac_buyer", *inlineName("Dock"), "shipping_address").Return(&domain.Address{ID: "ad_dock"}, nil)

	billID, shipID, apiErr := saveInlineAddresses(context.Background(), med, "ac_buyer", inlineName("Office"), inlineName("Dock"), "billing_address", "shipping_address")

	require.Nil(t, apiErr)
	assert.Equal(t, "ad_office", billID)
	assert.Equal(t, "ad_dock", shipID)
}

func TestSaveInlineAddresses_OneAddressEditedTwoWaysIsRefused(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	med := mediatormock.NewMockAddressMed(ctrl)
	bill, ship := inlineName("One"), inlineName("Two")
	bill.ID, ship.ID = new("ad_dock"), new("ad_dock")

	_, _, apiErr := saveInlineAddresses(context.Background(), med, "ac_buyer", bill, ship, "bill_to_address", "ship_to_address")

	require.NotNil(t, apiErr)
	assert.Equal(t, "ship_to_address.id", apiErr.Param)
}

func TestCheckPurchaseOrderAddressChoices(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		params domain.CreatePurchaseOrderParams
		code   apierror.ErrorCode
		param  string
	}{
		{"ids", domain.CreatePurchaseOrderParams{BillToAddressID: new("ad_a"), ShipToAddressID: new("ad_b")}, "", ""},
		{"inline", domain.CreatePurchaseOrderParams{BillToAddress: inlineName("Office"), ShipToAddress: inlineName("Dock")}, "", ""},
		{"flat fields", domain.CreatePurchaseOrderParams{BillToName: new("Office"), BillToCountry: new("US"), ShipToName: new("Dock"), ShipToCountry: new("US")}, "", ""},
		{"an id with partial flat fields, which it overrides", domain.CreatePurchaseOrderParams{BillToAddressID: new("ad_a"), BillToState: new("OH"), ShipToAddress: inlineName("Dock")}, "", ""},
		{"id and inline", domain.CreatePurchaseOrderParams{BillToAddressID: new("ad_dock"), BillToAddress: inlineName("Dock"), ShipToAddressID: new("ad_b")}, apierror.ErrorCodeValidationFailed, "bill_to_address"},
		{"flat fields and inline", domain.CreatePurchaseOrderParams{BillToAddressID: new("ad_a"), ShipToName: new("Flat"), ShipToAddress: inlineName("Dock")}, apierror.ErrorCodeValidationFailed, "ship_to_address"},
		{"a conflict is reported before a missing side", domain.CreatePurchaseOrderParams{ShipToName: new("Flat"), ShipToAddress: inlineName("Dock")}, apierror.ErrorCodeValidationFailed, "ship_to_address"},
		{"no bill-to", domain.CreatePurchaseOrderParams{ShipToAddressID: new("ad_b")}, apierror.ErrorCodeMissingField, "bill_to_address_id"},
		{"no ship-to", domain.CreatePurchaseOrderParams{BillToAddress: inlineName("Office")}, apierror.ErrorCodeMissingField, "ship_to_address_id"},
		{"no address at all", domain.CreatePurchaseOrderParams{}, apierror.ErrorCodeMissingField, "bill_to_address_id"},
		{"flat fields without a country", domain.CreatePurchaseOrderParams{BillToName: new("Office"), BillToLocality: new("Columbus"), ShipToAddressID: new("ad_b")}, apierror.ErrorCodeMissingField, "bill_to_country"},
		{"flat fields without a name", domain.CreatePurchaseOrderParams{BillToAddressID: new("ad_a"), ShipToStreetLine1: new("1 Dock Rd"), ShipToCountry: new("US")}, apierror.ErrorCodeMissingField, "ship_to_name"},
		{"a blank flat name", domain.CreatePurchaseOrderParams{BillToName: new("  "), BillToCountry: new("US"), ShipToAddressID: new("ad_b")}, apierror.ErrorCodeMissingField, "bill_to_name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := checkPurchaseOrderAddressChoices(tc.params)
			if tc.param == "" {
				assert.Nil(t, apiErr)
				return
			}
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.code, apiErr.Code)
			assert.Equal(t, tc.param, apiErr.Param)
		})
	}
}

func TestCheckSalesOrderCreateAddressChoices(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		params domain.CreateSalesOrderParams
		code   apierror.ErrorCode
		param  string
	}{
		{"ids", domain.CreateSalesOrderParams{BillToAddressID: "ad_a", ShipToAddressID: "ad_b"}, "", ""},
		{"inline", domain.CreateSalesOrderParams{BillToAddress: inlineName("A"), ShipToAddress: inlineName("B")}, "", ""},
		{"neither for the bill-to", domain.CreateSalesOrderParams{ShipToAddressID: "ad_b"}, apierror.ErrorCodeMissingField, "bill_to_address_id"},
		{"neither for the ship-to", domain.CreateSalesOrderParams{BillToAddress: inlineName("A")}, apierror.ErrorCodeMissingField, "ship_to_address_id"},
		{"both for the ship-to", domain.CreateSalesOrderParams{BillToAddressID: "ad_a", ShipToAddressID: "ad_b", ShipToAddress: inlineName("B")}, apierror.ErrorCodeValidationFailed, "ship_to_address"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := checkSalesOrderCreateAddressChoices(tc.params)
			if tc.param == "" {
				assert.Nil(t, apiErr)
				return
			}
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.code, apiErr.Code)
			assert.Equal(t, tc.param, apiErr.Param)
		})
	}
}

func internalCtxWithPerms(accountID string, perms ...types.Permission) context.Context {
	granted := make(map[string]bool, len(perms))
	for _, p := range perms {
		granted[p.String()] = true
	}
	roleType := "member"
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			RoleType:     &roleType,
			Permissions:  granted,
		},
	})
}

func startedIdempotency(ctrl *gomock.Controller) *mediatormock.MockIdempotencyMed {
	med := mediatormock.NewMockIdempotencyMed(ctrl)
	med.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_test", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	med.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).Return(nil).AnyTimes()
	med.EXPECT().CacheErrorResponse(gomock.Any(), "idk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()
	return med
}

// The order's addresses are saved to its supplier under purchase_orders:update alone, and the order is pointed at them.
func TestPurchaseOrderSvc_UpdateSavesInlineAddressesToTheSupplier(t *testing.T) {
	t.Parallel()
	const accountID, supplierID, orderID = "ac_owner", "ac_supplier", "or_po"
	ctrl := gomock.NewController(t)

	orders := repositorymock.NewMockPurchaseOrderRepo(ctrl)
	old := &domain.PurchaseOrder{ID: orderID, BillingAddressID: "ad_old", ShippingAddressID: "ad_old", SellerAccountID: supplierID}
	orders.EXPECT().Get(gomock.Any(), accountID, orderID).Return(old, nil).Times(2)
	var written domain.UpdatePurchaseOrderParams
	orders.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdatePurchaseOrderParams) (*domain.PurchaseOrder, *apierror.APIError) {
			written = params
			return old, nil
		})
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewPurchaseOrderRepo().Return(orders).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	addresses := mediatormock.NewMockAddressMed(ctrl)
	billing := inlineName("Office")
	billing.ID = new("ad_old")
	billing.Phone = field.Clear[string]()
	addresses.EXPECT().Save(gomock.Any(), supplierID, *billing, "billing_address").Return(&domain.Address{ID: "ad_old"}, nil)
	addresses.EXPECT().Save(gomock.Any(), supplierID, *inlineName("Dock"), "shipping_address").Return(&domain.Address{ID: "ad_new"}, nil)

	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: startedIdempotency(ctrl), Address: addresses}).AnyTimes()

	svc := NewPurchaseOrderSvc(&PurchaseOrderSvcConfig{Repos: repos, MediatorFactory: mediators, TxManager: &stubTxManager{factory: repos}})
	ctx := internalCtxWithPerms(accountID, types.Permission{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate})
	_, apiErr := svc.UpdatePurchaseOrder(ctx, domain.UpdatePurchaseOrderParams{
		PurchaseOrderID: orderID,
		BillingAddress:  billing,
		ShippingAddress: inlineName("Dock"),
	})

	require.Nil(t, apiErr)
	require.NotNil(t, written.BillingAddressID)
	require.NotNil(t, written.ShippingAddressID)
	assert.Equal(t, "ad_old", *written.BillingAddressID)
	assert.Equal(t, "ad_new", *written.ShippingAddressID)
}

func TestPurchaseOrderSvc_UpdateRefusesAnIDAndAnInlineAddress(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	svc := NewPurchaseOrderSvc(&PurchaseOrderSvcConfig{Repos: factorymock.NewMockRepoFactory(ctrl), MediatorFactory: factorymock.NewMockMediatorFactory(ctrl), TxManager: &stubTxManager{}})
	ctx := internalCtxWithPerms("ac_owner", types.Permission{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate})

	_, apiErr := svc.UpdatePurchaseOrder(ctx, domain.UpdatePurchaseOrderParams{
		PurchaseOrderID:   "or_po",
		ShippingAddressID: new("ad_dock"),
		ShippingAddress:   inlineName("Dock"),
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, "shipping_address", apiErr.Param)
}

// The account's default addresses are saved to the account itself under self:update alone, and become its defaults.
func TestAccountSvc_UpdateSavesInlineDefaultsToTheAccount(t *testing.T) {
	t.Parallel()
	const accountID = "ac_self"
	ctrl := gomock.NewController(t)

	accounts := repositorymock.NewMockAccountRepo(ctrl)
	accounts.EXPECT().GetByID(gomock.Any(), accountID).Return(&domain.Account{ID: accountID}, nil).Times(2)
	accounts.EXPECT().HasAddress(gomock.Any(), accountID, "ad_hq").Return(true, nil)
	accounts.EXPECT().HasAddress(gomock.Any(), accountID, "ad_dock").Return(true, nil)
	accounts.EXPECT().SetDefaultAddresses(gomock.Any(), accountID, new("ad_hq"), new("ad_dock")).Return(nil)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAccountRepo().Return(accounts).AnyTimes()
	repos.EXPECT().NewAccountUserRepo().Return(nil).AnyTimes()
	repos.EXPECT().NewAccountRelationRepo().Return(nil).AnyTimes()
	repos.EXPECT().NewRolePermissionRepo().Return(nil).AnyTimes()
	repos.EXPECT().NewRoleRepo().Return(nil).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	addresses := mediatormock.NewMockAddressMed(ctrl)
	billing := inlineName("HQ")
	billing.ID = new("ad_hq")
	addresses.EXPECT().Save(gomock.Any(), accountID, *billing, "default_billing_address").Return(&domain.Address{ID: "ad_hq"}, nil)
	addresses.EXPECT().Save(gomock.Any(), accountID, *inlineName("Dock"), "default_shipping_address").Return(&domain.Address{ID: "ad_dock"}, nil)

	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: startedIdempotency(ctrl), Address: addresses}).AnyTimes()

	svc := &accountSvcImpl{accountRepo: accounts, repos: repos, mediatorFactory: mediators, txManager: &stubTxManager{factory: repos}}
	_, apiErr := svc.UpdateAccount(accountUpdateCtx(accountID), domain.UpdateAccountParams{
		AccountID:              accountID,
		DefaultBillingAddress:  billing,
		DefaultShippingAddress: inlineName("Dock"),
	})

	require.Nil(t, apiErr)
}

func TestAccountSvc_UpdateRefusesAnIDAndAnInlineDefault(t *testing.T) {
	t.Parallel()
	const accountID = "ac_self"
	svc := &accountSvcImpl{}

	_, apiErr := svc.UpdateAccount(accountUpdateCtx(accountID), domain.UpdateAccountParams{
		AccountID:               accountID,
		DefaultBillingAddressID: new("ad_hq"),
		DefaultBillingAddress:   inlineName("HQ"),
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, "default_billing_address", apiErr.Param)
}
