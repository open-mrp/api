package mediator

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	tracing "github.com/augno/api/shared/tracing"
)

var accountUserMedTracer = tracing.GetTracer("auth-service.account_user_mediator")

type accountUserMedImpl struct {
	repos domain.RepoFactory
}

type AccountUserMedConfig struct {
	Repos domain.RepoFactory
}

func NewAccountUserMed(config AccountUserMedConfig) domain.AccountUserMed {
	return &accountUserMedImpl{
		repos: config.Repos,
	}
}

func DefaultAccountUserMedConfig(queries *sqlc.Queries) AccountUserMedConfig {
	return AccountUserMedConfig{
		Repos: repository.NewRepoFactory(queries),
	}
}

func NewDefaultAccountUserMed(queries *sqlc.Queries) domain.AccountUserMed {
	return NewAccountUserMed(DefaultAccountUserMedConfig(queries))
}

// MarkUsedIfNotRecent marks an account user as used if it has not been used in the last 24 hours.
func (s *accountUserMedImpl) MarkUsedIfNotRecent(ctx context.Context, accountUser *domain.AccountUser) *contracts.APIError {
	ctx, span := accountUserMedTracer.Start(ctx, "mediator.account_user.mark_used_if_not_recent")
	defer span.End()

	accountUserRepo := s.repos.NewAccountUserRepo()

	now := time.Now().UTC()
	threshold := now.Add(-24 * time.Hour)

	// The user has not made any requests on this account before, so let's mark it as used now
	if accountUser.LastUsedAt == nil {
		return accountUserRepo.UpdateLastUsedAt(ctx, accountUser.ID, now)
	}

	// We only want to note that a user has made a request for an account once a day at most
	lastUsed := *accountUser.LastUsedAt
	if lastUsed.Before(threshold) {
		return accountUserRepo.UpdateLastUsedAt(ctx, accountUser.ID, now)
	}

	return nil
}
