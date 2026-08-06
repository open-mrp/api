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

type MachineBulkUpsertTestSuite struct {
	suite.Suite
	machineSvc      domain.MachineSvc
	machineRepo     *repositorymock.MockMachineRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	ctrl            *gomock.Controller
}

func (suite *MachineBulkUpsertTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.machineRepo = repositorymock.NewMockMachineRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewMachineRepo().Return(suite.machineRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.machineSvc = NewMachineSvc(&MachineSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		// These tests reject at the accept phase, before the job is raised, so the factory
		// is never exercised; a real one satisfies validate().
		JobSvcFactory: NewJobSvcFactory(),
	})
}

func (suite *MachineBulkUpsertTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestMachineBulkUpsertTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MachineBulkUpsertTestSuite))
}

func internalMachineCtx(accountID string) context.Context {
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
				"machines:read":   true,
				"machines:create": true,
				"machines:update": true,
				"machines:delete": true,
			},
		},
	})
}

// --- BulkUpsertMachines validation guards (reject before idempotency / tx) ---

func (suite *MachineBulkUpsertTestSuite) TestBulkUpsertMachines_EmptyRejected() {
	ctx := internalMachineCtx("ac_test123")

	result, err := suite.machineSvc.BulkUpsertMachines(ctx, domain.BulkUpsertMachinesParams{Machines: nil})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *MachineBulkUpsertTestSuite) TestBulkUpsertMachines_TooManyRejected() {
	ctx := internalMachineCtx("ac_test123")

	machines := make([]domain.UpsertMachineParams, 1001)
	for i := range machines {
		machines[i] = domain.UpsertMachineParams{Name: "Machine-" + strconv.Itoa(i), SerialNumber: "SN-" + strconv.Itoa(i), Department: domain.ObjectIdentifier{Name: "Knitting"}}
	}

	result, err := suite.machineSvc.BulkUpsertMachines(ctx, domain.BulkUpsertMachinesParams{Machines: machines})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *MachineBulkUpsertTestSuite) TestBulkUpsertMachines_DuplicateSerialRejected() {
	ctx := internalMachineCtx("ac_test123")

	result, err := suite.machineSvc.BulkUpsertMachines(ctx, domain.BulkUpsertMachinesParams{
		Machines: []domain.UpsertMachineParams{
			{Name: "Loom A", SerialNumber: "SN-SAME", Department: domain.ObjectIdentifier{Name: "Knitting"}},
			{Name: "Loom B", SerialNumber: "sn-same", Department: domain.ObjectIdentifier{Name: "Knitting"}}, // duplicate differing only by casing
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *MachineBulkUpsertTestSuite) TestBulkUpsertMachines_DuplicateNameRejected() {
	ctx := internalMachineCtx("ac_test123")

	result, err := suite.machineSvc.BulkUpsertMachines(ctx, domain.BulkUpsertMachinesParams{
		Machines: []domain.UpsertMachineParams{
			{Name: "Loom", SerialNumber: "SN-1", Department: domain.ObjectIdentifier{Name: "Knitting"}},
			{Name: "lOoM", SerialNumber: "SN-2", Department: domain.ObjectIdentifier{Name: "Knitting"}}, // duplicate differing only by casing
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

const machineExportSheet = "Machines"

type MachineExportTestSuite struct {
	suite.Suite
	svc         domain.MachineSvc
	machineRepo *repositorymock.MockMachineRepo
	repoFactory *factorymock.MockRepoFactory
	ctrl        *gomock.Controller
}

func (suite *MachineExportTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.machineRepo = repositorymock.NewMockMachineRepo(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewMachineRepo().Return(suite.machineRepo).AnyTimes()

	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{}).AnyTimes()

	suite.svc = NewMachineSvc(&MachineSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		JobSvcFactory:   NewJobSvcFactory(),
	})
}

func (suite *MachineExportTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestMachineExportTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MachineExportTestSuite))
}

// reopens the export so assertions read what a spreadsheet would
func (suite *MachineExportTestSuite) open(export *domain.Export) *excelize.File {
	return openExportFile(suite.T(), export)
}

func (suite *MachineExportTestSuite) machines() []*domain.Machine {
	knitting := "Knitting"
	notes := "Runs the night shift"
	return []*domain.Machine{
		{ID: "mch_1", Name: "Loom 1", SerialNumber: "SN-001", DepartmentName: &knitting, Notes: &notes},
		{ID: "mch_2", Name: "Loom 2", SerialNumber: "SN-002", DepartmentName: &knitting},
	}
}

func (suite *MachineExportTestSuite) TestExport_WritesHeadersAndRows() {
	ctx := internalMachineCtx("ac_test123")
	suite.machineRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(suite.machines(), nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportMachinesParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows, err := suite.open(export).GetRows(machineExportSheet)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 3)
	suite.Equal([]string{"ID", "Name", "Serial Number", "Department", "Notes"}, rows[0])
	suite.Equal([]string{"mch_1", "Loom 1", "SN-001", "Knitting", "Runs the night shift"}, rows[1])
	// Trailing blanks are trimmed on read, so the row stops at its last filled cell.
	suite.Equal([]string{"mch_2", "Loom 2", "SN-002", "Knitting"}, rows[2])
}

// an account with nothing to export still gets a usable, header-only workbook
func (suite *MachineExportTestSuite) TestExport_EmptyAccountYieldsHeaderOnlyFile() {
	ctx := internalMachineCtx("ac_test123")
	suite.machineRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(nil, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportMachinesParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(0), export.RowCount)

	rows, err := suite.open(export).GetRows(machineExportSheet)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1)
	suite.Equal("ID", rows[0][0])
}

// the account comes from the caller's identity, never from the request
func (suite *MachineExportTestSuite) TestExport_ScopesToTheIdentitysAccount() {
	ctx := internalMachineCtx("ac_owner")
	suite.machineRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportMachinesParams) ([]*domain.Machine, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportMachinesParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}

func (suite *MachineExportTestSuite) TestExport_RejectsAnIdentitylessContext() {
	export, apiErr := suite.svc.ExportMachines(suite.T().Context(), domain.ExportMachinesParams{})
	suite.Nil(export)
	suite.NotNil(apiErr)
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *MachineExportTestSuite) buildExport(ctx context.Context, params domain.ExportMachinesParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.svc.(*machineSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}
