package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/field"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const supplierTestOwnerID = "ac_supowner"

type supplierTestDeps struct {
	repos           *factorymock.MockRepoFactory
	supplierRepo    *repositorymock.MockSupplierRepo
	addressRepo     *repositorymock.MockAddressRepo
	materialRepo    *repositorymock.MockMaterialRepo
	supplierMatRepo *repositorymock.MockSupplierMaterialRepo
	idempotency     *mediatormock.MockIdempotencyMed
	// items are the item states GetAssignability reports, by ID.
	items          map[string]domain.ItemAssignability
	supplierSvc    domain.SupplierSvc
	supplierMatSvc domain.SupplierMaterialSvc
}

func newSupplierTestDeps(t *testing.T) *supplierTestDeps {
	ctrl := gomock.NewController(t)
	d := &supplierTestDeps{
		repos:           factorymock.NewMockRepoFactory(ctrl),
		supplierRepo:    repositorymock.NewMockSupplierRepo(ctrl),
		addressRepo:     repositorymock.NewMockAddressRepo(ctrl),
		materialRepo:    repositorymock.NewMockMaterialRepo(ctrl),
		supplierMatRepo: repositorymock.NewMockSupplierMaterialRepo(ctrl),
		idempotency:     mediatormock.NewMockIdempotencyMed(ctrl),
	}
	d.repos.EXPECT().NewSupplierRepo().Return(d.supplierRepo).AnyTimes()
	d.repos.EXPECT().NewAddressRepo().Return(d.addressRepo).AnyTimes()
	d.repos.EXPECT().NewMaterialRepo().Return(d.materialRepo).AnyTimes()
	assignableItems := repositorymock.NewMockItemRepo(ctrl)
	d.items = map[string]domain.ItemAssignability{}
	assignableItems.EXPECT().GetAssignability(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, ids []string) ([]domain.ItemAssignability, *apierror.APIError) {
			var out []domain.ItemAssignability
			for _, id := range ids {
				if state, ok := d.items[id]; ok {
					out = append(out, state)
				}
			}
			return out, nil
		}).AnyTimes()
	d.repos.EXPECT().NewItemRepo().Return(assignableItems).AnyTimes()
	d.repos.EXPECT().NewSupplierMaterialRepo().Return(d.supplierMatRepo).AnyTimes()
	d.repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	mediators := factorymock.NewMockMediatorFactory(ctrl)
	site := mediatormock.NewMockSiteMed(ctrl)
	site.EXPECT().CreateDefaultSite(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: d.idempotency, Site: site}).AnyTimes()
	d.idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_sup", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).AnyTimes()
	d.idempotency.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_sup", gomock.Any()).Return(nil).AnyTimes()
	d.idempotency.EXPECT().CacheErrorResponse(gomock.Any(), "idk_sup", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()

	tx := &stubTxManager{factory: d.repos}
	d.supplierSvc = NewSupplierSvc(&SupplierSvcConfig{Repos: d.repos, MediatorFactory: mediators, TxManager: tx})
	d.supplierMatSvc = NewSupplierMaterialSvc(&SupplierMaterialSvcConfig{Repos: d.repos, MediatorFactory: mediators, TxManager: tx})
	return d
}

func supplierAdminCtx() context.Context {
	owner := supplierTestOwnerID
	admin := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: owner},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "us_sup",
			AccountID:    &owner,
			RoleType:     &admin,
		},
	})
}

func notFound() *apierror.APIError { return apierror.NewNotFoundError("Resource not found.") }

// --- Supplier materials ---

func TestCreateSupplierMaterial_RefusesASupplierThatIsNotTheOwners(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.supplierRepo.EXPECT().Get(gomock.Any(), domain.GetSupplierParams{OwnerAccountID: supplierTestOwnerID, SupplierID: "ac_other"}).Return(nil, notFound())

	_, apiErr := d.supplierMatSvc.CreateSupplierMaterial(supplierAdminCtx(), domain.CreateSupplierMaterialParams{
		SupplierAccountID: "ac_other", MaterialID: "ml_1", SupplierPartNumber: "P-1",
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code.Status())
	assert.Equal(t, "supplier_id", apiErr.Param)
}

func TestCreateSupplierMaterial_RefusesAMaterialThatIsNotTheOwners(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup"}, nil)
	d.materialRepo.EXPECT().GetByID(gomock.Any(), domain.GetMaterialParams{AccountID: supplierTestOwnerID, MaterialID: "ml_other"}).Return(nil, notFound())

	_, apiErr := d.supplierMatSvc.CreateSupplierMaterial(supplierAdminCtx(), domain.CreateSupplierMaterialParams{
		SupplierAccountID: "ac_sup", MaterialID: "ml_other", SupplierPartNumber: "P-1",
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code.Status())
	assert.Equal(t, "material_id", apiErr.Param)
}

func TestCreateSupplierMaterial_LinksTheOwnersSupplierAndMaterial(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup"}, nil)
	d.materialRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&domain.Material{ID: "ml_1"}, nil)
	d.supplierMatRepo.EXPECT().ExistsByMaterialAndSupplier(gomock.Any(), supplierTestOwnerID, "ml_1", "ac_sup").Return(false, nil)
	d.supplierMatRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.SupplierMaterial{ID: "spml_1"}, nil)

	created, apiErr := d.supplierMatSvc.CreateSupplierMaterial(supplierAdminCtx(), domain.CreateSupplierMaterialParams{
		SupplierAccountID: "ac_sup", MaterialID: "ml_1", SupplierPartNumber: "P-1",
	})

	require.Nil(t, apiErr)
	assert.Equal(t, "spml_1", created.ID)
}

// The gateway counts characters; four-byte characters within that count still overflow the column.
func TestSupplierMaterialDescription_RefusesMoreBytesThanTheColumnHolds(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	tooLong := strings.Repeat("🧶", 20000)

	_, apiErr := d.supplierMatSvc.CreateSupplierMaterial(supplierAdminCtx(), domain.CreateSupplierMaterialParams{
		SupplierAccountID: "ac_sup", MaterialID: "ml_1", SupplierPartNumber: "P-1", SupplierDescription: &tooLong,
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.Code.Status())
	assert.Equal(t, "supplier_description", apiErr.Param)

	_, apiErr = d.supplierMatSvc.UpdateSupplierMaterial(supplierAdminCtx(), domain.UpdateSupplierMaterialParams{
		SupplierAccountID: "ac_sup", MaterialID: "ml_1", SupplierDescription: &tooLong, UpdateDescription: true,
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, "supplier_description", apiErr.Param)
}

// --- Suppliers ---

func TestUpdateSupplier_RefusesAnAddressThatIsNotTheSuppliers(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	own := "ad_own"
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", BillToAddressID: &own}, nil)
	d.addressRepo.EXPECT().IsInAccount(gomock.Any(), "ac_sup", "ad_foreign").Return(false, nil)

	_, apiErr := d.supplierSvc.UpdateSupplier(supplierAdminCtx(), domain.UpdateSupplierParams{SupplierID: "ac_sup", ShipToAddressID: field.Set("ad_foreign")})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code.Status())
	assert.Equal(t, "defaults.ship_to_address_id", apiErr.Param)
}

// An update writes every column, so each one it was not sent goes back as the supplier held it.
func TestUpdateSupplier_WritesBackWhatItWasNotSentAndSkipsTheCheckForAnUnchangedAddress(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	own, instructions := "ad_own", "Call ahead"
	old := &domain.Supplier{ID: "ac_sup", Name: "Old", Number: "S-1", ArchiveStatus: constants.ArchiveStatusActive, BillToAddressID: &own, Instructions: &instructions}
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(old, nil).Times(2)
	d.supplierRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, w domain.SupplierRelationWrite, _ []string) (*domain.Supplier, *apierror.APIError) {
		assert.Equal(t, "New", w.Name)
		assert.Equal(t, "S-1", w.Number)
		assert.True(t, w.IsActive)
		assert.Equal(t, &own, w.BillToAddressID, "an unchanged default is written back as it was")
		assert.Nil(t, w.Instructions, "a cleared field is written as null")
		return &domain.Supplier{ID: "ac_sup", Name: "New"}, nil
	})

	name := "New"
	_, apiErr := d.supplierSvc.UpdateSupplier(supplierAdminCtx(), domain.UpdateSupplierParams{
		SupplierID: "ac_sup", Name: &name, BillToAddressID: field.Set("ad_own"), Instructions: field.Clear[string](),
	})

	require.Nil(t, apiErr)
}

func TestDeleteSupplier_RefusesASupplierStillInUse(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup"}, nil)
	d.supplierRepo.EXPECT().InUseBy(gomock.Any(), supplierTestOwnerID, "ac_sup").Return([]string{"purchase orders", "supplier items"}, nil)
	d.supplierRepo.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, apiErr := d.supplierSvc.DeleteSupplier(supplierAdminCtx(), domain.DeleteSupplierParams{SupplierID: "ac_sup"})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code.Status())
	assert.Equal(t, "resource_in_use", string(apiErr.Code))
}

func TestCreateSupplier_IdenticalShipToReusesTheBillToAddress(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	street := "1 Mill Rd"
	address := domain.CreateAddressParams{Name: "Dock", StreetLine1: &street, Country: "US"}
	shipTo := address

	d.supplierRepo.EXPECT().LockNumbers(gomock.Any(), supplierTestOwnerID).Return(nil)
	d.supplierRepo.EXPECT().ExistsByNumber(gomock.Any(), supplierTestOwnerID, "S-1", nil).Return(false, nil)
	d.addressRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.Address{}, nil).Times(1)
	d.supplierRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, _ domain.CreateSupplierParams, billTo, shipTo *string) (*domain.Supplier, *apierror.APIError) {
			require.NotNil(t, billTo)
			assert.Equal(t, billTo, shipTo, "both defaults are the one address")
			return &domain.Supplier{ID: "ac_new"}, nil
		})
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_new"}, nil)

	_, apiErr := d.supplierSvc.CreateSupplier(supplierAdminCtx(), domain.CreateSupplierParams{
		Name: "Acme", Number: "S-1", BillToAddress: &address, ShipToAddress: &shipTo,
	})
	require.Nil(t, apiErr)
}

// No unique index guards the number, so two concurrent creates both pass a check made without the owner's lock.
func TestCreateSupplier_ChecksTheNumberOnlyUnderTheOwnersLock(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	gomock.InOrder(
		d.supplierRepo.EXPECT().LockNumbers(gomock.Any(), supplierTestOwnerID).Return(nil),
		d.supplierRepo.EXPECT().ExistsByNumber(gomock.Any(), supplierTestOwnerID, "S-1", nil).Return(true, nil),
	)

	_, apiErr := d.supplierSvc.CreateSupplier(supplierAdminCtx(), domain.CreateSupplierParams{Name: "Acme", Number: "S-1"})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code.Status())
	assert.Equal(t, "number", apiErr.Param)
}

// The lock precedes every read: a read before it would pin the snapshot the number check sees.
func TestUpdateSupplier_LocksTheOwnersNumbersBeforeItsFirstRead(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	self := "ac_sup"
	gomock.InOrder(
		d.supplierRepo.EXPECT().LockNumbers(gomock.Any(), supplierTestOwnerID).Return(nil),
		d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: self}, nil),
		d.supplierRepo.EXPECT().ExistsByNumber(gomock.Any(), supplierTestOwnerID, "S-2", &self).Return(true, nil),
	)

	number := "S-2"
	_, apiErr := d.supplierSvc.UpdateSupplier(supplierAdminCtx(), domain.UpdateSupplierParams{SupplierID: self, Number: &number})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code.Status())
	assert.Equal(t, "number", apiErr.Param)
}

func TestUpdateSupplier_LeavesTheNumbersUnlockedWhenTheNumberIsNotSent(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.supplierRepo.EXPECT().LockNumbers(gomock.Any(), gomock.Any()).Times(0)
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", Name: "Old"}, nil).Times(2)
	d.supplierRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", Name: "New"}, nil)

	name := "New"
	_, apiErr := d.supplierSvc.UpdateSupplier(supplierAdminCtx(), domain.UpdateSupplierParams{SupplierID: "ac_sup", Name: &name})
	require.Nil(t, apiErr)
}

func supplierRoleCtx(permissions ...string) context.Context {
	owner := supplierTestOwnerID
	role := string(constants.RoleTypeCustom)
	granted := map[string]bool{}
	for _, p := range permissions {
		granted[p] = true
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: owner},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "us_sup",
			AccountID:    &owner,
			RoleType:     &role,
			Permissions:  granted,
		},
	})
}

func TestSupplierSvc_BatchGetServesWhoeverMayReadADocumentNamingTheSupplier(t *testing.T) {
	t.Parallel()

	for _, perm := range []string{"suppliers:read", "purchase_orders:read", "receiving_orders:read"} {
		t.Run(perm, func(t *testing.T) {
			t.Parallel()
			d := newSupplierTestDeps(t)
			want := []*domain.Supplier{{ID: "ac_sup1"}}
			d.supplierRepo.EXPECT().GetByIDs(gomock.Any(), supplierTestOwnerID, []string{"ac_sup1"}, nil).Return(want, nil)

			got, apiErr := d.supplierSvc.BatchGetSuppliersByIDs(supplierRoleCtx(perm), []string{"ac_sup1"}, nil)
			require.Nil(t, apiErr)
			assert.Equal(t, want, got)
		})
	}

	t.Run("a role that reads none of them", func(t *testing.T) {
		t.Parallel()
		d := newSupplierTestDeps(t)

		_, apiErr := d.supplierSvc.BatchGetSuppliersByIDs(supplierRoleCtx("customers:read"), []string{"ac_sup1"}, nil)
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusForbidden, apiErr.Code.Status())
	})
}

// --- Supplier items ---

func TestCreateSupplierItem_RefusesAnArchivedSupplier(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", ArchiveStatus: constants.ArchiveStatusArchived}, nil)

	_, apiErr := d.supplierMatSvc.CreateSupplierItem(supplierAdminCtx(), domain.CreateSupplierItemParams{SupplierAccountID: "ac_sup", ItemID: "it_1", SupplierSKU: "S"})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.Code.Status())
	assert.Equal(t, "supplier_id", apiErr.Param)
}

func TestCreateSupplierItem_ChecksTheItem(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		state      *domain.ItemAssignability
		hasSection bool
		status     int
	}{
		"not the owner's":     {status: http.StatusNotFound},
		"archived":            {state: &domain.ItemAssignability{ItemID: "it_1"}, status: http.StatusUnprocessableEntity},
		"no material section": {state: &domain.ItemAssignability{ItemID: "it_1", IsAssignable: true}, status: http.StatusUnprocessableEntity},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := newSupplierTestDeps(t)
			if tc.state != nil {
				d.items["it_1"] = *tc.state
			}
			d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", ArchiveStatus: constants.ArchiveStatusActive}, nil)
			d.materialRepo.EXPECT().GetByItemID(gomock.Any(), supplierTestOwnerID, "it_1").Return(nil, notFound()).AnyTimes()

			_, apiErr := d.supplierMatSvc.CreateSupplierItem(supplierAdminCtx(), domain.CreateSupplierItemParams{SupplierAccountID: "ac_sup", ItemID: "it_1", SupplierSKU: "S"})

			require.NotNil(t, apiErr)
			assert.Equal(t, tc.status, apiErr.Code.Status())
			assert.Equal(t, "item_id", apiErr.Param)
		})
	}
}

func TestCreateSupplierItem_OnePerItemAndSupplier(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.items["it_1"] = domain.ItemAssignability{ItemID: "it_1", IsAssignable: true}
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", ArchiveStatus: constants.ArchiveStatusActive}, nil)
	d.materialRepo.EXPECT().GetByItemID(gomock.Any(), supplierTestOwnerID, "it_1").Return(&domain.Material{ID: "ml_1"}, nil)
	d.supplierMatRepo.EXPECT().ExistsByMaterialAndSupplier(gomock.Any(), supplierTestOwnerID, "ml_1", "ac_sup").Return(true, nil)

	_, apiErr := d.supplierMatSvc.CreateSupplierItem(supplierAdminCtx(), domain.CreateSupplierItemParams{SupplierAccountID: "ac_sup", ItemID: "it_1", SupplierSKU: "S"})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Code.Status())
}

func TestCreateSupplierItem_StoresTheLinkUnderTheItemsMaterial(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.items["it_1"] = domain.ItemAssignability{ItemID: "it_1", IsAssignable: true}
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", ArchiveStatus: constants.ArchiveStatusActive}, nil)
	d.materialRepo.EXPECT().GetByItemID(gomock.Any(), supplierTestOwnerID, "it_1").Return(&domain.Material{ID: "ml_1"}, nil)
	d.supplierMatRepo.EXPECT().ExistsByMaterialAndSupplier(gomock.Any(), supplierTestOwnerID, "ml_1", "ac_sup").Return(false, nil)
	d.supplierMatRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, id string, p domain.CreateSupplierMaterialParams) (*domain.SupplierMaterial, *apierror.APIError) {
			assert.Equal(t, "ml_1", p.MaterialID)
			assert.False(t, p.IsActive, "an archived supplier item is stored inactive")
			return &domain.SupplierMaterial{ID: id}, nil
		})
	d.supplierMatRepo.EXPECT().GetItem(gomock.Any(), supplierTestOwnerID, "ac_sup", gomock.Any()).Return(&domain.SupplierItem{ID: "suit_1", ItemID: "it_1", SupplierAccountID: "ac_sup"}, nil)

	created, apiErr := d.supplierMatSvc.CreateSupplierItem(supplierAdminCtx(), domain.CreateSupplierItemParams{
		SupplierAccountID: "ac_sup", ItemID: "it_1", SupplierSKU: "S", ArchiveStatus: constants.ArchiveStatusArchived,
	})

	require.Nil(t, apiErr)
	assert.Equal(t, "it_1", created.ItemID)
}

func TestUpdateSupplierItem_ChangesOnlyWhatWasSent(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	desc := "old"
	old := &domain.SupplierItem{ID: "suit_1", SupplierSKU: "A", SupplierDescription: &desc, Status: constants.ArchiveStatusActive}
	d.supplierMatRepo.EXPECT().GetItem(gomock.Any(), supplierTestOwnerID, "ac_sup", "suit_1").Return(old, nil).Times(2)
	d.supplierMatRepo.EXPECT().UpdateItem(gomock.Any(), supplierTestOwnerID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, item *domain.SupplierItem) *apierror.APIError {
			assert.Equal(t, "A", item.SupplierSKU)
			assert.Nil(t, item.SupplierDescription)
			assert.Equal(t, constants.ArchiveStatusArchived, item.Status)
			return nil
		})

	archived := constants.ArchiveStatusArchived
	_, apiErr := d.supplierMatSvc.UpdateSupplierItem(supplierAdminCtx(), domain.UpdateSupplierItemParams{
		SupplierAccountID: "ac_sup", SupplierItemID: "suit_1", SupplierDescription: field.Clear[string](), ArchiveStatus: &archived,
	})
	require.Nil(t, apiErr)
}
