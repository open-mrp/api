package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	s3client "github.com/augno/api/shared/cloud/s3"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var tenancySvcTracer = tracing.GetTracer("core-service.tenancy_service")

type tenancySvcImpl struct {
	accountUserRepo     domain.AccountUserRepo
	accountRelationRepo domain.AccountRelationRepo
	accountRepo         domain.AccountRepo
	userRepo            domain.UserRepo
	sandboxAccountRepo  domain.SandboxAccountRepo
	rolePermissionRepo  domain.RolePermissionRepo
	authClient          domain.CoreAuthClient
	s3Client            s3client.ObjectStore
	userPhotosBucket    string
}

type TenancySvcConfig struct {
	// RepoFactory (required) is the repository factory.
	RepoFactory domain.RepoFactory

	// AuthClient (required) is the auth-service client.
	AuthClient domain.CoreAuthClient

	// S3Client (required) is the object store client used for file storage.
	S3Client s3client.ObjectStore

	// UserPhotosBucket (optional; default: "") is the S3 bucket for user photos. It is not validated at construction.
	UserPhotosBucket string
}

func (c *TenancySvcConfig) validate() error {
	if c.RepoFactory == nil {
		return fmt.Errorf("tenancy service: repo factory is required")
	}
	if c.AuthClient == nil {
		return fmt.Errorf("tenancy service: auth client is required")
	}
	if c.S3Client == nil {
		return fmt.Errorf("tenancy service: s3 client is required")
	}
	return nil
}

func NewTenancySvc(config *TenancySvcConfig) domain.TenancySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &tenancySvcImpl{
		accountUserRepo:     config.RepoFactory.NewAccountUserRepo(),
		accountRelationRepo: config.RepoFactory.NewAccountRelationRepo(),
		accountRepo:         config.RepoFactory.NewAccountRepo(),
		userRepo:            config.RepoFactory.NewUserRepo(),
		sandboxAccountRepo:  config.RepoFactory.NewSandboxAccountRepo(),
		rolePermissionRepo:  config.RepoFactory.NewRolePermissionRepo(),
		authClient:          config.AuthClient,
		s3Client:            config.S3Client,
		userPhotosBucket:    config.UserPhotosBucket,
	}
}

func (s *tenancySvcImpl) GetTenancy(ctx context.Context, userID string, targetAccountID *string) (*domain.Tenancy, *apierror.APIError) {
	ctx, span := tenancySvcTracer.Start(ctx, "service.tenancy.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	// Tenancy is account-agnostic: the user may not have selected an account yet, so require an authenticated user actor but not an assigned account.
	if apiErr := identity.CheckHasUserActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	allAccounts, apiErr := s.accountUserRepo.FindTenancyAccountsByUserID(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	statusMap := buildStatusMap(allAccounts)
	accounts := filterActiveAccounts(allAccounts, statusMap)

	var currentAccount *domain.TenancyAccount
	if targetAccountID != nil {
		for i := range accounts {
			if accounts[i].AccountID == *targetAccountID {
				currentAccount = &accounts[i]
				break
			}
		}
	}

	if currentAccount == nil && len(accounts) > 0 {
		sorted := make([]domain.TenancyAccount, len(accounts))
		copy(sorted, accounts)
		sort.Slice(sorted, func(i, j int) bool {
			aIsPaid := sorted[i].PlanCode != "free"
			bIsPaid := sorted[j].PlanCode != "free"
			if aIsPaid && !bIsPaid {
				return true
			}
			if !aIsPaid && bIsPaid {
				return false
			}
			if sorted[i].LastUsedAt != nil && sorted[j].LastUsedAt != nil {
				return sorted[i].LastUsedAt.After(*sorted[j].LastUsedAt)
			}
			return false
		})
		currentAccount = &sorted[0]
	}

	pendingRegistration, apiErr := s.loadPendingRegistration(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if currentAccount == nil {
		return &domain.Tenancy{
			HasTenancy:          false,
			PendingRegistration: pendingRegistration,
		}, nil
	}

	return s.buildTenancyResponse(ctx, currentAccount, accounts, pendingRegistration)
}

func (s *tenancySvcImpl) SwitchAccount(ctx context.Context, userID, accountID string) (*domain.Tenancy, *apierror.APIError) {
	ctx, span := tenancySvcTracer.Start(ctx, "service.tenancy.switch_account")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	// Switching accounts is account-agnostic by definition; require an authenticated user actor but not an assigned account.
	if apiErr := identity.CheckHasUserActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	allAccounts, apiErr := s.accountUserRepo.FindTenancyAccountsByUserID(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	statusMap := buildStatusMap(allAccounts)

	var targetAccount *domain.TenancyAccount
	for i := range allAccounts {
		if allAccounts[i].AccountID == accountID {
			targetAccount = &allAccounts[i]
			break
		}
	}
	if targetAccount == nil {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this account."))
	}

	if targetAccount.AccountUserStatusCode == "disabled" {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Your account has been locked. Please contact your administrator to regain access."))
	}
	if targetAccount.AccountUserStatusCode == "removed" {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Your account has been removed. Please contact your administrator to regain access."))
	}

	if targetAccount.AccountTypeCode == "sandbox" && targetAccount.OwnerAccountID != nil {
		ownerStatus := statusMap[*targetAccount.OwnerAccountID]
		if ownerStatus == "disabled" {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("Your account has been locked on the production account. Please contact your administrator to regain access."))
		}
		if ownerStatus == "removed" {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("Your account has been removed from the production account. Please contact your administrator to regain access."))
		}
	}

	if targetAccount.OnboardingStatusCode == "suspended" || targetAccount.OnboardingStatusCode == "deactivated" {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Cannot switch to an inactive account."))
	}

	accounts := filterActiveAccounts(allAccounts, statusMap)

	if apiErr := s.accountUserRepo.MarkUsedByAccountAndUser(ctx, accountID, userID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	pendingRegistration, apiErr := s.loadPendingRegistration(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.buildTenancyResponse(ctx, targetAccount, accounts, pendingRegistration)
}

func (s *tenancySvcImpl) GetCurrentUser(ctx context.Context, userID string, targetAccountID *string) (*domain.UserRecord, *apierror.APIError) {
	ctx, span := tenancySvcTracer.Start(ctx, "service.tenancy.get_current_user")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	// Account-agnostic like GetTenancy: /me is called before an account is selected, so the identity may have no actor account. CheckIsUser would 403 there; require an authenticated user actor only.
	if apiErr := identity.CheckHasUserActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	user, apiErr := s.userRepo.FindByID(ctx, userID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("User not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}

	if targetAccountID != nil && user.ImageURL != nil {
		// A non-null user.image_url is the authoritative avatar existence signal (set only on photo upload), so we presign directly instead of doing an S3 HeadObject — presigning is a local SigV4 operation with no network I/O, keeping /me off the S3 critical path. On any signing error we simply leave the persisted image_url value in place.
		key := *targetAccountID + "/" + userID + ".png"
		if url, apiErr := s.s3Client.GetPresignedURL(ctx, s.userPhotosBucket, key, 60*time.Minute); apiErr == nil {
			user.ImageURL = &url
		}
	}

	return user, nil
}

func (s *tenancySvcImpl) ListCustomerAccountsForUser(ctx context.Context, userID, vendorAccountID string) ([]domain.CustomerAccountSummary, *apierror.APIError) {
	ctx, span := tenancySvcTracer.Start(ctx, "service.tenancy.list_customer_accounts")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckHasUserActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accounts, apiErr := s.accountRelationRepo.FindCustomerAccountsByVendorAndUser(ctx, vendorAccountID, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return accounts, nil
}

func (s *tenancySvcImpl) loadPendingRegistration(ctx context.Context, userID string) (*domain.TenancyPendingRegistration, *apierror.APIError) {
	session, apiErr := s.authClient.GetIncompleteRegistrationSession(ctx, userID)
	if apiErr != nil {
		return nil, apiErr
	}
	if session == nil {
		return nil, nil
	}
	return &domain.TenancyPendingRegistration{
		SessionID: session.SessionID,
		PlanCode:  session.PlanCode,
		Step:      session.Step,
		CreatedAt: session.CreatedAt,
	}, nil
}

func (s *tenancySvcImpl) buildTenancyResponse(ctx context.Context, currentAccount *domain.TenancyAccount, activeAccounts []domain.TenancyAccount, pendingRegistration *domain.TenancyPendingRegistration) (*domain.Tenancy, *apierror.APIError) {
	var role *domain.TenancyRole
	if currentAccount.RoleID != nil {
		role = &domain.TenancyRole{
			ID:       *currentAccount.RoleID,
			Name:     derefStr(currentAccount.RoleName),
			RoleType: derefStr(currentAccount.RoleType),
		}
		if currentAccount.RoleCreatedAt != nil {
			role.CreatedAt = *currentAccount.RoleCreatedAt
		}
		if currentAccount.RoleUpdatedAt != nil {
			role.UpdatedAt = *currentAccount.RoleUpdatedAt
		}

		permMap, apiErr := s.rolePermissionRepo.FindByRoleID(ctx, *currentAccount.RoleID)
		if apiErr != nil {
			return nil, apiErr
		}
		permissions := make([]string, 0, len(permMap))
		for code := range permMap {
			permissions = append(permissions, code)
		}
		sort.Strings(permissions)
		role.Permissions = permissions
	}

	var accountPlan *domain.TenancyAccountPlan
	if currentAccount.Plan != nil {
		limits, apiErr := s.accountRepo.ListPlanLimits(ctx, currentAccount.Plan.TypeID)
		if apiErr != nil {
			return nil, apiErr
		}
		features, apiErr := s.accountRepo.ListPlanFeatures(ctx, currentAccount.Plan.TypeID)
		if apiErr != nil {
			return nil, apiErr
		}
		accountPlan = &domain.TenancyAccountPlan{
			TypeID:        currentAccount.Plan.TypeID,
			Name:          currentAccount.Plan.Name,
			PlanTypeCode:  currentAccount.Plan.PlanTypeCode,
			Version:       currentAccount.Plan.Version,
			PricePerSeat:  currentAccount.Plan.PricePerSeat,
			PricePerMonth: currentAccount.Plan.PricePerMonth,
			SeatMinimum:   currentAccount.Plan.SeatMinimum,
			Limits:        limits,
			Features:      features,
		}
	}

	isAdmin := currentAccount.RoleType != nil && *currentAccount.RoleType == "admin"

	var sandboxes []domain.TenancySandbox
	var ownerAccount *domain.TenancyOwnerAccount

	if currentAccount.AccountTypeCode == "sandbox" {
		if currentAccount.OwnerAccountID != nil {
			ownerName, _ := s.accountRepo.GetName(ctx, *currentAccount.OwnerAccountID)
			ownerAccount = &domain.TenancyOwnerAccount{
				ID:   *currentAccount.OwnerAccountID,
				Name: ownerName,
			}
		}
	} else {
		ownerAccount = &domain.TenancyOwnerAccount{
			ID:   currentAccount.AccountID,
			Name: currentAccount.AccountName,
		}

		if isAdmin {
			sandboxList, apiErr := s.sandboxAccountRepo.List(ctx, currentAccount.AccountID, nil, 100, nil, nil)
			if apiErr == nil && sandboxList != nil {
				for _, sb := range sandboxList.Sandboxes {
					sandboxes = append(sandboxes, domain.TenancySandbox{
						ID:   sb.AccountID,
						Name: sb.Name,
					})
				}
			}
		}
	}

	sandboxIDs := make(map[string]bool)
	for _, sb := range sandboxes {
		sandboxIDs[sb.ID] = true
	}

	slug, _ := s.accountRepo.GetPortalSlug(ctx, currentAccount.AccountID)

	var otherAccounts []domain.TenancyOtherAccount
	for _, a := range activeAccounts {
		if a.AccountID == currentAccount.AccountID {
			continue
		}
		if sandboxIDs[a.AccountID] {
			continue
		}
		if ownerAccount != nil && a.AccountID == ownerAccount.ID {
			continue
		}
		if a.OnboardingStatusCode != "active" {
			continue
		}
		if !isAdmin && a.AccountTypeCode == "sandbox" {
			continue
		}
		otherAccounts = append(otherAccounts, domain.TenancyOtherAccount{
			ID:   a.AccountID,
			Name: a.AccountName,
			Type: a.AccountTypeCode,
		})
	}

	if sandboxes == nil {
		sandboxes = []domain.TenancySandbox{}
	}
	if otherAccounts == nil {
		otherAccounts = []domain.TenancyOtherAccount{}
	}

	return &domain.Tenancy{
		HasTenancy: true,
		CurrentAccount: &domain.TenancyCurrentAccount{
			ID:                       currentAccount.AccountID,
			Name:                     currentAccount.AccountName,
			Type:                     currentAccount.AccountTypeCode,
			OnboardingStatus:         currentAccount.OnboardingStatusCode,
			PlanCode:                 currentAccount.PlanCode,
			Slug:                     slug,
			Role:                     role,
			InternalStripeCustomerID: currentAccount.InternalStripeCustomerID,
			AccountPlan:              accountPlan,
		},
		Sandboxes:           sandboxes,
		OwnerAccount:        ownerAccount,
		OtherAccounts:       otherAccounts,
		PendingRegistration: pendingRegistration,
	}, nil
}

func buildStatusMap(accounts []domain.TenancyAccount) map[string]string {
	m := make(map[string]string, len(accounts))
	for _, a := range accounts {
		m[a.AccountID] = a.AccountUserStatusCode
	}
	return m
}

func filterActiveAccounts(allAccounts []domain.TenancyAccount, statusMap map[string]string) []domain.TenancyAccount {
	var result []domain.TenancyAccount
	for _, a := range allAccounts {
		if a.AccountUserStatusCode != "active" {
			continue
		}
		if a.AccountTypeCode == "sandbox" && a.OwnerAccountID != nil {
			ownerStatus := statusMap[*a.OwnerAccountID]
			if ownerStatus == "disabled" || ownerStatus == "removed" {
				continue
			}
		}
		result = append(result, a)
	}
	return result
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
