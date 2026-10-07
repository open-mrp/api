package service

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	pickTestAccountID = "ac_picktest"
	pickTestPickID    = "pk_picktest"
)

type PickSvcTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	pickRepo       *repositorymock.MockPickRepo
	pickLineRepo   *repositorymock.MockPickLineRepo
	unitRepo       *repositorymock.MockUnitRepo
	idempotencyMed *mediatormock.MockIdempotencyMed
	svc            *pickSvcImpl
}

func TestPickSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PickSvcTestSuite))
}

func (suite *PickSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.pickRepo = repositorymock.NewMockPickRepo(suite.ctrl)
	suite.pickLineRepo = repositorymock.NewMockPickLineRepo(suite.ctrl)
	suite.unitRepo = repositorymock.NewMockUnitRepo(suite.ctrl)
	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)

	repos := factorymock.NewMockRepoFactory(suite.ctrl)
	repos.EXPECT().NewPickRepo().Return(suite.pickRepo).AnyTimes()
	repos.EXPECT().NewPickLineRepo().Return(suite.pickLineRepo).AnyTimes()
	repos.EXPECT().NewUnitRepo().Return(suite.unitRepo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	mediators := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: suite.idempotencyMed}).AnyTimes()

	suite.svc = NewPickSvc(&PickSvcConfig{
		Repos:           repos,
		MediatorFactory: mediators,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: repos},
	}).(*pickSvcImpl)
}

func (suite *PickSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func pickTestCtx() context.Context {
	accountID := pickTestAccountID
	role := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_picktest",
			AccountID:    &accountID,
			RoleType:     &role,
			Permissions:  map[string]bool{"picks:read": true, "picks:update": true},
		},
	})
}

func (suite *PickSvcTestSuite) packInTx() (packOutcome, *apierror.APIError) {
	var out packOutcome
	apiErr := packPickInTx(pickTestCtx(), suite.svc, pickTestAccountID, pickTestPickID, 1, &out)
	return out, apiErr
}

// --- pack ---

func (suite *PickSvcTestSuite) TestPack_RefusesAFinishedPick() {
	suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(true, nil)

	_, apiErr := suite.packInTx()

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *PickSvcTestSuite) TestPack_LockedBeforeLinesAreRead() {
	gomock.InOrder(
		suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(false, nil),
		suite.pickRepo.EXPECT().LockLinesToPack(gomock.Any(), pickTestPickID).Return(nil, nil),
	)

	_, apiErr := suite.packInTx()

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code, "a pack that finds nothing left to pack is a bad request")
}

// A shortfall means another writer packed lines this pack read; shipping them would put them on two shipments.
func (suite *PickSvcTestSuite) TestPack_ConflictsWhenFewerLinesPackThanWereRead() {
	suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(false, nil)
	suite.pickRepo.EXPECT().LockLinesToPack(gomock.Any(), pickTestPickID).Return([]*domain.PickLineToPack{
		{ID: "pkln_a", SalesOrderLineID: "sol_a", QuantityValue: "6", QuantityUnitID: "un_ea"},
		{ID: "pkln_b", SalesOrderLineID: "sol_b", QuantityValue: "4", QuantityUnitID: "un_ea"},
	}, nil)
	suite.pickRepo.EXPECT().PackLines(gomock.Any(), []string{"pkln_a", "pkln_b"}).Return(int64(1), nil)

	_, apiErr := suite.packInTx()

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

func (suite *PickSvcTestSuite) TestPack_ShipsExactlyTheLinesItLocked() {
	suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(false, nil)
	suite.pickRepo.EXPECT().LockLinesToPack(gomock.Any(), pickTestPickID).Return([]*domain.PickLineToPack{
		{ID: "pkln_a", SalesOrderLineID: "sol_a", QuantityValue: "6", QuantityUnitID: "un_ea"},
		{ID: "pkln_b", SalesOrderLineID: "sol_b", QuantityValue: "4", QuantityUnitID: "un_ea"},
	}, nil)
	suite.pickRepo.EXPECT().PackLines(gomock.Any(), []string{"pkln_a", "pkln_b"}).Return(int64(2), nil)

	// sol_a still has 4 outstanding, so it gets a remainder line; sol_b is covered.
	suite.pickLineRepo.EXPECT().CalculateRemainingForOrderLine(gomock.Any(), "sol_a").Return("4", "un_ea", nil)
	suite.pickLineRepo.EXPECT().CalculateRemainingForOrderLine(gomock.Any(), "sol_b").Return("0", "un_ea", nil)
	suite.pickLineRepo.EXPECT().HasUnpackedPickLineForOrderLine(gomock.Any(), "sol_a").Return(false, nil)
	suite.pickLineRepo.EXPECT().CreateForRemaining(gomock.Any(), gomock.Any(), gomock.Any(), pickTestPickID, "sol_a").Return(nil)

	suite.pickRepo.EXPECT().GetSalesOrderForPick(gomock.Any(), pickTestAccountID, pickTestPickID).
		Return(&domain.PickSalesOrder{ID: "so_1", Number: "SO-1"}, nil)
	suite.pickRepo.EXPECT().CountShipmentsByOrder(gomock.Any(), "so_1").Return(int64(0), nil)
	suite.pickRepo.EXPECT().CreateShipment(gomock.Any(), gomock.Any()).Return(nil)

	shipped := map[string]string{}
	quantities := map[string]string{}
	suite.pickRepo.EXPECT().CreateQuantity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id, value, _ string) *apierror.APIError {
			quantities[id] = value
			return nil
		}).AnyTimes()
	suite.pickRepo.EXPECT().CreateShipmentLine(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateShipmentLineParams) *apierror.APIError {
			shipped[params.SalesOrderLineID] = quantities[params.QuantityID]
			return nil
		}).Times(2)
	suite.unitRepo.EXPECT().GetCurrencyBaseUnitID(gomock.Any()).Return("un_usd", nil)
	suite.unitRepo.EXPECT().GetFreightWeightUnitID(gomock.Any()).Return("un_lb", nil)
	suite.pickRepo.EXPECT().CreateShippingCase(gomock.Any(), gomock.Any()).Return(nil)
	suite.pickRepo.EXPECT().MarkFinishedIfAllPacked(gomock.Any(), pickTestPickID).Return(nil)
	suite.pickRepo.EXPECT().Get(gomock.Any(), pickTestAccountID, pickTestPickID).
		Return(&domain.Pick{ID: pickTestPickID, SalesOrderID: "so_1"}, nil)

	out, apiErr := suite.packInTx()

	suite.Require().Nil(apiErr)
	suite.Equal("SO-1", out.ShipmentNumber)
	suite.Equal(map[string]string{"sol_a": "6", "sol_b": "4"}, shipped,
		"each shipment line carries the quantity read under the lock")
}

func (suite *PickSvcTestSuite) TestPackAccept_RefusesAFinishedPick() {
	finishedAt := time.Now()
	suite.idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_pack", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	suite.pickRepo.EXPECT().Get(gomock.Any(), pickTestAccountID, pickTestPickID).
		Return(&domain.Pick{ID: pickTestPickID, FinishedAt: &finishedAt}, nil)

	job, apiErr := suite.svc.PackPick(pickTestCtx(), pickTestPickID, 1)

	suite.Nil(job)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code, "the caller learns at accept, not from a failed job")
}

// --- void ---

func (suite *PickSvcTestSuite) TestVoid_ChecksForShipmentsUnderThePickLock() {
	suite.pickRepo.EXPECT().Get(gomock.Any(), pickTestAccountID, pickTestPickID).
		Return(&domain.Pick{ID: pickTestPickID}, nil)
	gomock.InOrder(
		suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(false, nil),
		suite.pickRepo.EXPECT().HasShippedItems(gomock.Any(), pickTestAccountID, pickTestPickID).Return(true, nil),
	)

	pick, apiErr := suite.svc.VoidPick(pickTestCtx(), pickTestPickID)

	suite.Nil(pick)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// --- pick all ---

func (suite *PickSvcTestSuite) TestPickAll_LocksThePickBeforeTouchingLines() {
	gomock.InOrder(
		suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(false, nil),
		suite.pickRepo.EXPECT().Get(gomock.Any(), pickTestAccountID, pickTestPickID).Return(&domain.Pick{ID: pickTestPickID}, nil),
		suite.pickRepo.EXPECT().PickAllLines(gomock.Any(), pickTestPickID).Return(nil),
		suite.pickRepo.EXPECT().Get(gomock.Any(), pickTestAccountID, pickTestPickID).Return(&domain.Pick{ID: pickTestPickID}, nil),
		suite.pickRepo.EXPECT().GetLines(gomock.Any(), pickTestPickID).Return(nil, nil),
	)

	pick, apiErr := suite.svc.PickAllLines(pickTestCtx(), pickTestPickID)

	suite.Require().Nil(apiErr)
	suite.Equal(pickTestPickID, pick.ID)
}

// --- order reopen ---

func (suite *PickSvcTestSuite) TestReopenClosedLines_KeepsShippedLinesPacked() {
	gomock.InOrder(
		suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(true, nil),
		suite.pickRepo.EXPECT().ListPackedLines(gomock.Any(), pickTestPickID).Return([]*domain.PackedPickLine{
			{ID: "pkln_shipped", SalesOrderLineID: "sol_a"},
			{ID: "pkln_remainder", SalesOrderLineID: "sol_a"},
			{ID: "pkln_never_shipped", SalesOrderLineID: "sol_b"},
		}, nil),
		suite.pickRepo.EXPECT().CountShipmentLinesByOrderLine(gomock.Any(), pickTestPickID).Return(map[string]int64{"sol_a": 1}, nil),
		suite.pickRepo.EXPECT().ReopenLines(gomock.Any(), []string{"pkln_remainder", "pkln_never_shipped"}).Return(nil),
	)

	suite.Nil(reopenClosedPickLines(pickTestCtx(), suite.pickRepo, pickTestAccountID, pickTestPickID))
}

func (suite *PickSvcTestSuite) TestReopenClosedLines_LeavesAFullyShippedPickAlone() {
	suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(true, nil)
	suite.pickRepo.EXPECT().ListPackedLines(gomock.Any(), pickTestPickID).Return([]*domain.PackedPickLine{
		{ID: "pkln_1", SalesOrderLineID: "sol_a"},
		{ID: "pkln_2", SalesOrderLineID: "sol_a"},
	}, nil)
	suite.pickRepo.EXPECT().CountShipmentLinesByOrderLine(gomock.Any(), pickTestPickID).Return(map[string]int64{"sol_a": 2}, nil)

	suite.Nil(reopenClosedPickLines(pickTestCtx(), suite.pickRepo, pickTestAccountID, pickTestPickID))
}

func TestUnshippedPackedLineIDs(t *testing.T) {
	t.Parallel()
	line := func(id, orderLine string) *domain.PackedPickLine {
		return &domain.PackedPickLine{ID: id, SalesOrderLineID: orderLine}
	}

	cases := []struct {
		name          string
		packed        []*domain.PackedPickLine
		shipmentLines map[string]int64
		want          []string
	}{
		{"nothing packed", nil, nil, nil},
		{"closed without shipping", []*domain.PackedPickLine{line("a", "sol_1")}, nil, []string{"a"}},
		{"every line shipped", []*domain.PackedPickLine{line("a", "sol_1"), line("b", "sol_1")}, map[string]int64{"sol_1": 2}, nil},
		{
			"two shipments then a close",
			[]*domain.PackedPickLine{line("a", "sol_1"), line("b", "sol_1"), line("c", "sol_1")},
			map[string]int64{"sol_1": 2},
			[]string{"c"},
		},
		{
			"counted per order line",
			[]*domain.PackedPickLine{line("a", "sol_1"), line("b", "sol_2"), line("c", "sol_1"), line("d", "sol_2")},
			map[string]int64{"sol_1": 1, "sol_2": 2},
			[]string{"c"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := unshippedPackedLineIDs(tc.packed, tc.shipmentLines); !slices.Equal(got, tc.want) {
				t.Errorf("unshippedPackedLineIDs() = %v, want %v", got, tc.want)
			}
		})
	}
}
