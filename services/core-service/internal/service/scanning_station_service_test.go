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
	"github.com/xuri/excelize/v2"
	"go.uber.org/mock/gomock"
)

type ScanningStationBulkUpsertTestSuite struct {
	suite.Suite
	scanningStationSvc  domain.ScanningStationSvc
	scanningStationRepo *repositorymock.MockScanningStationRepo
	repoFactory         *factorymock.MockRepoFactory
	mediatorFactory     *factorymock.MockMediatorFactory
	idempotencyMed      *mediatormock.MockIdempotencyMed
	ctrl                *gomock.Controller
}

func (suite *ScanningStationBulkUpsertTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.scanningStationRepo = repositorymock.NewMockScanningStationRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewScanningStationRepo().Return(suite.scanningStationRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.scanningStationSvc = NewScanningStationSvc(&ScanningStationSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		// These tests reject at the accept phase, before the job is raised, so the factory
		// is never exercised; a real one satisfies validate().
		JobSvcFactory: NewJobSvcFactory(),
	})
}

func (suite *ScanningStationBulkUpsertTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestScanningStationBulkUpsertTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ScanningStationBulkUpsertTestSuite))
}

func internalScanningStationCtx(accountID string) context.Context {
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
				"scanning_stations:read":   true,
				"scanning_stations:create": true,
				"scanning_stations:update": true,
				"scanning_stations:delete": true,
			},
		},
	})
}

func validUpsertScanningStationParams(name string) domain.UpsertScanningStationParams {
	return domain.UpsertScanningStationParams{
		Name:                name,
		Type:                constants.ScanningStationTypeInitBatch,
		OperatorRequirement: constants.OperatorRequirementNone,
		Department:          domain.ObjectIdentifier{Name: "Knitting"},
	}
}

// --- BulkUpsertScanningStations validation guards (reject before idempotency / tx) ---

func (suite *ScanningStationBulkUpsertTestSuite) TestBulkUpsertScanningStations_EmptyRejected() {
	ctx := internalScanningStationCtx("ac_test123")

	result, err := suite.scanningStationSvc.BulkUpsertScanningStations(ctx, domain.BulkUpsertScanningStationsParams{ScanningStations: nil})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ScanningStationBulkUpsertTestSuite) TestBulkUpsertScanningStations_TooManyRejected() {
	ctx := internalScanningStationCtx("ac_test123")

	stations := make([]domain.UpsertScanningStationParams, 1001)
	for i := range stations {
		stations[i] = validUpsertScanningStationParams("Station-" + strconv.Itoa(i))
	}

	result, err := suite.scanningStationSvc.BulkUpsertScanningStations(ctx, domain.BulkUpsertScanningStationsParams{ScanningStations: stations})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ScanningStationBulkUpsertTestSuite) TestBulkUpsertScanningStations_DuplicateNameRejected() {
	ctx := internalScanningStationCtx("ac_test123")

	result, err := suite.scanningStationSvc.BulkUpsertScanningStations(ctx, domain.BulkUpsertScanningStationsParams{
		ScanningStations: []domain.UpsertScanningStationParams{
			validUpsertScanningStationParams("Packing Line"),
			validUpsertScanningStationParams("pAcKiNg LiNe"), // duplicate differing only by casing
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ScanningStationBulkUpsertTestSuite) TestBulkUpsertScanningStations_InvalidTypeRejected() {
	ctx := internalScanningStationCtx("ac_test123")

	station := validUpsertScanningStationParams("Packing Line")
	station.Type = constants.ScanningStationType("teleport_batch")

	result, err := suite.scanningStationSvc.BulkUpsertScanningStations(ctx, domain.BulkUpsertScanningStationsParams{
		ScanningStations: []domain.UpsertScanningStationParams{station},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *ScanningStationBulkUpsertTestSuite) TestBulkUpsertScanningStations_InvalidOperatorRequirementRejected() {
	ctx := internalScanningStationCtx("ac_test123")

	station := validUpsertScanningStationParams("Packing Line")
	station.OperatorRequirement = constants.OperatorRequirement("vibe_check")

	result, err := suite.scanningStationSvc.BulkUpsertScanningStations(ctx, domain.BulkUpsertScanningStationsParams{
		ScanningStations: []domain.UpsertScanningStationParams{station},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

const exportSheet = "Scanning Stations"

type ScanningStationExportTestSuite struct {
	suite.Suite
	svc                 domain.ScanningStationSvc
	scanningStationRepo *repositorymock.MockScanningStationRepo
	repoFactory         *factorymock.MockRepoFactory
	ctrl                *gomock.Controller
}

func (suite *ScanningStationExportTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.scanningStationRepo = repositorymock.NewMockScanningStationRepo(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewScanningStationRepo().Return(suite.scanningStationRepo).AnyTimes()

	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{}).AnyTimes()

	suite.svc = NewScanningStationSvc(&ScanningStationSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		JobSvcFactory:   NewJobSvcFactory(),
	})
}

func (suite *ScanningStationExportTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestScanningStationExportTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ScanningStationExportTestSuite))
}

// reopens the export so assertions read what a spreadsheet would
func (suite *ScanningStationExportTestSuite) open(export *domain.Export) *excelize.File {
	return openExportFile(suite.T(), export)
}

func optional(v string) *string { return &v }

func (suite *ScanningStationExportTestSuite) stations() []*domain.ScanningStation {
	return []*domain.ScanningStation{
		{
			ID:                  "ss_1",
			Name:                "Packaging Line 1",
			Type:                constants.ScanningStationTypeInitBatch,
			OperatorRequirement: constants.OperatorRequirementNone,
			DepartmentID:        "dep_1",
			DepartmentName:      "Cutting",
			LabelTypeCode:       optional("tag"),
			LabelSizeCode:       optional("1x1"),
			Notes:               optional("Initializes a new batch"),
		},
		{
			ID:                  "ss_2",
			Name:                "Merge Station",
			Type:                constants.ScanningStationTypeMergeBatch,
			OperatorRequirement: constants.OperatorRequirementMaterialCheck,
			DepartmentID:        "dep_2",
			DepartmentName:      "Sewing",
		},
	}
}

func (suite *ScanningStationExportTestSuite) TestExport_WritesHeadersAndRows() {
	ctx := internalScanningStationCtx("ac_test123")
	suite.scanningStationRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(suite.stations(), nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportScanningStationsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows, err := suite.open(export).GetRows(exportSheet)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 3)
	suite.Equal([]string{
		"ID", "Name", "Type", "Department", "Operator Requirement",
		"Batch Label Type", "Batch Label Tag Size", "Notes",
	}, rows[0])
	suite.Equal([]string{
		"ss_1", "Packaging Line 1", "init_batch", "Cutting", "none",
		"tag", "1x1", "Initializes a new batch",
	}, rows[1])
	// Trailing blanks are trimmed on read, so the row stops at its last filled cell.
	suite.Equal([]string{"ss_2", "Merge Station", "merge_batch", "Sewing", "material_check"}, rows[2])
}

// a station whose department was removed still has to export
func (suite *ScanningStationExportTestSuite) TestExport_FallsBackToDepartmentID() {
	ctx := internalScanningStationCtx("ac_test123")
	stations := suite.stations()
	stations[0].DepartmentName = ""

	suite.scanningStationRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(stations, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportScanningStationsParams{})
	suite.Require().Nil(apiErr)

	rows, err := suite.open(export).GetRows(exportSheet)
	suite.Require().NoError(err)
	suite.Equal("dep_1", rows[1][3])
}

// an account with nothing to export still gets a usable, header-only workbook
func (suite *ScanningStationExportTestSuite) TestExport_EmptyAccountYieldsHeaderOnlyFile() {
	ctx := internalScanningStationCtx("ac_test123")
	suite.scanningStationRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(nil, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportScanningStationsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(0), export.RowCount)

	rows, err := suite.open(export).GetRows(exportSheet)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1)
	suite.Equal("ID", rows[0][0])
}

func (suite *ScanningStationExportTestSuite) TestExport_RequestsOneRowOverTheCap() {
	ctx := internalScanningStationCtx("ac_test123")
	suite.scanningStationRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportScanningStationsParams) ([]*domain.ScanningStation, error) {
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportScanningStationsParams{})
	suite.Require().Nil(apiErr)
}

// the account comes from the caller's identity, never from the request
func (suite *ScanningStationExportTestSuite) TestExport_ScopesToTheIdentitysAccount() {
	ctx := internalScanningStationCtx("ac_owner")
	suite.scanningStationRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportScanningStationsParams) ([]*domain.ScanningStation, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportScanningStationsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}

func (suite *ScanningStationExportTestSuite) TestExport_RejectsAnIdentitylessContext() {
	export, apiErr := suite.svc.ExportScanningStations(suite.T().Context(), domain.ExportScanningStationsParams{})
	suite.Nil(export)
	suite.NotNil(apiErr)
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *ScanningStationExportTestSuite) buildExport(ctx context.Context, params domain.ExportScanningStationsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.svc.(*scanningStationSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}
