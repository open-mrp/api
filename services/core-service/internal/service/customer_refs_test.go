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
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

type customerRefsSetup struct {
	repos        *factorymock.MockRepoFactory
	carriers     *repositorymock.MockCarrierRepo
	paymentTerms *repositorymock.MockPaymentTermRepo
	groups       *repositorymock.MockAccountGroupRepo
	calendars    *repositorymock.MockOperatingCalendarRepo
	accountUsers *repositorymock.MockAccountUserRepo
}

func newCustomerRefsSetup(t *testing.T) *customerRefsSetup {
	ctrl := gomock.NewController(t)
	s := &customerRefsSetup{
		repos:        factorymock.NewMockRepoFactory(ctrl),
		carriers:     repositorymock.NewMockCarrierRepo(ctrl),
		paymentTerms: repositorymock.NewMockPaymentTermRepo(ctrl),
		groups:       repositorymock.NewMockAccountGroupRepo(ctrl),
		calendars:    repositorymock.NewMockOperatingCalendarRepo(ctrl),
		accountUsers: repositorymock.NewMockAccountUserRepo(ctrl),
	}
	s.repos.EXPECT().NewCarrierRepo().Return(s.carriers).AnyTimes()
	s.repos.EXPECT().NewPaymentTermRepo().Return(s.paymentTerms).AnyTimes()
	s.repos.EXPECT().NewAccountGroupRepo().Return(s.groups).AnyTimes()
	s.repos.EXPECT().NewOperatingCalendarRepo().Return(s.calendars).AnyTimes()
	s.repos.EXPECT().NewAccountUserRepo().Return(s.accountUsers).AnyTimes()
	return s
}

func refsPtr(s string) *string { return &s }

func TestCheckCustomerRefs_RefusesOnTheFieldThatNamedIt(t *testing.T) {
	s := newCustomerRefsSetup(t)
	s.carriers.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"car_own"}).
		Return([]*domain.Carrier{{ID: "car_own"}}, nil)
	// Another account's term is not among the seller's own or the system's, so the scoped read omits it.
	s.paymentTerms.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"pytm_other"}).Return(nil, nil)

	apiErr := checkCustomerRefs(context.Background(), s.repos, "ac_seller", newCustomerRefs(domain.CreateCustomerParams{
		DefaultCarrierID:     refsPtr("car_own"),
		DefaultPaymentTermID: refsPtr("pytm_other"),
	}))

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	assert.Equal(t, "default_payment_term_id", apiErr.Param)
}

func TestCheckCustomerRefs_ReadsTheTypeAndPriceGroupsTogether(t *testing.T) {
	s := newCustomerRefsSetup(t)
	s.groups.EXPECT().GetByIDs(gomock.Any(), "ac_seller", []string{"acgp_type", "acgp_price", "acgp_other"}).
		Return([]*domain.AccountGroup{{ID: "acgp_type"}, {ID: "acgp_price"}}, nil).Times(1)

	apiErr := checkCustomerRefs(context.Background(), s.repos, "ac_seller", newCustomerRefs(domain.CreateCustomerParams{
		CustomerTypeGroupID:   refsPtr("acgp_type"),
		CustomerPriceGroupIDs: []string{"acgp_price", "acgp_other"},
	}))

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	assert.Equal(t, "customer_price_group_ids", apiErr.Param)
}

func TestCheckCustomerRefs_CalendarAndSalesRepMustBeTheAccounts(t *testing.T) {
	s := newCustomerRefsSetup(t)
	s.calendars.EXPECT().Get(gomock.Any(), "ac_seller", "opcal_own").Return(&domain.OperatingCalendar{ID: "opcal_own"}, nil)
	s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_seller", "acus_other", nil).
		Return(nil, apierror.NewResourceNotFoundError("Account user not found."))

	apiErr := checkCustomerRefs(context.Background(), s.repos, "ac_seller", newCustomerRefs(domain.CreateCustomerParams{
		ReceiveCalendarID: refsPtr("opcal_own"),
		DefaultSalesRepID: refsPtr("acus_other"),
	}))

	require.NotNil(t, apiErr)
	assert.Equal(t, "default_sales_rep_id", apiErr.Param)
}

func TestCheckCustomerRefs_PassesAnUnexpectedLookupError(t *testing.T) {
	s := newCustomerRefsSetup(t)
	s.calendars.EXPECT().Get(gomock.Any(), "ac_seller", "opcal_own").Return(nil, apierror.NewInternalError(nil, "boom"))

	apiErr := checkCustomerRefs(context.Background(), s.repos, "ac_seller", newCustomerRefs(domain.CreateCustomerParams{
		ReceiveCalendarID: refsPtr("opcal_own"),
	}))

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInternalError, apiErr.Code)
}

func TestChangedCustomerRefs_LeavesWhatTheCustomerHoldsAlone(t *testing.T) {
	old := &domain.Customer{
		DefaultCarrierID:  refsPtr("car_held"),
		TypeGroupID:       refsPtr("acgp_type"),
		ReceiveCalendarID: refsPtr("opcal_held"),
		PriceGroups:       []domain.CustomerAccountGroup{{ID: "acgp_held"}},
	}

	refs := changedCustomerRefs(domain.UpdateCustomerParams{
		DefaultCarrierID:         refsPtr("car_held"),
		DefaultPaymentTermID:     refsPtr("pytm_new"),
		ReceiveCalendarID:        field.Clear[string](),
		DefaultServiceLevelID:    field.Set("crop_new"),
		CustomerTypeGroupID:      refsPtr("acgp_type"),
		HasCustomerPriceGroupIDs: true,
		CustomerPriceGroupIDs:    []string{"acgp_held", "acgp_new"},
	}, old)

	assert.Equal(t, customerRefs{
		{kind: customerRefServiceLevel, id: "crop_new", param: "default_service_level_id"},
		{kind: customerRefPaymentTerm, id: "pytm_new", param: "default_payment_term_id"},
		{kind: customerRefAccountGroup, id: "acgp_new", param: "customer_price_group_ids"},
	}, refs)
}
