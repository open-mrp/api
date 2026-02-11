package service

import (
	"context"
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
}

func NewAccountSvc(queries *sqlc.Queries) domain.AccountSvc {
	return &accountSvcImpl{
		accountRepo:         repository.NewAccountRepo(queries),
		accountUserRepo:     repository.NewAccountUserRepo(queries),
		accountRelationRepo: repository.NewAccountRelationRepo(queries),
		rolePermissionRepo:  repository.NewRolePermissionRepo(queries),
	}
}

// GetAccountContext returns the context of an account including whether it's a sandbox
func (s *accountSvcImpl) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_context")
	defer span.End()

	return s.accountRepo.GetAccountContext(ctx, accountID)
}

// GetUserAccountAccess returns the user's access to an account including their role and permissions
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

// GetAccountRelationByUserID returns the relationship between accounts based on user
func (s *accountSvcImpl) GetAccountRelationByUserID(ctx context.Context, ownerAccountID, userID string) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_relation_by_user_id")
	defer span.End()

	return s.accountRelationRepo.FindByOwnerAccountAndUserID(ctx, ownerAccountID, userID)
}

// GetAccountRelationByAPIKeyID returns the relationship between accounts based on API key
func (s *accountSvcImpl) GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_relation_by_api_key_id")
	defer span.End()

	return s.accountRelationRepo.FindByOwnerAccountAndAPIKeyID(ctx, ownerAccountID, apiKeyID)
}

// MarkAccountUserUsed marks an account user as recently used
func (s *accountSvcImpl) MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.mark_account_user_used")
	defer span.End()

	return s.accountUserRepo.UpdateLastUsedAt(ctx, accountUserID, time.Now().UTC())
}

// ListUserAccountAffiliations returns the accounts a user is affiliated with
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
