package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

func TestRequireAssignableLocation(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	locations := repositorymock.NewMockLocationRepo(ctrl)
	repos.EXPECT().NewLocationRepo().Return(locations).AnyTimes()
	ctx := context.Background()

	locations.EXPECT().GetByIDs(ctx, "ac_1", []string{"sl_active"}).Return([]*domain.Location{{ID: "sl_active", Status: constants.ArchiveStatusActive}}, nil)
	assert.Nil(t, requireAssignableLocation(ctx, repos, "ac_1", "location_id", "sl_active"))

	locations.EXPECT().GetByIDs(ctx, "ac_1", []string{"sl_old"}).Return([]*domain.Location{{ID: "sl_old", Status: constants.ArchiveStatusArchived}}, nil)
	err := requireAssignableLocation(ctx, repos, "ac_1", "location_id", "sl_old")
	require.NotNil(t, err)
	assert.Equal(t, apperr.CodeResourceArchived, err.Code)
	assert.Equal(t, "location_id", err.Param)

	locations.EXPECT().GetByIDs(ctx, "ac_1", []string{"sl_other"}).Return(nil, nil)
	err = requireAssignableLocation(ctx, repos, "ac_1", "location_id", "sl_other")
	require.NotNil(t, err)
	assert.Equal(t, "location_id", err.Param, "another account's location reads as unknown")
}

func TestRequireAssignableMachines(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	machines := repositorymock.NewMockMachineRepo(ctrl)
	repos.EXPECT().NewMachineRepo().Return(machines).AnyTimes()
	ctx := context.Background()

	assert.Nil(t, requireAssignableMachines(ctx, repos, "ac_1", "machine_ids", nil), "nothing to check reads nothing")

	machines.EXPECT().GetByIDs(ctx, "ac_1", []string{"mch_1", "mch_2"}).Return([]*domain.Machine{
		{ID: "mch_1", Status: constants.ArchiveStatusActive}, {ID: "mch_2", Status: constants.ArchiveStatusArchived},
	}, nil)
	err := requireAssignableMachines(ctx, repos, "ac_1", "machine_ids", []string{"mch_1", "mch_2"})
	require.NotNil(t, err)
	assert.Equal(t, apperr.CodeResourceArchived, err.Code)

	machines.EXPECT().GetByIDs(ctx, "ac_1", []string{"mch_missing"}).Return(nil, nil)
	err = requireAssignableMachines(ctx, repos, "ac_1", "machine_ids", []string{"mch_missing"})
	require.NotNil(t, err)
	assert.Equal(t, apierror.CodeValidationFailed, err.Code)
	assert.Equal(t, "machine_ids", err.Param)
}

func TestDeleteDepartment_InUseIsAConflict(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	departments := repositorymock.NewMockDepartmentRepo(ctrl)
	references := repositorymock.NewMockReferenceRepo(ctrl)
	repos.EXPECT().NewDepartmentRepo().Return(departments).AnyTimes()
	repos.EXPECT().NewReferenceRepo().Return(references).AnyTimes()
	departments.EXPECT().Get(gomock.Any(), domain.GetDepartmentParams{AccountID: "ac_1", DepartmentID: "wkcc_1"}).
		Return(&domain.Department{ID: "wkcc_1", AccountID: "ac_1"}, nil)
	references.EXPECT().InUseBy(gomock.Any(), "wkcc_1", domain.DepartmentReferences.Probes).Return([]string{"machines"}, nil)

	svc := NewDepartmentSvc(&DepartmentSvcConfig{Repos: repos, MediatorFactory: factorymock.NewMockMediatorFactory(ctrl), JobSvcFactory: NewJobSvcFactory(), TxManager: &stubTxManager{factory: repos}})
	err := svc.DeleteDepartment(deletedScopeCtx("ac_1"), "wkcc_1")
	require.NotNil(t, err)
	assert.Equal(t, apierror.CodeResourceInUse, err.Code)
	assert.Equal(t, "This work center can't be deleted while machines use it.", err.PublicMessage)
}

func TestDeleteMachine_InUsePointsToArchiving(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	machines := repositorymock.NewMockMachineRepo(ctrl)
	references := repositorymock.NewMockReferenceRepo(ctrl)
	repos.EXPECT().NewMachineRepo().Return(machines).AnyTimes()
	repos.EXPECT().NewReferenceRepo().Return(references).AnyTimes()
	machines.EXPECT().Get(gomock.Any(), domain.GetMachineParams{AccountID: "ac_1", MachineID: "mch_1"}).Return(&domain.Machine{ID: "mch_1"}, nil)
	references.EXPECT().InUseBy(gomock.Any(), "mch_1", domain.MachineReferences.Probes).Return([]string{"downtime events"}, nil)

	svc := NewMachineSvc(&MachineSvcConfig{Repos: repos, MediatorFactory: factorymock.NewMockMediatorFactory(ctrl), JobSvcFactory: NewJobSvcFactory(), TxManager: &stubTxManager{factory: repos}})
	err := svc.DeleteMachine(deletedScopeCtx("ac_1"), "mch_1")
	require.NotNil(t, err)
	assert.Equal(t, apierror.CodeResourceInUse, err.Code)
	assert.Contains(t, err.PublicMessage, "Archive the machine instead")
}

func TestMachineStatus_DefaultsToActive(t *testing.T) {
	t.Parallel()
	assert.Equal(t, constants.ArchiveStatusActive, machineStatus(nil))
	archived := constants.ArchiveStatusArchived
	assert.Equal(t, constants.ArchiveStatusArchived, machineStatus(&archived))
}
