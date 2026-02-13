package service

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/repository"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountSvcTracer = tracing.GetTracer("core-service.account_service")

type accountSvcImpl struct {
	accountRepo         domain.AccountRepo
	accountUserRepo     domain.AccountUserRepo
	accountRelationRepo domain.AccountRelationRepo
	rolePermissionRepo  domain.RolePermissionRepo
	sandboxAccountRepo  domain.SandboxAccountRepo
}

type AccountSvcConfig struct {
	Queries *sqlc.Queries
}

// WithDefaults returns a new AccountSvcConfig with zero-value fields replaced by defaults.
func (c *AccountSvcConfig) WithDefaults() *AccountSvcConfig {
	if c == nil {
		c = &AccountSvcConfig{}
	}
	return &AccountSvcConfig{
		Queries: c.Queries,
	}
}

func (c *AccountSvcConfig) validate() error {
	if c.Queries == nil {
		return fmt.Errorf("account service: queries is required")
	}
	return nil
}

func NewAccountSvc(config *AccountSvcConfig) domain.AccountSvc {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountSvcImpl{
		accountRepo:         repository.NewAccountRepo(config.Queries),
		accountUserRepo:     repository.NewAccountUserRepo(config.Queries),
		accountRelationRepo: repository.NewAccountRelationRepo(config.Queries),
		rolePermissionRepo:  repository.NewRolePermissionRepo(config.Queries),
		sandboxAccountRepo:  repository.NewSandboxAccountRepo(config.Queries),
	}
}

func (s *accountSvcImpl) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_context")
	defer span.End()

	return s.accountRepo.GetAccountContext(ctx, accountID)
}

func (s *accountSvcImpl) GetUserAccountAccess(ctx context.Context, userID, accountID string) (*domain.AccountUserAccess, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_user_account_access")
	defer span.End()

	accountUser, apiErr := s.accountUserRepo.FindByAccountAndUserID(ctx, userID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if accountUser == nil {
		return nil, nil
	}

	// Get permissions if user has a role
	permissions := map[string]bool{}
	if accountUser.RoleID != nil {
		rolePerms, apiErr := s.rolePermissionRepo.FindByRoleID(ctx, *accountUser.RoleID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		permissions = rolePerms
	}

	return &domain.AccountUserAccess{
		AccountUserID: accountUser.ID,
		AccountID:     accountUser.AccountID,
		RoleID:        accountUser.RoleID,
		RoleTypeCode:  accountUser.RoleTypeCode,
		Permissions:   permissions,
		LastUsedAt:    accountUser.LastUsedAt,
	}, nil
}

func (s *accountSvcImpl) GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_role_permissions")
	defer span.End()

	if roleID == "" {
		return map[string]bool{}, nil
	}

	return s.rolePermissionRepo.FindByRoleID(ctx, roleID)
}

func (s *accountSvcImpl) GetAccountRelationByUserID(ctx context.Context, ownerAccountID, userID string) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_relation_by_user_id")
	defer span.End()

	return s.accountRelationRepo.FindByOwnerAccountAndUserID(ctx, ownerAccountID, userID)
}

func (s *accountSvcImpl) GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_relation_by_api_key_id")
	defer span.End()

	return s.accountRelationRepo.FindByOwnerAccountAndAPIKeyID(ctx, ownerAccountID, apiKeyID)
}

func (s *accountSvcImpl) MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.mark_account_user_used")
	defer span.End()

	return s.accountUserRepo.UpdateLastUsedAt(ctx, accountUserID, time.Now().UTC())
}

func (s *accountSvcImpl) ListUserAccountAffiliations(ctx context.Context, userID string) ([]domain.AccountAffiliation, *string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.list_user_account_affiliations")
	defer span.End()

	affiliations, apiErr := s.accountUserRepo.FindAffiliationsByUserID(ctx, userID)
	if apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}

	lastUsedAccountID, apiErr := s.accountUserRepo.FindLastUsedAccountID(ctx, userID)
	if apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}

	var lastUsedPtr *string
	if lastUsedAccountID != "" {
		lastUsedPtr = &lastUsedAccountID
	}

	return affiliations, lastUsedPtr, nil
}

func (s *accountSvcImpl) GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_sandbox_account_by_owner")
	defer span.End()

	return s.sandboxAccountRepo.FindFirstByOwnerAccountID(ctx, ownerAccountID)
}

func (s *accountSvcImpl) GetAdminRole(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_admin_role")
	defer span.End()

	return s.accountUserRepo.GetAdminRoleID(ctx)
}
