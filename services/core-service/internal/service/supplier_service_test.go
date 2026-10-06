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
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

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
	supplierSvc     domain.SupplierSvc
	supplierMatSvc  domain.SupplierMaterialSvc
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
	d.repos.EXPECT().NewSupplierMaterialRepo().Return(d.supplierMatRepo).AnyTimes()
	d.repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: d.idempotency}).AnyTimes()
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

func notFound() *apierror.APIError { return apierror.NewResourceNotFoundError("Resource not found.") }

// --- Supplier materials ---

func TestCreateSupplierMaterial_RefusesASupplierThatIsNotTheOwners(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	d.supplierRepo.EXPECT().Get(gomock.Any(), domain.GetSupplierParams{OwnerAccountID: supplierTestOwnerID, SupplierID: "ac_other"}).Return(nil, notFound())

	_, apiErr := d.supplierMatSvc.CreateSupplierMaterial(supplierAdminCtx(), domain.CreateSupplierMaterialParams{
		SupplierAccountID: "ac_other", MaterialID: "ml_1", SupplierPartNumber: "P-1",
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apierror.GetHTTPStatusCode(apiErr.Code))
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
	assert.Equal(t, http.StatusNotFound, apierror.GetHTTPStatusCode(apiErr.Code))
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
	assert.Equal(t, http.StatusBadRequest, apierror.GetHTTPStatusCode(apiErr.Code))
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
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.Supplier{ID: "ac_sup", BillToAddress: &domain.CustomerAddress{ID: "ad_own"}}, nil)
	d.addressRepo.EXPECT().IsInAccount(gomock.Any(), "ac_sup", "ad_foreign").Return(false, nil)

	foreign := "ad_foreign"
	_, apiErr := d.supplierSvc.UpdateSupplier(supplierAdminCtx(), domain.UpdateSupplierParams{SupplierID: "ac_sup", ShipToAddressID: &foreign})

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apierror.GetHTTPStatusCode(apiErr.Code))
	assert.Equal(t, "ship_to_address_id", apiErr.Param)
}

// The update reads the supplier back with its addresses, as it read the old one, so the audit diff holds only what changed.
func TestUpdateSupplier_ReadsBackWithAddressesAndSkipsTheCheckForAnUnchangedAddress(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	old := &domain.Supplier{ID: "ac_sup", Name: "Old", BillToAddress: &domain.CustomerAddress{ID: "ad_own"}}
	d.supplierRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(old, nil)
	d.supplierRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p domain.UpdateSupplierParams) (*domain.Supplier, *apierror.APIError) {
		assert.ElementsMatch(t, []string{"bill_to_address", "ship_to_address"}, p.Includes)
		assert.Equal(t, "ad_own", *p.BillToAddressID, "an unchanged default is written back as it was")
		return &domain.Supplier{ID: "ac_sup", Name: "New", BillToAddress: old.BillToAddress}, nil
	})

	same, name := "ad_own", "New"
	updated, apiErr := d.supplierSvc.UpdateSupplier(supplierAdminCtx(), domain.UpdateSupplierParams{SupplierID: "ac_sup", Name: &name, BillToAddressID: &same})

	require.Nil(t, apiErr)
	assert.Equal(t, "New", updated.Name)
}

func TestCreateSupplier_IdenticalShipToReusesTheBillToAddress(t *testing.T) {
	t.Parallel()
	d := newSupplierTestDeps(t)
	street := "1 Mill Rd"
	address := domain.CreateAddressParams{Name: "Dock", StreetLine1: &street, Country: "US"}
	shipTo := address

	d.supplierRepo.EXPECT().ExistsByNumber(gomock.Any(), supplierTestOwnerID, "S-1", nil).Return(false, nil)
	d.addressRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.Address{}, nil).Times(1)
	d.supplierRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, p domain.CreateSupplierParams, billTo, shipTo *string) (*domain.Supplier, *apierror.APIError) {
			require.NotNil(t, billTo)
			assert.Equal(t, billTo, shipTo, "both defaults are the one address")
			assert.ElementsMatch(t, []string{"bill_to_address", "ship_to_address"}, p.Includes)
			return &domain.Supplier{ID: "ac_new"}, nil
		})

	_, apiErr := d.supplierSvc.CreateSupplier(supplierAdminCtx(), domain.CreateSupplierParams{
		Name: "Acme", Number: "S-1", BillToAddress: &address, ShipToAddress: &shipTo,
	})
	require.Nil(t, apiErr)
}
