package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var accountSvcTracer = tracing.GetTracer("core-service.account_service")

type accountSvcImpl struct {
	accountRepo         domain.AccountRepo
	accountUserRepo     domain.AccountUserRepo
	accountRelationRepo domain.AccountRelationRepo
	rolePermissionRepo  domain.RolePermissionRepo
	mediatorFactory     domain.MediatorFactory
	txManager           TransactionManager
}

type AccountSvcConfig struct {
	RepoFactory     domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *AccountSvcConfig) validate() error {
	if c.RepoFactory == nil {
		return fmt.Errorf("account service: repo factory is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("account service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("account service: tx manager is required")
	}
	return nil
}

func NewAccountSvc(config *AccountSvcConfig) domain.AccountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountSvcImpl{
		accountRepo:         config.RepoFactory.NewAccountRepo(),
		accountUserRepo:     config.RepoFactory.NewAccountUserRepo(),
		accountRelationRepo: config.RepoFactory.NewAccountRelationRepo(),
		rolePermissionRepo:  config.RepoFactory.NewRolePermissionRepo(),
		mediatorFactory:     config.MediatorFactory,
		txManager:           config.TxManager,
	}
}

func (s *accountSvcImpl) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_context")
	defer span.End()

	return s.accountRepo.GetAccountContext(ctx, accountID)
}

func (s *accountSvcImpl) GetUserAccountAccess(ctx context.Context, userID, accountID string) (*domain.AccountUserAccess, bool, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_user_account_access")
	defer span.End()

	accountUser, apiErr := s.accountUserRepo.FindByAccountAndUserID(ctx, userID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, false, nil
		}
		return nil, false, tracing.Trace(span, apiErr)
	}

	// Get permissions if user has a role
	permissions := map[string]bool{}
	if accountUser.RoleID != nil {
		rolePerms, apiErr := s.rolePermissionRepo.FindByRoleID(ctx, *accountUser.RoleID)
		if apiErr != nil {
			return nil, false, tracing.Trace(span, apiErr)
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
	}, true, nil
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

	relation, apiErr := s.accountRelationRepo.FindByOwnerAccountAndUserID(ctx, ownerAccountID, userID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return relation, nil
}

func (s *accountSvcImpl) GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_relation_by_api_key_id")
	defer span.End()

	relation, apiErr := s.accountRelationRepo.FindByOwnerAccountAndAPIKeyID(ctx, ownerAccountID, apiKeyID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return relation, nil
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

func (s *accountSvcImpl) GetAdminRole(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_admin_role")
	defer span.End()

	return s.accountUserRepo.GetAdminRoleID(ctx)
}

func (s *accountSvcImpl) UpdateAccountSubscription(ctx context.Context, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.update_account_subscription")
	defer span.End()

	planTypeID, apiErr := s.accountRepo.GetPlanTypeIDByCode(ctx, planCode)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := s.accountRepo.UpdateSubscription(ctx, accountID, status, planCode, &planTypeID, stripeSubID, periodEnd, stripeCustomerID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Adjust active users based on the new plan's seat limit.
	s.adjustAccountSeats(ctx, accountID, planCode)

	return nil
}

// adjustAccountSeats reconciles active users with the new plan's seat limit.
// On downgrade: deactivates excess users (keeps the requesting user or admin).
// On upgrade: reactivates disabled users up to the new seat limit.
// Failures are logged but not propagated — the subscription update already succeeded.
func (s *accountSvcImpl) adjustAccountSeats(ctx context.Context, accountID, planCode string) {
	slog.InfoContext(ctx, "[seat-adjust] starting",
		"account_id", accountID, "plan_code", planCode)

	seatLimit, apiErr := s.accountRepo.GetSeatLimitByPlanCode(ctx, planCode)
	if apiErr != nil {
		slog.WarnContext(ctx, "[seat-adjust] failed to get seat limit",
			"account_id", accountID, "plan_code", planCode, "error", apiErr.PublicMessage)
		return
	}

	// nil means unlimited seats — reactivate any previously disabled users
	if seatLimit == nil {
		s.reactivateAllDisabledUsers(ctx, accountID)
		return
	}

	activeCount, apiErr := s.accountUserRepo.CountActive(ctx, accountID)
	if apiErr != nil {
		slog.WarnContext(ctx, "[seat-adjust] failed to count active users",
			"account_id", accountID, "error", apiErr.PublicMessage)
		return
	}

	limit := int64(*seatLimit)

	slog.InfoContext(ctx, "[seat-adjust] computed",
		"account_id", accountID, "active_count", activeCount, "seat_limit", limit)

	if activeCount > limit {
		// Downgrade: we must know which user to protect. Without identity
		// (e.g. webhook-triggered) we cannot safely choose, so skip deactivation.
		// The user-initiated path (SwitchPlan/ConfirmPlanSwitch) carries identity
		// and will handle deactivation correctly.
		identity, ok := appctx.GetIdentityFromContext(ctx)
		if !ok || identity == nil || identity.Actor == nil {
			slog.WarnContext(ctx, "[seat-adjust] skipping deactivation — no identity in context (webhook path)",
				"account_id", accountID, "active_count", activeCount, "seat_limit", limit)
			return
		}

		excess := int32(min(activeCount-limit, math.MaxInt32)) // #nosec G115 -- capped at MaxInt32
		s.deactivateExcessUsers(ctx, accountID, identity.Actor.ID, excess)
	} else if activeCount < limit {
		// Upgrade: reactivate disabled users up to the limit
		slotsAvailable := int32(min(limit-activeCount, math.MaxInt32)) // #nosec G115 -- capped at MaxInt32
		reactivated, apiErr := s.accountUserRepo.ReactivateUsers(ctx, accountID, slotsAvailable)
		if apiErr != nil {
			slog.WarnContext(ctx, "[seat-adjust] failed to reactivate users",
				"account_id", accountID, "error", apiErr.PublicMessage)
			return
		}
		if reactivated > 0 {
			slog.InfoContext(ctx, "[seat-adjust] reactivated users on upgrade",
				"account_id", accountID, "reactivated_count", reactivated, "seat_limit", limit)
		}
	}
}

// reactivateAllDisabledUsers reactivates all disabled users when the plan has no seat limit.
func (s *accountSvcImpl) reactivateAllDisabledUsers(ctx context.Context, accountID string) {
	// Use a large number to effectively reactivate all disabled users
	reactivated, apiErr := s.accountUserRepo.ReactivateUsers(ctx, accountID, 2147483647)
	if apiErr != nil {
		slog.WarnContext(ctx, "Failed to reactivate users on plan upgrade",
			"account_id", accountID, "error", apiErr.PublicMessage)
		return
	}
	if reactivated > 0 {
		slog.InfoContext(ctx, "Reactivated all disabled users on plan upgrade",
			"account_id", accountID, "reactivated_count", reactivated)
	}
}

// deactivateExcessUsers disables `excess` account users, protecting the requesting
// user (or admin fallback) from deactivation. Least-recently-used users are removed first.
// As a safety net, the keep user is explicitly re-activated after deactivation in case
// the SQL exclusion didn't match (e.g. identity propagation edge cases).
func (s *accountSvcImpl) deactivateExcessUsers(ctx context.Context, accountID, keepUserID string, excess int32) {
	slog.InfoContext(ctx, "[seat-adjust] deactivating excess users",
		"account_id", accountID, "keep_user_id", keepUserID, "excess", excess)

	deactivated, apiErr := s.accountUserRepo.DeactivateExcept(ctx, accountID, keepUserID, excess)
	if apiErr != nil {
		slog.WarnContext(ctx, "[seat-adjust] DeactivateExcept failed",
			"account_id", accountID, "keep_user_id", keepUserID, "error", apiErr.PublicMessage)
		return
	}

	slog.InfoContext(ctx, "[seat-adjust] DeactivateExcept result",
		"account_id", accountID, "keep_user_id", keepUserID, "rows_affected", deactivated)

	// Safety net: ensure the keep user is active regardless of what the bulk
	// deactivation did.
	if apiErr := s.accountUserRepo.EnsureActive(ctx, accountID, keepUserID); apiErr != nil {
		slog.WarnContext(ctx, "[seat-adjust] EnsureActive failed",
			"account_id", accountID, "keep_user_id", keepUserID, "error", apiErr.PublicMessage)
	}
}

func (s *accountSvcImpl) ClearAccountStripeCustomer(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.clear_account_stripe_customer")
	defer span.End()

	return s.accountRepo.ClearStripeCustomer(ctx, accountID)
}

func (s *accountSvcImpl) GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_by_stripe_customer_id")
	defer span.End()

	return s.accountRepo.GetByStripeCustomerID(ctx, stripeCustomerID)
}

func (s *accountSvcImpl) CompleteRegistration(ctx context.Context, input domain.CompleteRegistrationInput) (*domain.CompleteRegistrationOutput, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.complete_registration")
	defer span.End()

	// Generate production account ID before transaction
	accountID, genErr := id.GenID(id.AccountIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}

	// Enforce registration limits before creating accounts
	planCode := constants.PlanCode(input.PlanCode)
	limits := constants.GetRegistrationLimits(planCode)
	if limits.PublicLimit > 0 {
		count, countErr := s.accountRepo.CountNonSandboxByPlanCode(ctx, input.PlanCode)
		if countErr != nil {
			return nil, tracing.Trace(span, countErr)
		}
		if count >= limits.PublicLimit {
			return nil, tracing.Trace(span, apierror.NewRegistrationClosedError(
				fmt.Sprintf("Registration for the %s plan is currently at capacity.", input.PlanCode)))
		}
	}

	// Fetch admin role ID before transaction (read-only reference data)
	adminRoleID, apiErr := s.accountUserRepo.GetAdminRoleID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var sandboxAccountID string

	// Wrap all writes in a single transaction
	txErr := s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txRegRepo := f.NewRegistrationRepo()

		// 1. Create production account
		if apiErr := txRegRepo.CreateAccountForRegistration(txCtx, domain.CreateAccountParams{
			ID:                   accountID,
			Name:                 input.AccountData.AccountName,
			PlanCode:             input.PlanCode,
			StripeCustomerID:     input.StripeCustomerID,
			StripeSubscriptionID: input.StripeSubscriptionID,
		}); apiErr != nil {
			return apiErr
		}

		// 2. Link user to production account
		if apiErr := txRegRepo.CreateAccountUser(txCtx, accountID, input.UserID, adminRoleID); apiErr != nil {
			return apiErr
		}

		// 3. Create business address (log-only on error)
		if input.BusinessAddress != nil {
			if apiErr := txRegRepo.CreateBusinessAddress(txCtx, accountID, input.AccountData.AccountName, *input.BusinessAddress); apiErr != nil {
				slog.WarnContext(txCtx, "Failed to create business address during registration", "account_id", accountID, "error", apiErr.PublicMessage)
			}
		}

		// 4. Create portal for production account
		if apiErr := txRegRepo.CreateAccountPortal(txCtx, accountID); apiErr != nil {
			return apiErr
		}

		// 5. Create sandbox account (reuses same mediator logic as the create-sandbox endpoint)
		sandboxName := input.AccountData.AccountName + " Sandbox"
		meds := s.mediatorFactory.Build(f)
		sandbox, createErr := meds.Sandbox.Create(txCtx, accountID, input.UserID, sandboxName)
		if createErr != nil {
			return createErr
		}
		sandboxAccountID = sandbox.AccountID

		// 6. Enqueue seed message for the new sandbox account
		payloadJSON, err := json.Marshal(map[string]string{"account_id": sandboxAccountID})
		if err != nil {
			return apierror.NewInternalError(err, "Failed to marshal seed payload.")
		}

		seedMsg := contracts.AmqpMessage{Data: payloadJSON}
		if _, err := f.NewOutboxRepo().Create(txCtx, messaging.OutboxMessageInput{
			ServiceName: domain.ServiceName,
			MessageType: string(contracts.CoreCmdSeedSandbox),
			Destination: messaging.ApplicationExchange,
			RoutingKey:  string(contracts.CoreCmdSeedSandbox),
			Payload:     seedMsg,
		}); err != nil {
			return apierror.NewInternalError(err, "Failed to create seed outbox message.")
		}

		return nil
	})
	if txErr != nil {
		return nil, tracing.Trace(span, txErr)
	}

	return &domain.CompleteRegistrationOutput{
		AccountID: accountID,
		SandboxID: sandboxAccountID,
	}, nil
}
