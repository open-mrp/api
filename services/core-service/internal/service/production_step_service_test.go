package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"github.com/xuri/excelize/v2"
	"go.uber.org/mock/gomock"
)

type ProductionStepBulkUpsertTestSuite struct {
	suite.Suite
	productionStepSvc       domain.ProductionStepSvc
	productionStepRepo      *repositorymock.MockProductionStepRepo
	productionStepQueryRepo *repositorymock.MockProductionStepQueryRepo
	repoFactory             *factorymock.MockRepoFactory
	mediatorFactory         *factorymock.MockMediatorFactory
	idempotencyMed          *mediatormock.MockIdempotencyMed
	ctrl                    *gomock.Controller
}

func (suite *ProductionStepBulkUpsertTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.productionStepRepo = repositorymock.NewMockProductionStepRepo(suite.ctrl)
	suite.productionStepQueryRepo = repositorymock.NewMockProductionStepQueryRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewProductionStepRepo().Return(suite.productionStepRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewProductionStepQueryRepo().Return(suite.productionStepQueryRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	// The bulk upsert runs through the async engine, which raises a job. These tests
	// reject at the accept phase before any job is raised, so a real factory that is
	// never exercised is enough; the engine's job interactions are covered in
	// async_bulk_upsert_test.go.
	suite.productionStepSvc = NewProductionStepSvc(&ProductionStepSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *ProductionStepBulkUpsertTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestProductionStepBulkUpsertTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ProductionStepBulkUpsertTestSuite))
}

// productionStepCtx builds an identity context with an explicit role and permission
// map. Note admins bypass the permission map entirely — permission-denial tests must
// use a non-admin role.
func productionStepCtx(accountID string, relationType types.IdentityRelationType, roleType constants.RoleType, permissions map[string]bool) context.Context {
	roleCode := string(roleType)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: relationType,
			ID:           "usr_test123",
			AccountID:    &accountID,
			RoleType:     &roleCode,
			Permissions:  permissions,
		},
	})
}

func internalProductionStepCtx(accountID string) context.Context {
	return productionStepCtx(accountID, types.IdentityRelationTypeInternal, constants.RoleTypeAdmin, map[string]bool{
		"production_steps:read":   true,
		"production_steps:create": true,
		"production_steps:update": true,
		"production_steps:delete": true,
	})
}

func validUpsertProductionStepParams(name string) domain.UpsertProductionStepParams {
	return domain.UpsertProductionStepParams{
		Name:         name,
		LaborRate:    domain.UpsertRateParams{Value: "25.00", NumeratorUnit: domain.UnitIdentifier{Abbreviation: "$"}, DenominatorUnit: domain.UnitIdentifier{Abbreviation: "hr"}},
		LaborTime:    domain.UpsertRateParams{Value: "1.5", NumeratorUnit: domain.UnitIdentifier{Abbreviation: "hr"}, DenominatorUnit: domain.UnitIdentifier{Abbreviation: "piece"}},
		OverheadRate: domain.UpsertRateParams{Value: "15.00", NumeratorUnit: domain.UnitIdentifier{Abbreviation: "$"}, DenominatorUnit: domain.UnitIdentifier{Abbreviation: "hr"}},
		Production:   domain.UpsertProductionParams{Item: domain.ItemIdentifier{SKU: "SKU-1"}, QuantityValue: "100", QuantityUnit: domain.UnitIdentifier{Abbreviation: "piece"}},
	}
}

// --- BulkUpsertProductionSteps identity guards (reject before any validation) ---

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_MissingIdentityRejected() {
	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(context.Background(), domain.BulkUpsertProductionStepsParams{
		ProductionSteps: []domain.UpsertProductionStepParams{validUpsertProductionStepParams("Mixing")},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_ExternalActorRejected() {
	ctx := productionStepCtx("ac_test123", types.IdentityRelationTypeCustomer, constants.RoleTypeCustom, map[string]bool{
		"production_steps:create": true,
		"production_steps:update": true,
	})

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{
		ProductionSteps: []domain.UpsertProductionStepParams{validUpsertProductionStepParams("Mixing")},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_MissingCreatePermissionRejected() {
	ctx := productionStepCtx("ac_test123", types.IdentityRelationTypeInternal, constants.RoleTypeCustom, map[string]bool{
		"production_steps:update": true,
	})

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{
		ProductionSteps: []domain.UpsertProductionStepParams{validUpsertProductionStepParams("Mixing")},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_MissingUpdatePermissionRejected() {
	ctx := productionStepCtx("ac_test123", types.IdentityRelationTypeInternal, constants.RoleTypeCustom, map[string]bool{
		"production_steps:create": true,
	})

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{
		ProductionSteps: []domain.UpsertProductionStepParams{validUpsertProductionStepParams("Mixing")},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

// --- BulkUpsertProductionSteps validation guards (reject before idempotency / tx) ---

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_EmptyRejected() {
	ctx := internalProductionStepCtx("ac_test123")

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{ProductionSteps: nil})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_TooManyRejected() {
	ctx := internalProductionStepCtx("ac_test123")

	steps := make([]domain.UpsertProductionStepParams, 1001)
	for i := range steps {
		steps[i] = validUpsertProductionStepParams("Step-" + strconv.Itoa(i))
	}

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{ProductionSteps: steps})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_DuplicateNameRejected() {
	ctx := internalProductionStepCtx("ac_test123")

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{
		ProductionSteps: []domain.UpsertProductionStepParams{
			validUpsertProductionStepParams("Mixing"),
			validUpsertProductionStepParams("mIxInG"), // duplicate differing only by casing
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_DuplicateConsumptionSKURejected() {
	ctx := internalProductionStepCtx("ac_test123")

	step := validUpsertProductionStepParams("Mixing")
	step.Consumptions = []domain.UpsertStepConsumptionParams{
		{Item: domain.ItemIdentifier{SKU: "SKU-C"}, QuantityValue: "1", QuantityUnit: domain.UnitIdentifier{Abbreviation: "piece"}},
		{Item: domain.ItemIdentifier{SKU: "sku-c"}, QuantityValue: "2", QuantityUnit: domain.UnitIdentifier{Abbreviation: "piece"}}, // duplicate differing only by casing
	}

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{
		ProductionSteps: []domain.UpsertProductionStepParams{step},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ProductionStepBulkUpsertTestSuite) TestBulkUpsertProductionSteps_MissingProductionRejected() {
	ctx := internalProductionStepCtx("ac_test123")

	step := validUpsertProductionStepParams("Mixing")
	step.Production = domain.UpsertProductionParams{}

	result, err := suite.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{
		ProductionSteps: []domain.UpsertProductionStepParams{step},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// reopens the export so assertions read what a spreadsheet would
func (suite *ProductionStepBulkUpsertTestSuite) exportedFile(export *domain.Export) *excelize.File {
	return openExportFile(suite.T(), export)
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *ProductionStepBulkUpsertTestSuite) buildExport(ctx context.Context, params domain.ExportProductionStepsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.productionStepSvc.(*productionStepSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

func ptr(v string) *string { return &v }

// the step's own columns sit on its first consumption's row and are blank on the rest
func (suite *ProductionStepBulkUpsertTestSuite) TestExportProductionSteps_ListsConsumptionsOnePerRow() {
	ctx := internalProductionStepCtx("ac_test123")
	suite.productionStepQueryRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.ProductionStepExport{
		{
			ID:                    "pstp_1",
			Name:                  "Mixing",
			DepartmentName:        ptr("Blending"),
			ScanningStationName:   ptr("Mix Station"),
			LaborRate:             ptr("25"),
			LaborRateCurrencyUnit: ptr("$"),
			LaborRateTimeUnit:     ptr("hr"),
			// 0 defaults must blank so a re-import keeps them.
			Allowances:       "0",
			LevelingFactor:   "0.5",
			ProducedItemSKU:  "SKU-OUT",
			ProducedQuantity: "1",
			ProducedUnit:     "kg",
			Consumptions: []domain.ProductionStepExportConsumption{
				{ItemSKU: "SKU-A", Quantity: "2", Unit: "kg", WasteQuantity: ptr("0.1"), WasteUnit: ptr("kg"), Instructions: ptr("Add slowly")},
				{ItemSKU: "SKU-B", Quantity: "3", Unit: "kg", WasteQuantity: ptr("0")},
			},
		},
		{ID: "pstp_2", Name: "Packing", Allowances: "0", LevelingFactor: "0", ProducedItemSKU: "SKU-P", ProducedQuantity: "1", ProducedUnit: "ea"},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportProductionStepsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount, "two steps, even though they span three sheet rows")

	f := suite.exportedFile(export)
	rows, err := f.GetRows("Production Steps")
	suite.Require().NoError(err)
	suite.Require().Len(rows, 4)
	suite.Require().Len(rows[0], 24, "all 24 columns are present")
	suite.Equal("Name", rows[0][0])
	suite.Equal("Notes", rows[0][23])

	suite.Equal("Mixing", rows[1][0])
	suite.Equal("Blending", rows[1][1])
	suite.Equal("", rows[1][12], "an allowance of 0 blanks so re-import keeps the default")
	suite.Equal("0.5", rows[1][13])
	suite.Equal("SKU-A", rows[1][17])
	suite.Equal("Add slowly", rows[1][22])

	// The continuation row carries only the consumption; the step's columns are blank.
	suite.Equal("", rows[2][0], "the step name is blank on a continuation row")
	suite.Equal("SKU-B", rows[2][17])
	suite.Equal("kg", rows[2][19])
	// A waste quantity of 0 blanks too, and trailing blanks are trimmed on read,
	// so the row stops at the consumed unit.
	suite.Len(rows[2], 20)

	// A step with no consumptions still gets a row.
	suite.Equal("Packing", rows[3][0])
}

// the 24 header notes are the guidance a reader needs to edit and re-upload
func (suite *ProductionStepBulkUpsertTestSuite) TestExportProductionSteps_CarriesEveryHeaderNote() {
	ctx := internalProductionStepCtx("ac_test123")
	suite.productionStepQueryRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(nil, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportProductionStepsParams{})
	suite.Require().Nil(apiErr)

	comments, err := suite.exportedFile(export).GetComments("Production Steps")
	suite.Require().NoError(err)
	suite.Len(comments, 24, "every column carries its guidance")
}

func (suite *ProductionStepBulkUpsertTestSuite) TestExportProductionSteps_ScopesToTheIdentitysAccount() {
	ctx := internalProductionStepCtx("ac_owner")
	suite.productionStepQueryRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportProductionStepsParams) ([]*domain.ProductionStepExport, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportProductionStepsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}
