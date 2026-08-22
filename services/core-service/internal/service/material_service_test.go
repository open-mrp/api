package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type MaterialBulkUpsertTestSuite struct {
	suite.Suite
	materialSvc     domain.MaterialSvc
	materialRepo    *repositorymock.MockMaterialRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	ctrl            *gomock.Controller
}

func (suite *MaterialBulkUpsertTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.materialRepo = repositorymock.NewMockMaterialRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewMaterialRepo().Return(suite.materialRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.materialSvc = NewMaterialSvc(&MaterialSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *MaterialBulkUpsertTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestMaterialBulkUpsertTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MaterialBulkUpsertTestSuite))
}

func internalMaterialCtx(accountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"materials:read":   true,
				"materials:create": true,
				"materials:update": true,
				"materials:delete": true,
			},
		},
	})
}

// --- BulkUpsertMaterials validation guards (reject before idempotency / tx) ---

func (suite *MaterialBulkUpsertTestSuite) TestBulkUpsertMaterials_EmptyRejected() {
	ctx := internalMaterialCtx("ac_test123")

	result, err := suite.materialSvc.BulkUpsertMaterials(ctx, domain.BulkUpsertMaterialsParams{Materials: nil})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *MaterialBulkUpsertTestSuite) TestBulkUpsertMaterials_TooManyRejected() {
	ctx := internalMaterialCtx("ac_test123")

	materials := make([]domain.UpsertMaterialParams, 1001)
	for i := range materials {
		materials[i] = domain.UpsertMaterialParams{SKU: "SKU-" + strconv.Itoa(i), Category: domain.ObjectIdentifier{ID: "ic_001"}}
	}

	result, err := suite.materialSvc.BulkUpsertMaterials(ctx, domain.BulkUpsertMaterialsParams{Materials: materials})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *MaterialBulkUpsertTestSuite) TestBulkUpsertMaterials_DuplicateSKURejected() {
	ctx := internalMaterialCtx("ac_test123")

	result, err := suite.materialSvc.BulkUpsertMaterials(ctx, domain.BulkUpsertMaterialsParams{
		Materials: []domain.UpsertMaterialParams{
			{SKU: "SKU-DUP", Category: domain.ObjectIdentifier{ID: "ic_001"}},
			{SKU: "SKU-DUP", Category: domain.ObjectIdentifier{ID: "ic_001"}},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

// --- ExportMaterials ---

func (suite *MaterialBulkUpsertTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Materials")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *MaterialBulkUpsertTestSuite) buildExport(ctx context.Context, params domain.ExportMaterialsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.materialSvc.(*materialSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

// a material whose numbers come back at MySQL's full DECIMAL scale, as a real row does
func exportMaterialFixture() *domain.Material {
	description := "Merino yarn"
	notes := "kept dry"
	return &domain.Material{
		ID:         "ml_1",
		OrderPoint: &domain.Quantity{Value: "12.000000000000000000000000000000", UnitAbbreviation: "kg"},
		LeadTime:   &domain.Quantity{Value: "3.500000000000000000000000000000", UnitAbbreviation: "d"},
		Item: &domain.Item{
			SKU:          "YRN-1",
			Description:  &description,
			Notes:        &notes,
			CategoryName: "Yarn",
			UnitValue:    &domain.Rate{Value: "1.500000000000000000000000000000"},
			UnitCost:     &domain.Rate{Value: "0.750000000000000000000000000000"},
			Category: &domain.ItemCategory{Name: "Yarn", Properties: []domain.ItemCategoryProperty{
				{ID: "pp_fibre", Name: "Fibre"},
			}},
			Attributes: []*domain.ItemAttribute{{PropertyID: "pp_fibre", Value: "Merino"}},
		},
	}
}

func (suite *MaterialBulkUpsertTestSuite) TestExportMaterials_WritesHeadersAndRows() {
	ctx := internalMaterialCtx("ac_test123")
	suite.materialRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		Return([]*domain.Material{exportMaterialFixture()}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportMaterialsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(1), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 2)
	// The property column is derived from the data and lands after the fixed ones.
	suite.Equal([]string{
		"ID", "SKU", "Description", "Notes", "Category",
		"Order Point", "Lead Time", "Unit Price", "Unit Cost", "Fibre",
	}, rows[0])
	// No column carries the storage scale, and none names a unit: the importer
	// supplies those itself and would read a stray header as a property.
	suite.Equal([]string{
		"ml_1", "YRN-1", "Merino yarn", "kept dry", "Yarn",
		"12", "3.5", "1.5", "0.75", "Merino",
	}, rows[1])
}

// a material with no rates, quantities or properties still exports, all-blank
func (suite *MaterialBulkUpsertTestSuite) TestExportMaterials_LeavesUnsetValuesBlank() {
	ctx := internalMaterialCtx("ac_test123")
	suite.materialRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		Return([]*domain.Material{{ID: "ml_2", Item: &domain.Item{SKU: "YRN-2", CategoryName: "Yarn"}}}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportMaterialsParams{})
	suite.Require().Nil(apiErr)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 2)
	suite.Equal([]string{"ID", "SKU", "Description", "Notes", "Category", "Order Point", "Lead Time", "Unit Price", "Unit Cost"}, rows[0])
	// Trailing blanks are trimmed on read, so the row ends at the last filled cell.
	suite.Equal([]string{"ml_2", "YRN-2", "", "", "Yarn"}, rows[1])
}

// an account with nothing to export still gets a usable, header-only workbook
func (suite *MaterialBulkUpsertTestSuite) TestExportMaterials_EmptyAccountYieldsHeaderOnlyFile() {
	ctx := internalMaterialCtx("ac_test123")
	suite.materialRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(nil, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportMaterialsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(0), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 1)
	suite.Equal("ID", rows[0][0])
}

// the account comes from the caller's identity, never from the request
func (suite *MaterialBulkUpsertTestSuite) TestExportMaterials_ScopesToTheIdentitysAccount() {
	ctx := internalMaterialCtx("ac_owner")
	suite.materialRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.ExportMaterialsParams) ([]*domain.Material, *apierror.APIError) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportMaterialsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}

func (suite *MaterialBulkUpsertTestSuite) TestExportMaterials_RejectsAnIdentitylessContext() {
	export, apiErr := suite.materialSvc.ExportMaterials(context.Background(), domain.ExportMaterialsParams{})
	suite.Nil(export)
	suite.NotNil(apiErr)
}
