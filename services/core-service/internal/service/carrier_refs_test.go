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
	"github.com/open-mrp/apikit/field"
)

func TestCheckCustomerRefs_RefusesAnArchivedCarrierOrServiceLevel(t *testing.T) {
	s := newCustomerRefsSetup(t)
	s.carriers.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"car_old"}).
		Return([]*domain.Carrier{{ID: "car_old", IsActive: false}}, nil)

	apiErr := checkCustomerRefs(context.Background(), s.repos, "ac_seller", newCustomerRefs(domain.CreateCustomerParams{
		DefaultCarrierID: refsPtr("car_old"),
	}))
	require.NotNil(t, apiErr)
	assert.Equal(t, apperr.CodeResourceArchived, apiErr.Code)
	assert.Equal(t, "defaults.carrier_id", apiErr.Param)

	s.carriers.EXPECT().GetOptionsByIDs(gomock.Any(), "ac_seller", []string{"silv_old"}).
		Return([]*domain.ServiceLevel{{ID: "silv_old", IsActive: false}}, nil)
	apiErr = checkCustomerRefs(context.Background(), s.repos, "ac_seller", newCustomerRefs(domain.CreateCustomerParams{
		DefaultServiceLevelID: refsPtr("silv_old"),
	}))
	require.NotNil(t, apiErr)
	assert.Equal(t, apperr.CodeResourceArchived, apiErr.Code)
	assert.Equal(t, "defaults.service_level_id", apiErr.Param)
}

// A reference the customer already holds is not re-checked, so archiving a carrier leaves its customers editable.
func TestCheckCustomerRefs_KeepsAHeldArchivedCarrier(t *testing.T) {
	s := newCustomerRefsSetup(t)
	held := "car_old"
	apiErr := checkCustomerRefs(context.Background(), s.repos, "ac_seller", changedCustomerRefs(domain.UpdateCustomerParams{
		DefaultCarrierID: refsPtr(held),
	}, &domain.Customer{DefaultCarrierID: &held}))
	assert.Nil(t, apiErr)
}

func newDefaultServiceLevelSvc(t *testing.T) (*carrierSvcImpl, *repositorymock.MockServiceLevelRepo) {
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	levels := repositorymock.NewMockServiceLevelRepo(ctrl)
	repos.EXPECT().NewServiceLevelRepo().Return(levels).AnyTimes()
	return &carrierSvcImpl{repos: repos}, levels
}

func TestCheckDefaultServiceLevel(t *testing.T) {
	account := "ac_seller"
	current := "silv_current"
	carrier := &domain.Carrier{ID: "car_1", AccountID: &account, DefaultServiceLevelID: &current}

	t.Run("unchanged or cleared is not checked", func(t *testing.T) {
		svc, _ := newDefaultServiceLevelSvc(t)
		assert.Nil(t, svc.checkDefaultServiceLevel(context.Background(), carrier, field.Unset[string]()))
		assert.Nil(t, svc.checkDefaultServiceLevel(context.Background(), carrier, field.Clear[string]()))
		assert.Nil(t, svc.checkDefaultServiceLevel(context.Background(), carrier, field.Set(current)))
	})

	t.Run("another carrier's service level", func(t *testing.T) {
		svc, levels := newDefaultServiceLevelSvc(t)
		levels.EXPECT().IsInCarrier(gomock.Any(), "silv_other", "car_1").Return(false, nil)
		apiErr := svc.checkDefaultServiceLevel(context.Background(), carrier, field.Set("silv_other"))
		require.NotNil(t, apiErr)
		assert.Equal(t, "default_service_level_id", apiErr.Param)
	})

	t.Run("archived", func(t *testing.T) {
		svc, levels := newDefaultServiceLevelSvc(t)
		levels.EXPECT().IsInCarrier(gomock.Any(), "silv_old", "car_1").Return(true, nil)
		levels.EXPECT().Get(gomock.Any(), account, "silv_old").Return(&domain.ServiceLevel{ID: "silv_old", IsActive: false}, nil)
		apiErr := svc.checkDefaultServiceLevel(context.Background(), carrier, field.Set("silv_old"))
		require.NotNil(t, apiErr)
		assert.Equal(t, apperr.CodeResourceArchived, apiErr.Code)
	})

	t.Run("active and its own", func(t *testing.T) {
		svc, levels := newDefaultServiceLevelSvc(t)
		levels.EXPECT().IsInCarrier(gomock.Any(), "silv_new", "car_1").Return(true, nil)
		levels.EXPECT().Get(gomock.Any(), account, "silv_new").Return(&domain.ServiceLevel{ID: "silv_new", IsActive: true}, nil)
		apiErr := svc.checkDefaultServiceLevel(context.Background(), carrier, field.Set("silv_new"))
		assert.Nil(t, apiErr)
	})
}
