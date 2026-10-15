package mediator

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SandboxMedTestSuite struct {
	suite.Suite
	sandboxMed         domain.SandboxMed
	accountRepo        *repositorymock.MockAccountRepo
	accountUserRepo    *repositorymock.MockAccountUserRepo
	sandboxAccountRepo *repositorymock.MockSandboxAccountRepo
	registrationRepo   *repositorymock.MockRegistrationRepo
	calendarRepo       *repositorymock.MockOperatingCalendarRepo
	locationRepo       *repositorymock.MockLocationRepo
	repoFactory        *factorymock.MockRepoFactory
	ctrl               *gomock.Controller
}

// expectDefaultSite allows the default site a new sandbox gets.
func (suite *SandboxMedTestSuite) expectDefaultSite(name string) {
	suite.locationRepo.EXPECT().SiteSeed(gomock.Any(), gomock.Any()).Return(&domain.AccountSiteSeed{AccountName: name}, nil).Times(1)
	suite.locationRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Location{ID: "lc_site", Name: name, TypeCode: "site"}, nil).Times(1)
	suite.locationRepo.EXPECT().SetDefaultSiteIfUnset(gomock.Any(), gomock.Any(), "lc_site").Return(true, nil).Times(1)
}

func (suite *SandboxMedTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.accountRepo = repositorymock.NewMockAccountRepo(suite.ctrl)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.sandboxAccountRepo = repositorymock.NewMockSandboxAccountRepo(suite.ctrl)
	suite.registrationRepo = repositorymock.NewMockRegistrationRepo(suite.ctrl)
	suite.calendarRepo = repositorymock.NewMockOperatingCalendarRepo(suite.ctrl)
	suite.locationRepo = repositorymock.NewMockLocationRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAccountRepo().Return(suite.accountRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewSandboxAccountRepo().Return(suite.sandboxAccountRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewRegistrationRepo().Return(suite.registrationRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOperatingCalendarRepo().Return(suite.calendarRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewLocationRepo().Return(suite.locationRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&recordingOutboxRepo{}).AnyTimes()

	// Provisioning seeds a sandbox's shipping and receiving calendars. What it writes is covered in internal/calendarseed; here it only has to not be a surprise.
	suite.calendarRepo.EXPECT().GetByCode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	suite.calendarRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	suite.calendarRepo.EXPECT().UpsertClosures(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	suite.sandboxMed = NewSandboxMed(&SandboxMedConfig{
		Repos: suite.repoFactory,
	})
}

func (suite *SandboxMedTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestSandboxMedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SandboxMedTestSuite))
}

func (suite *SandboxMedTestSuite) TestCreate_Success() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"
	userID := "usr_user123"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCodeFree, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetSandboxLimit(gomock.Any(), ownerAccountID).
		Return(new(int32(3)), nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		CountByOwnerAccountID(gomock.Any(), ownerAccountID).
		Return(int64(0), nil).
		Times(1)

	suite.accountRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), "My Sandbox", domain.AccountTypeSandbox, constants.PlanCodeFree).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateBusinessAddress(gomock.Any(), gomock.Any(), "My Sandbox", domain.RegistrationAddress{Country: "US"}).
		Return(nil).
		Times(1)
	suite.expectDefaultSite("My Sandbox")

	suite.accountUserRepo.EXPECT().
		GetAdminRoleID(gomock.Any()).
		Return("role_admin", nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountUser(gomock.Any(), gomock.Any(), userID, "role_admin").
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountPortal(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateSystemProducts(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountBranding(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), ownerAccountID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*domain.SandboxAccount, *apierror.APIError) {
			return &domain.SandboxAccount{
				ID:             1,
				TypeID:         typeID,
				OwnerAccountID: ownerAccountID,
				Name:           "My Sandbox",
			}, nil
		}).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, userID, "My Sandbox")

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal(ownerAccountID, result.OwnerAccountID)
	suite.Equal("My Sandbox", result.Name)
}

func (suite *SandboxMedTestSuite) TestCreate_CopiesOwnerPlanCode() {
	ctx := context.Background()
	ownerAccountID := "ac_proowner"
	userID := "usr_prouser"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCodePro, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetSandboxLimit(gomock.Any(), ownerAccountID).
		Return(new(int32(999)), nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		CountByOwnerAccountID(gomock.Any(), ownerAccountID).
		Return(int64(0), nil).
		Times(1)

	suite.accountRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), "Pro Sandbox", domain.AccountTypeSandbox, constants.PlanCodePro).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateBusinessAddress(gomock.Any(), gomock.Any(), "Pro Sandbox", domain.RegistrationAddress{Country: "US"}).
		Return(nil).
		Times(1)

	// The sandbox gets its default site, named after it, at its business address.
	suite.locationRepo.EXPECT().
		SiteSeed(gomock.Any(), gomock.Any()).
		Return(&domain.AccountSiteSeed{AccountName: "Pro Sandbox", ShippingAddressID: new("adr_sb")}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			suite.Equal("Pro Sandbox", p.Name)
			suite.Equal("site", p.TypeCode)
			suite.Equal("adr_sb", *p.AddressID)
			return &domain.Location{ID: id, Name: p.Name, TypeCode: p.TypeCode, AddressID: p.AddressID}, nil
		}).
		Times(1)
	suite.locationRepo.EXPECT().SetDefaultSiteIfUnset(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).Times(1)

	suite.accountUserRepo.EXPECT().
		GetAdminRoleID(gomock.Any()).
		Return("role_admin", nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountUser(gomock.Any(), gomock.Any(), userID, "role_admin").
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountPortal(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateSystemProducts(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountBranding(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), ownerAccountID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*domain.SandboxAccount, *apierror.APIError) {
			return &domain.SandboxAccount{
				ID:             1,
				TypeID:         typeID,
				OwnerAccountID: ownerAccountID,
				Name:           "Pro Sandbox",
			}, nil
		}).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, userID, "Pro Sandbox")

	suite.Nil(err)
	suite.NotNil(result)
}

// SAFETY: DO NOT REMOVE — ensures sandbox accounts cannot create nested
// sandboxes, which would bypass billing and resource limits.
func (suite *SandboxMedTestSuite) TestSafety_Create_RejectsSandboxOwner() {
	ctx := context.Background()
	sandboxAccountID := "ac_sandbox123"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), sandboxAccountID).
		Return(&domain.AccountContext{
			AccountID: sandboxAccountID,
			IsSandbox: true,
		}, nil).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, sandboxAccountID, "usr_user1", "Nested Sandbox")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.CodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "Sandbox accounts cannot create sandboxes.")
}

func (suite *SandboxMedTestSuite) TestCreate_GetAccountContextError() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(nil, apierror.NewNotFoundError("Account not found")).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, "usr_user1", "My Sandbox")

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *SandboxMedTestSuite) TestCreate_MaxSandboxesReached() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCodeFree, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetSandboxLimit(gomock.Any(), ownerAccountID).
		Return(new(int32(3)), nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		CountByOwnerAccountID(gomock.Any(), ownerAccountID).
		Return(int64(3), nil).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, "usr_user1", "My Sandbox")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apperr.CodeLimitExceeded, err.Code)
	suite.Equal(apperr.QuotaInfo{Limit: 3, Used: 3}, err.Details)
}

func (suite *SandboxMedTestSuite) TestCreate_GetPlanCodeError() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCode(""), apierror.NewInternalError(nil, "Failed to get plan code")).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, "usr_user1", "My Sandbox")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.CodeInternalError, err.Code)
}

func (suite *SandboxMedTestSuite) TestCreate_GetSandboxLimitError() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCodeFree, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetSandboxLimit(gomock.Any(), ownerAccountID).
		Return(nil, apierror.NewInternalError(nil, "Failed to get sandbox limit")).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, "usr_user1", "My Sandbox")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.CodeInternalError, err.Code)
}

func (suite *SandboxMedTestSuite) TestCreate_GetSandboxLimitNotFound_ReturnsInternalError() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCodeFree, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetSandboxLimit(gomock.Any(), ownerAccountID).
		Return(nil, apierror.NewNotFoundError("Resource not found.")).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, "usr_user1", "My Sandbox")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.CodeInternalError, err.Code)
}

func (suite *SandboxMedTestSuite) TestCreate_UnlimitedSandboxes() {
	ctx := context.Background()
	ownerAccountID := "ac_unlimited"
	userID := "usr_unlimited"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCodePro, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetSandboxLimit(gomock.Any(), ownerAccountID).
		Return(nil, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), "Unlimited Sandbox", domain.AccountTypeSandbox, constants.PlanCodePro).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateBusinessAddress(gomock.Any(), gomock.Any(), "Unlimited Sandbox", domain.RegistrationAddress{Country: "US"}).
		Return(nil).
		Times(1)
	suite.expectDefaultSite("Unlimited Sandbox")

	suite.accountUserRepo.EXPECT().
		GetAdminRoleID(gomock.Any()).
		Return("role_admin", nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountUser(gomock.Any(), gomock.Any(), userID, "role_admin").
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountPortal(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateSystemProducts(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.registrationRepo.EXPECT().
		CreateAccountBranding(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), ownerAccountID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*domain.SandboxAccount, *apierror.APIError) {
			return &domain.SandboxAccount{
				ID:             1,
				TypeID:         typeID,
				OwnerAccountID: ownerAccountID,
				Name:           "Unlimited Sandbox",
			}, nil
		}).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, userID, "Unlimited Sandbox")

	suite.Nil(err)
	suite.NotNil(result)
}

func (suite *SandboxMedTestSuite) TestCreate_LimitOfOneReached() {
	ctx := context.Background()
	ownerAccountID := "ac_freeuser"

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), ownerAccountID).
		Return(&domain.AccountContext{
			AccountID: ownerAccountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetPlanCode(gomock.Any(), ownerAccountID).
		Return(constants.PlanCodeFree, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetSandboxLimit(gomock.Any(), ownerAccountID).
		Return(new(int32(1)), nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		CountByOwnerAccountID(gomock.Any(), ownerAccountID).
		Return(int64(1), nil).
		Times(1)

	result, err := suite.sandboxMed.Create(ctx, ownerAccountID, "usr_freeuser", "My Sandbox")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apperr.CodeLimitExceeded, err.Code)
	suite.Contains(err.PublicMessage, "Maximum of 1 sandbox accounts per account reached.")
	suite.Equal(apperr.QuotaInfo{Limit: 1, Used: 1}, err.Details)
}

func (suite *SandboxMedTestSuite) TestDelete_Success() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"
	sandboxTypeID := "sbac_sandbox1"
	accountID := "ac_sandbox_acct"

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), sandboxTypeID, gomock.Nil()).
		Return(&domain.SandboxAccount{
			ID:             1,
			TypeID:         sandboxTypeID,
			OwnerAccountID: ownerAccountID,
			AccountID:      accountID,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), accountID).
		Return(&domain.AccountContext{
			AccountID: accountID,
			IsSandbox: true,
		}, nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		DeleteByID(gomock.Any(), int64(1)).
		Return(nil).
		Times(1)

	suite.accountRepo.EXPECT().
		Delete(gomock.Any(), accountID).
		Return(nil).
		Times(1)

	resultAccountID, err := suite.sandboxMed.Delete(ctx, ownerAccountID, sandboxTypeID)

	suite.Nil(err)
	suite.Equal(accountID, resultAccountID)
}

func (suite *SandboxMedTestSuite) TestDelete_LastSandboxCanBeDeleted() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"
	sandboxTypeID := "sbac_sandbox1"
	accountID := "ac_sandbox_acct"

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), sandboxTypeID, gomock.Nil()).
		Return(&domain.SandboxAccount{
			ID:             1,
			TypeID:         sandboxTypeID,
			OwnerAccountID: ownerAccountID,
			AccountID:      accountID,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), accountID).
		Return(&domain.AccountContext{
			AccountID: accountID,
			IsSandbox: true,
		}, nil).
		Times(1)

	suite.sandboxAccountRepo.EXPECT().
		DeleteByID(gomock.Any(), int64(1)).
		Return(nil).
		Times(1)

	suite.accountRepo.EXPECT().
		Delete(gomock.Any(), accountID).
		Return(nil).
		Times(1)

	resultAccountID, err := suite.sandboxMed.Delete(ctx, ownerAccountID, sandboxTypeID)

	suite.Nil(err)
	suite.Equal(accountID, resultAccountID)
}

func (suite *SandboxMedTestSuite) TestDelete_SandboxNotFound() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"
	sandboxTypeID := "sbac_nonexistent"

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), sandboxTypeID, gomock.Nil()).
		Return(nil, apierror.NewNotFoundError("Sandbox not found.")).
		Times(1)

	resultAccountID, err := suite.sandboxMed.Delete(ctx, ownerAccountID, sandboxTypeID)

	suite.NotNil(err)
	suite.Equal("", resultAccountID)
}

func (suite *SandboxMedTestSuite) TestDelete_OwnerMismatch() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"
	sandboxTypeID := "sbac_sandbox1"

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), sandboxTypeID, gomock.Nil()).
		Return(&domain.SandboxAccount{
			ID:             1,
			TypeID:         sandboxTypeID,
			OwnerAccountID: "ac_different_owner",
			AccountID:      "ac_sandbox_acct",
		}, nil).
		Times(1)

	resultAccountID, err := suite.sandboxMed.Delete(ctx, ownerAccountID, sandboxTypeID)

	suite.NotNil(err)
	suite.Equal("", resultAccountID)
	suite.Equal(apierror.CodeResourceNotFound, err.Code)
}

// SAFETY: DO NOT REMOVE — ensures Delete rejects non-sandbox accounts,
// preventing accidental deletion of production account data.
func (suite *SandboxMedTestSuite) TestSafety_Delete_RejectsNonSandboxAccount() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"
	sandboxTypeID := "sbac_sandbox1"
	accountID := "ac_sandbox_acct"

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), sandboxTypeID, gomock.Nil()).
		Return(&domain.SandboxAccount{
			ID:             1,
			TypeID:         sandboxTypeID,
			OwnerAccountID: ownerAccountID,
			AccountID:      accountID,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), accountID).
		Return(&domain.AccountContext{
			AccountID: accountID,
			IsSandbox: false,
		}, nil).
		Times(1)

	resultAccountID, err := suite.sandboxMed.Delete(ctx, ownerAccountID, sandboxTypeID)

	suite.NotNil(err)
	suite.Equal("", resultAccountID)
	suite.Equal(apierror.CodeInternalError, err.Code)
}

func (suite *SandboxMedTestSuite) TestDelete_GetAccountContextError() {
	ctx := context.Background()
	ownerAccountID := "ac_owner123"
	sandboxTypeID := "sbac_sandbox1"
	accountID := "ac_sandbox_acct"

	suite.sandboxAccountRepo.EXPECT().
		FindByTypeID(gomock.Any(), sandboxTypeID, gomock.Nil()).
		Return(&domain.SandboxAccount{
			ID:             1,
			TypeID:         sandboxTypeID,
			OwnerAccountID: ownerAccountID,
			AccountID:      accountID,
		}, nil).
		Times(1)

	suite.accountRepo.EXPECT().
		GetAccountContext(gomock.Any(), accountID).
		Return(nil, apierror.NewInternalError(nil, "DB error")).
		Times(1)

	resultAccountID, err := suite.sandboxMed.Delete(ctx, ownerAccountID, sandboxTypeID)

	suite.NotNil(err)
	suite.Equal("", resultAccountID)
	suite.Equal(apierror.CodeInternalError, err.Code)
}
