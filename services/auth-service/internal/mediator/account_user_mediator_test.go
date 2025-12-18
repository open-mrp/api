package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/contracts"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// AccountUserMedTestSuite provides a test suite for AccountUserMed tests
type AccountUserMedTestSuite struct {
	suite.Suite
	accountUserMed  domain.AccountUserMed
	accountUserRepo *repositorymock.MockAccountUserRepo
	repoFactory     *factorymock.MockRepoFactory
	ctrl            *gomock.Controller
}

// SetupTest runs before each test in the suite
func (suite *AccountUserMedTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()

	accountUserMedConfig := AccountUserMedConfig{
		Repos: suite.repoFactory,
	}
	suite.accountUserMed = NewAccountUserMed(accountUserMedConfig)
}

// TearDownTest runs after each test in the suite
func (suite *AccountUserMedTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// TestAccountUserMedTestSuite runs the test suite
func TestAccountUserMedTestSuite(t *testing.T) {
	suite.Run(t, new(AccountUserMedTestSuite))
}

func (suite *AccountUserMedTestSuite) TestMarkUsedIfNotRecent_LastUsedAtIsNil() {
	ctx := context.Background()
	accountUser := &domain.AccountUser{
		ID:         "account-user-id",
		UserID:     "user-id",
		AccountID:  "account-id",
		LastUsedAt: nil,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	suite.accountUserRepo.EXPECT().
		UpdateLastUsedAt(gomock.Any(), "account-user-id", gomock.Any()).
		DoAndReturn(func(ctx context.Context, accountUserID string, lastUsedAt time.Time) *contracts.APIError {
			// Verify the time is approximately now (within 1 second)
			now := time.Now().UTC()
			suite.WithinDuration(now, lastUsedAt, time.Second)
			return nil
		}).
		Times(1)

	err := suite.accountUserMed.MarkUsedIfNotRecent(ctx, accountUser)
	suite.Nil(err)
}

func (suite *AccountUserMedTestSuite) TestMarkUsedIfNotRecent_LastUsedAtOlderThan24Hours() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-25 * time.Hour)
	accountUser := &domain.AccountUser{
		ID:         "account-user-id",
		UserID:     "user-id",
		AccountID:  "account-id",
		LastUsedAt: &lastUsedAt,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	suite.accountUserRepo.EXPECT().
		UpdateLastUsedAt(gomock.Any(), "account-user-id", gomock.Any()).
		DoAndReturn(func(ctx context.Context, accountUserID string, lastUsedAt time.Time) *contracts.APIError {
			// Verify the time is approximately now (within 1 second)
			now := time.Now().UTC()
			suite.WithinDuration(now, lastUsedAt, time.Second)
			return nil
		}).
		Times(1)

	err := suite.accountUserMed.MarkUsedIfNotRecent(ctx, accountUser)
	suite.Nil(err)
}

func (suite *AccountUserMedTestSuite) TestMarkUsedIfNotRecent_LastUsedAtLessThan24HoursAgo() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-23 * time.Hour)
	accountUser := &domain.AccountUser{
		ID:         "account-user-id",
		UserID:     "user-id",
		AccountID:  "account-id",
		LastUsedAt: &lastUsedAt,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	err := suite.accountUserMed.MarkUsedIfNotRecent(ctx, accountUser)
	suite.Nil(err)
}

func (suite *AccountUserMedTestSuite) TestMarkUsedIfNotRecent_LastUsedAtVeryRecent() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-1 * time.Hour)
	accountUser := &domain.AccountUser{
		ID:         "account-user-id",
		UserID:     "user-id",
		AccountID:  "account-id",
		LastUsedAt: &lastUsedAt,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	err := suite.accountUserMed.MarkUsedIfNotRecent(ctx, accountUser)
	suite.Nil(err)
}

func (suite *AccountUserMedTestSuite) TestMarkUsedIfNotRecent_UpdateLastUsedAtReturnsError() {
	ctx := context.Background()
	// Use a time that's definitely more than 24 hours ago to ensure we trigger the update
	lastUsedAt := time.Now().UTC().Add(-25 * time.Hour)
	accountUser := &domain.AccountUser{
		ID:         "account-user-id-error",
		UserID:     "user-id",
		AccountID:  "account-id",
		LastUsedAt: &lastUsedAt,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	expectedError := contracts.NewInternalError(nil, "Failed to update account user last used at.")
	suite.accountUserRepo.EXPECT().
		UpdateLastUsedAt(gomock.Any(), "account-user-id-error", gomock.Any()).
		DoAndReturn(func(ctx context.Context, accountUserID string, lastUsedAt time.Time) *contracts.APIError {
			// Verify the time is approximately now (within 1 second)
			now := time.Now().UTC()
			suite.WithinDuration(now, lastUsedAt, time.Second)
			return expectedError
		}).
		Times(1)

	err := suite.accountUserMed.MarkUsedIfNotRecent(ctx, accountUser)
	suite.NotNil(err)
	suite.Equal(expectedError, err)
}
