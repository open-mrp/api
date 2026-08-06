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
	"go.uber.org/mock/gomock"
)

type DepartmentBulkUpsertTestSuite struct {
	suite.Suite
	departmentSvc   domain.DepartmentSvc
	departmentRepo  *repositorymock.MockDepartmentRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	ctrl            *gomock.Controller
}

func (suite *DepartmentBulkUpsertTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.departmentRepo = repositorymock.NewMockDepartmentRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewDepartmentRepo().Return(suite.departmentRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.departmentSvc = NewDepartmentSvc(&DepartmentSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		// These tests reject at the accept phase, before the job is raised, so the factory
		// is never exercised; a real one satisfies validate().
		JobSvcFactory: NewJobSvcFactory(),
	})
}

func (suite *DepartmentBulkUpsertTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestDepartmentBulkUpsertTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DepartmentBulkUpsertTestSuite))
}

func internalDepartmentCtx(accountID string) context.Context {
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
				"departments:read":   true,
				"departments:create": true,
				"departments:update": true,
				"departments:delete": true,
			},
		},
	})
}

// --- BulkUpsertDepartments validation guards (reject before idempotency / tx) ---

func (suite *DepartmentBulkUpsertTestSuite) TestBulkUpsertDepartments_EmptyRejected() {
	ctx := internalDepartmentCtx("ac_test123")

	result, err := suite.departmentSvc.BulkUpsertDepartments(ctx, domain.BulkUpsertDepartmentsParams{Departments: nil})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *DepartmentBulkUpsertTestSuite) TestBulkUpsertDepartments_TooManyRejected() {
	ctx := internalDepartmentCtx("ac_test123")

	departments := make([]domain.UpsertDepartmentParams, 1001)
	for i := range departments {
		departments[i] = domain.UpsertDepartmentParams{Name: "Dept-" + strconv.Itoa(i)}
	}

	result, err := suite.departmentSvc.BulkUpsertDepartments(ctx, domain.BulkUpsertDepartmentsParams{Departments: departments})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *DepartmentBulkUpsertTestSuite) TestBulkUpsertDepartments_DuplicateNameRejected() {
	ctx := internalDepartmentCtx("ac_test123")

	result, err := suite.departmentSvc.BulkUpsertDepartments(ctx, domain.BulkUpsertDepartmentsParams{
		Departments: []domain.UpsertDepartmentParams{
			{Name: "Knitting"},
			{Name: "kNiTTinG"}, // duplicate differing only by casing
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// reopens the export so assertions read what a spreadsheet would
func (suite *DepartmentBulkUpsertTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Departments")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *DepartmentBulkUpsertTestSuite) buildExport(ctx context.Context, params domain.ExportDepartmentsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.departmentSvc.(*departmentSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

// the two child columns are the whole reason this export needs extra queries
func (suite *DepartmentBulkUpsertTestSuite) TestExportDepartments_JoinsChildNames() {
	ctx := internalDepartmentCtx("ac_test123")
	location := "Knitting Floor"
	notes := "Runs two shifts"
	suite.departmentRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.Department{
		{
			ID:           "dep_1",
			Name:         "Knitting",
			LocationName: &location,
			Notes:        &notes,
			ScanningStations: []domain.DepartmentScanningStation{
				{ID: "ss_1", Name: "Init Line"},
				{ID: "ss_2", Name: "Merge Line"},
			},
			Machines: []domain.DepartmentMachine{{ID: "mch_1", Name: "Loom 1"}},
		},
		{ID: "dep_2", Name: "Packing"},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportDepartmentsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 3)
	suite.Equal([]string{"ID", "Name", "Location", "Scanning Stations", "Machines", "Notes"}, rows[0])
	suite.Equal([]string{
		"dep_1", "Knitting", "Knitting Floor", "Init Line; Merge Line", "Loom 1", "Runs two shifts",
	}, rows[1])
	// A department with no children or location stops at its name once blanks are trimmed.
	suite.Equal([]string{"dep_2", "Packing"}, rows[2])
}

func (suite *DepartmentBulkUpsertTestSuite) TestExportDepartments_ScopesToTheIdentitysAccount() {
	ctx := internalDepartmentCtx("ac_owner")
	suite.departmentRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportDepartmentsParams) ([]*domain.Department, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportDepartmentsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}
