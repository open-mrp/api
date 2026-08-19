package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/calendarseed"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	s3client "github.com/augno/api/shared/cloud/s3"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var accountSvcTracer = tracing.GetTracer("core-service.account_service")

type accountSvcImpl struct {
	accountRepo         domain.AccountRepo
	accountUserRepo     domain.AccountUserRepo
	accountRelationRepo domain.AccountRelationRepo
	rolePermissionRepo  domain.RolePermissionRepo
	roleRepo            domain.RoleRepo
	repos               domain.RepoFactory
	mediatorFactory     domain.MediatorFactory
	txManager           TransactionManager
	outboxRepo          messaging.OutboxRepo
	s3Client            s3client.ObjectStore
	accountPhotosBucket string
}

type AccountSvcConfig struct {
	// RepoFactory (required) is the repository factory.
	RepoFactory domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// S3Client (required) is the object store client used for file storage.
	S3Client s3client.ObjectStore

	// AccountPhotosBucket (optional; default: "") is the S3 bucket for account photos. It is not validated at construction.
	AccountPhotosBucket string
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
	if c.S3Client == nil {
		return fmt.Errorf("account service: s3 client is required")
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
		roleRepo:            config.RepoFactory.NewRoleRepo(),
		repos:               config.RepoFactory,
		mediatorFactory:     config.MediatorFactory,
		txManager:           config.TxManager,
		outboxRepo:          config.RepoFactory.NewOutboxRepo(),
		s3Client:            config.S3Client,
		accountPhotosBucket: config.AccountPhotosBucket,
	}
}

// GetAccountContext retrieves the account context (type, mode, and related metadata) for the given account.
func (s *accountSvcImpl) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_context")
	defer span.End()

	return s.accountRepo.GetAccountContext(ctx, accountID)
}

// GetUserAccountAccess checks whether a user has access to an account and returns their access details including role and permissions.
//
// 1. Look up the account-user link by user ID and account ID.
// 2. If not found, return nil with false (no access) without an error.
// 3. If the user has a role, fetch the role's permissions.
// 4. Return the access record with permissions and true (has access).
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
		RoleType:      accountUser.RoleType,
		Permissions:   permissions,
		LastUsedAt:    accountUser.LastUsedAt,
	}, true, nil
}

// GetRolePermissions returns the set of permissions granted by a role.
//
// 1. If the role ID is empty, return an empty permission map.
// 2. Query the role-permission repository for all permissions associated with the role.
func (s *accountSvcImpl) GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_role_permissions")
	defer span.End()

	if roleID == "" {
		return map[string]bool{}, nil
	}

	return s.rolePermissionRepo.FindByRoleID(ctx, roleID)
}

// GetRoleInfo returns the name and type code for a role.
//
// 1. If the role ID is empty, return a validation error.
// 2. Query the role repository by role ID and return the role info.
func (s *accountSvcImpl) GetRoleInfo(ctx context.Context, roleID string) (*domain.RoleInfo, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_role_info")
	defer span.End()

	if roleID == "" {
		return nil, apierror.NewInvariantViolationError("role ID is required")
	}

	return s.roleRepo.GetByID(ctx, roleID)
}

// GetAccountRelationByUserID finds the account relation linking a user to a target account.
//
//  1. Query by owner account (counterparty-side: e.g. customer user targeting merchant).
//  2. If actorAccountID is set, query by counterparty account constrained to that owner
//     (owner-side: e.g. merchant user in actor account targeting customer counterparty).
//     Without actorAccountID, the owner-side fallback is skipped so a user cannot match
//     via membership in an arbitrary account that happens to own a relation to the target.
//  3. If neither found, return nil without an error.
func (s *accountSvcImpl) GetAccountRelationByUserID(ctx context.Context, targetAccountID, actorAccountID, userID string) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_relation_by_user_id")
	defer span.End()

	// Try counterparty-side first (target is the relation owner, user belongs to counterparty).
	relation, apiErr := s.accountRelationRepo.FindByOwnerAccountAndUserID(ctx, targetAccountID, userID)
	if apiErr != nil {
		if apiErr.Code != apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if relation != nil {
		return relation, nil
	}

	// Owner-side requires a verified actor account; the relation's owner must equal it.
	if actorAccountID == "" {
		return nil, nil
	}

	// Try owner-side (target is the counterparty, user belongs to the actor account that owns the relation).
	relation, apiErr = s.accountRelationRepo.FindByCounterpartyAccountAndUserID(ctx, targetAccountID, actorAccountID, userID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return relation, nil
}

// GetAccountRelationByAPIKeyID finds the account relation linking an API key to a target account.
//
// 1. Query by owner account (counterparty-side: e.g. customer API key targeting merchant).
// 2. If not found, query by counterparty account (owner-side: e.g. merchant API key targeting customer).
// 3. If neither found, return nil without an error.
func (s *accountSvcImpl) GetAccountRelationByAPIKeyID(ctx context.Context, targetAccountID string, apiKeyID int64) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_relation_by_api_key_id")
	defer span.End()

	// Try counterparty-side first (target is the relation owner, API key belongs to counterparty).
	relation, apiErr := s.accountRelationRepo.FindByOwnerAccountAndAPIKeyID(ctx, targetAccountID, apiKeyID)
	if apiErr != nil {
		if apiErr.Code != apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if relation != nil {
		return relation, nil
	}

	// Try owner-side (target is the counterparty, API key belongs to the relation owner).
	relation, apiErr = s.accountRelationRepo.FindByCounterpartyAccountAndAPIKeyID(ctx, targetAccountID, apiKeyID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return relation, nil
}

// MarkAccountUserUsed updates the last-used timestamp on an account-user link to the current time.
//
// 1. Update the last_used_at field for the account-user record to now (UTC).
func (s *accountSvcImpl) MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.mark_account_user_used")
	defer span.End()

	return s.accountUserRepo.UpdateLastUsedAt(ctx, accountUserID, time.Now().UTC())
}

// ListUserAccountAffiliations returns all accounts a user belongs to and the ID of the most recently used one.
//
// 1. Fetch all account affiliations for the user.
// 2. Determine which account was last used by the user.
// 3. Return the affiliations and the last-used account ID (nil if none).
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

// GetAdminRole returns the role ID for the built-in admin role.
//
// 1. Query the account-user repository for the admin role ID.
func (s *accountSvcImpl) GetAdminRole(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_admin_role")
	defer span.End()

	return s.accountUserRepo.GetAdminRoleID(ctx)
}

// UpdateAccountSubscription persists subscription changes for an account and reconciles seats.
//
// 1. Fetch the account's current plan code to detect plan changes.
// 2. Resolve the plan type ID from the new plan code.
// 3. Update the account's subscription record (status, plan, Stripe IDs, period end).
// 4. If the plan changed, publish an admin notification email via the outbox.
// 5. Adjust active users to match the new plan's seat limit (deactivate or reactivate).
//
// Side effects:
//   - Sends a plan-change alert email to admins on plan transitions.
//   - Deactivates or reactivates account users based on the new seat limit.
func (s *accountSvcImpl) UpdateAccountSubscription(ctx context.Context, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.update_account_subscription")
	defer span.End()

	var oldPlanCode constants.PlanCode
	var planTypeID *string

	if planCode != "" {
		oldPlanCode, _ = s.accountRepo.GetPlanCode(ctx, accountID)

		resolved, apiErr := s.accountRepo.GetPlanTypeIDByCode(ctx, planCode)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		planTypeID = &resolved
	}

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *accountSvcImpl) *apierror.APIError {
		if apiErr := txSvc.accountRepo.UpdateSubscription(txCtx, accountID, status, planCode, planTypeID, stripeSubID, periodEnd, stripeCustomerID, billingProfileID, billingCadenceID, pricingPlanSubscriptionID, servicingStatus, collectionStatus); apiErr != nil {
			return apiErr
		}

		// When downgrading to Free, clear subscription fields that COALESCE won't null out.
		if planCode == string(constants.PlanCodeFree) {
			if apiErr := txSvc.accountRepo.ClearPricingPlanSubscription(txCtx, accountID); apiErr != nil {
				return apiErr
			}
		}

		if planCode != "" && oldPlanCode != "" && string(oldPlanCode) != planCode {
			txSvc.publishPlanChangeAlert(txCtx, accountID, string(oldPlanCode), planCode)
		}

		var changes []audit.FieldChange
		if planCode != "" && string(oldPlanCode) != planCode {
			changes = append(changes, audit.NewFieldChange("plan_code", string(oldPlanCode), planCode))
		}
		if status != nil {
			changes = append(changes, audit.NewFieldChange("subscription_status", nil, *status))
		}
		if servicingStatus != nil {
			changes = append(changes, audit.NewFieldChange("servicing_status", nil, *servicingStatus))
		}
		if collectionStatus != nil {
			changes = append(changes, audit.NewFieldChange("collection_status", nil, *collectionStatus))
		}

		if len(changes) > 0 {
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAccount,
				ResourceID:   accountID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}
		}

		return nil
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if planCode != "" {
		s.adjustAccountSeats(ctx, accountID, planCode)
	}

	return nil
}

// publishPlanChangeAlert sends a best-effort admin email when an account's plan changes.
func (s *accountSvcImpl) publishPlanChangeAlert(ctx context.Context, accountID, oldPlan, newPlan string) {
	emailData := messaging.EmailSendData{
		To:         []string{"dev@augno.com"},
		Subject:    fmt.Sprintf("[Plan Change] %s → %s", oldPlan, newPlan),
		TemplateID: constants.EmailTemplatePlanChangeAlert,
		Params: map[string]any{
			"AccountID":    accountID,
			"PreviousPlan": oldPlan,
			"NewPlan":      newPlan,
		},
	}

	emailJSON, err := json.Marshal(emailData)
	if err != nil {
		slog.WarnContext(ctx, "Failed to marshal plan change alert email data", "error", err, "account_id", accountID)
		return
	}

	emailMsg := contracts.AmqpMessage{Data: emailJSON}
	if _, err := s.outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdSendEmail),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdSendEmail),
		Payload:     emailMsg,
	}); err != nil {
		slog.WarnContext(ctx, "Failed to create plan change alert outbox message", "error", err, "account_id", accountID)
	}
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
		// Downgrade: we must know which user to protect. Without identity (e.g. webhook-triggered) we cannot safely choose, so skip deactivation. The user-initiated path (SwitchPlan/ConfirmPlanSwitch) carries identity and will handle deactivation correctly.
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

// deactivateExcessUsers disables `excess` account users, protecting the requesting user (or admin fallback) from deactivation. Least-recently-used users are removed first. As a safety net, the keep user is explicitly re-activated after deactivation in case the SQL exclusion didn't match (e.g. identity propagation edge cases).
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

	// Safety net: ensure the keep user is active regardless of what the bulk deactivation did.
	if apiErr := s.accountUserRepo.EnsureActive(ctx, accountID, keepUserID); apiErr != nil {
		slog.WarnContext(ctx, "[seat-adjust] EnsureActive failed",
			"account_id", accountID, "keep_user_id", keepUserID, "error", apiErr.PublicMessage)
	}
}

// ClearAccountStripeCustomer removes the Stripe customer ID from an account record.
//
// 1. Clear the Stripe customer reference in the account repository.
func (s *accountSvcImpl) ClearAccountStripeCustomer(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.clear_account_stripe_customer")
	defer span.End()

	return s.accountRepo.ClearStripeCustomer(ctx, accountID)
}

// GetAccountByStripeCustomerID looks up an account by its associated Stripe customer ID.
//
// 1. Query the account repository by Stripe customer ID and return the account ID and plan code.
func (s *accountSvcImpl) GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_by_stripe_customer_id")
	defer span.End()

	return s.accountRepo.GetByStripeCustomerID(ctx, stripeCustomerID)
}

// CompleteRegistration provisions a new production account with all supporting resources inside a single transaction.
//
//  1. Generate a new account ID.
//  2. Enforce registration limits for the selected plan.
//  3. Fetch the admin role ID for linking the user.
//  4. Within a transaction:
//     a. Create the production account record.
//     b. Link the registering user to the account with the admin role.
//     c. Create the business address (best-effort).
//     d. Create the account portal, system products, and branding records.
//     e. Create a sandbox account via the sandbox mediator.
//     f. Enqueue a seed-data message for the sandbox.
//     g. Enqueue an admin notification email for the new registration.
//  5. Return the production account ID and sandbox account ID.
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

		// 4b. Create system products (shipping + credit) for the production account
		if apiErr := txRegRepo.CreateSystemProducts(txCtx, accountID); apiErr != nil {
			return apiErr
		}

		// 4c. Create account branding for the production account
		if apiErr := txRegRepo.CreateAccountBranding(txCtx, accountID); apiErr != nil {
			return apiErr
		}

		// 4d. Seed the shipping and receiving calendars, so the account's first order is not committed to a Saturday or a federal holiday.
		if apiErr := calendarseed.Seed(txCtx, f, accountID, time.Now()); apiErr != nil {
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

		// 7. Enqueue admin notification for the new registration
		emailData := messaging.EmailSendData{
			To:         []string{"dev@augno.com"},
			Subject:    fmt.Sprintf("[New Registration] %s", input.AccountData.AccountName),
			TemplateID: constants.EmailTemplateNewRegistrationAlert,
			Params: map[string]any{
				"AccountName": input.AccountData.AccountName,
				"AccountID":   accountID,
				"PlanCode":    input.PlanCode,
				"UserID":      input.UserID,
			},
		}

		emailJSON, err := json.Marshal(emailData)
		if err != nil {
			return apierror.NewInternalError(err, "Failed to marshal registration alert email data.")
		}

		emailMsg := contracts.AmqpMessage{Data: emailJSON}
		if _, err := f.NewOutboxRepo().Create(txCtx, messaging.OutboxMessageInput{
			ServiceName: domain.ServiceName,
			MessageType: string(contracts.NotificationCmdSendEmail),
			Destination: messaging.ApplicationExchange,
			RoutingKey:  string(contracts.NotificationCmdSendEmail),
			Payload:     emailMsg,
		}); err != nil {
			return apierror.NewInternalError(err, "Failed to create registration alert outbox message.")
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

func (s *accountSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *accountSvcImpl) withTx(ctx context.Context, fn func(context.Context, *accountSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &accountSvcImpl{
			accountRepo:         f.NewAccountRepo(),
			accountUserRepo:     f.NewAccountUserRepo(),
			accountRelationRepo: f.NewAccountRelationRepo(),
			rolePermissionRepo:  f.NewRolePermissionRepo(),
			roleRepo:            f.NewRoleRepo(),
			repos:               f,
			mediatorFactory:     s.mediatorFactory,
			txManager:           s.txManager,
			outboxRepo:          f.NewOutboxRepo(),
			s3Client:            s.s3Client,
			accountPhotosBucket: s.accountPhotosBucket,
		}
		return fn(txCtx, txSvc)
	})
}

// agentCapUpdate is used as the idempotency cache payload for UpdateAgentSpendingCap. A wrapper struct is required so that a nil cap is round-tripped correctly through JSON.
type agentCapUpdate struct {
	CapCents *int64 `json:"cap_cents"`
}

func (s *accountSvcImpl) UpdateAgentSpendingCap(ctx context.Context, capCents *int64) (*int64, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.update_agent_spending_cap")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[agentCapUpdate](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Data != nil {
			return cached.Data.CapCents, cached.Error
		}
		return nil, cached.Error

	case domain.RecoveryPointStarted:
		var result *int64
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountSvcImpl) *apierror.APIError {
			oldCap, apiErr := txSvc.accountRepo.GetAgentSpendingCap(txCtx, accountID)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txSvc.accountRepo.UpdateAgentSpendingCap(txCtx, accountID, capCents); apiErr != nil {
				return apiErr
			}

			var changes []audit.FieldChange
			if !reflect.DeepEqual(oldCap, capCents) {
				changes = append(changes, audit.NewFieldChange("agent_monthly_spending_cap_cents", oldCap, capCents))
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAccount,
				ResourceID:   accountID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			result = capCents
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, agentCapUpdate{CapCents: capCents})
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// GetAccount returns the full account with branding and portal sub-resources.
func (s *accountSvcImpl) GetAccount(ctx context.Context, accountID string) (*domain.Account, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if accountID != identity.Target.AccountID {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You can only access your own account."))
	}

	account, apiErr := s.accountRepo.GetByID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	s.signBranding(ctx, account)
	return account, nil
}

// BatchGetAccountsByIDs returns accounts matching the input IDs that the caller is authorized to read. The caller can always read their own target account, plus any requested account they have an account_relation to (customer/supplier), so relationship-scoped includes (e.g. ContactMatch.account for a customer/supplier match) hydrate cross-account. IDs the caller neither owns nor relates to are silently dropped (matching the "missing IDs are absent" loader contract used by the api-gateway resourcekit resolver).
func (s *accountSvcImpl) BatchGetAccountsByIDs(ctx context.Context, ids []string) ([]*domain.Account, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	// Authorization: always allow the caller's own target account; additionally allow any requested id the caller has an account_relation to (customer/supplier), so relationship-scoped includes hydrate cross-account. Everything else is silently dropped (the resolver treats absence as "field stays nil").
	allowed := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == identity.Target.AccountID {
			allowed = append(allowed, id)
			continue
		}
		hasRelation, apiErr := s.accountRelationRepo.HasRelation(ctx, identity.Target.AccountID, id)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if hasRelation {
			allowed = append(allowed, id)
		}
	}
	if len(allowed) == 0 {
		return nil, nil
	}
	accounts, apiErr := s.accountRepo.GetByIDs(ctx, allowed)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, account := range accounts {
		s.signBranding(ctx, account)
	}
	return accounts, nil
}

// signBranding replaces an account's stored branding asset keys with presigned URLs in place. The column holds the object key the upload wrote, so a resource that hands it out unsigned advertises a URL field carrying something no client can load.
func (s *accountSvcImpl) signBranding(ctx context.Context, account *domain.Account) {
	if account == nil || account.Branding == nil {
		return
	}
	account.Branding.LogoURL = s.presignedBrandingURL(ctx, account.Branding.LogoURL)
	account.Branding.FaviconURL = s.presignedBrandingURL(ctx, account.Branding.FaviconURL)
}

// presignedBrandingURL converts a stored branding asset S3 key (logo or favicon) into a presigned download URL. Branding must still render without the asset, so it returns nil (best-effort) when there is no key or signing fails, and passes through values that are already absolute URLs.
func (s *accountSvcImpl) presignedBrandingURL(ctx context.Context, key *string) *string {
	if key == nil || *key == "" {
		return nil
	}
	if strings.HasPrefix(*key, "http://") || strings.HasPrefix(*key, "https://") {
		return key
	}
	url, apiErr := s.s3Client.GetPresignedURL(ctx, s.accountPhotosBucket, *key, time.Hour)
	if apiErr != nil {
		slog.WarnContext(ctx, "Failed to presign account branding URL", "error", apiErr)
		return nil
	}
	return &url
}

// GetAccountBySlug returns a minimal public account by portal slug (unauthenticated). When the account has a verified custom portal domain it is included so callers (e.g. auth email links) can target it.
func (s *accountSvcImpl) GetAccountBySlug(ctx context.Context, slug string) (*domain.PublicAccountBySlug, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_by_slug")
	defer span.End()

	account, apiErr := s.accountRepo.GetBySlug(ctx, slug)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	account.LogoURL = s.presignedBrandingURL(ctx, account.LogoURL)
	account.FaviconURL = s.presignedBrandingURL(ctx, account.FaviconURL)

	portalDomain, apiErr := s.repos.NewPortalDomainRepo().GetByAccountID(ctx, account.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if portalDomain != nil && portalDomain.Status == constants.PortalDomainStatusVerified {
		account.PortalDomain = &portalDomain.Domain
	}

	return account, nil
}

// GetPortalProfileBySlug returns the authenticated seller portal profile (identity + letterhead address) by portal slug. Requires an assigned actor; the address is the seller account's own default billing address, resolved server-side so no cross-account scope is needed.
func (s *accountSvcImpl) GetPortalProfileBySlug(ctx context.Context, slug string) (*domain.PortalProfile, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_portal_profile_by_slug")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	account, apiErr := s.accountRepo.GetBySlug(ctx, slug)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	profile := &domain.PortalProfile{
		ID:           account.ID,
		Name:         account.Name,
		Slug:         account.Slug,
		LogoURL:      s.presignedBrandingURL(ctx, account.LogoURL),
		FaviconURL:   s.presignedBrandingURL(ctx, account.FaviconURL),
		SupportEmail: account.SupportEmail,
	}

	if account.DefaultBillingAddressID != nil && *account.DefaultBillingAddressID != "" {
		addrs, apiErr := s.repos.NewAddressRepo().GetByIDs(ctx, account.ID, []string{*account.DefaultBillingAddressID})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(addrs) > 0 {
			profile.Address = addrs[0]
		}
	}

	return profile, nil
}

// UpdateAccount partially updates an account's name, branding fields, and/or portal slug.
func (s *accountSvcImpl) UpdateAccount(ctx context.Context, params domain.UpdateAccountParams) (*domain.Account, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.update_account")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if params.AccountID != identity.Target.AccountID {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You can only update your own account."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Account](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		// Signed on the way out, not before caching: a replay hours later must not hand back a link that expired with the original response.
		s.signBranding(ctx, cached.Data)
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Account
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountRepo()

			old, apiErr := txRepo.GetByID(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				if apiErr := txRepo.UpdateName(txCtx, params.AccountID, *params.Name); apiErr != nil {
					return apiErr
				}
			}

			if params.HasBrandingUpdates() {
				if apiErr := txRepo.UpdateBranding(txCtx, params.AccountID, params); apiErr != nil {
					return apiErr
				}
			}

			if params.Slug != nil {
				exists, apiErr := txRepo.ExistsPortalSlug(txCtx, *params.Slug, params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A portal with this slug already exists.", "slug")
				}

				if apiErr := txRepo.UpdatePortalSlug(txCtx, params.AccountID, *params.Slug); apiErr != nil {
					return apiErr
				}
			}

			updated, apiErr := txRepo.GetByID(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAccount,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		s.signBranding(ctx, result)
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// UploadAccountPhoto uploads an account logo to S3 and updates the branding record.
func (s *accountSvcImpl) UploadAccountPhoto(ctx context.Context, accountID string, file []byte, contentType string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.upload_account_photo")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if accountID != identity.Target.AccountID {
		return tracing.Trace(span, apierror.NewAuthorizationError("You can only update your own account."))
	}

	if contentType == "" {
		contentType = "image/png"
	}

	s3Key := accountID + "/logo.png"

	if apiErr := s.s3Client.Upload(ctx, s.accountPhotosBucket, s3Key, bytes.NewReader(file), contentType); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := s.accountRepo.UpdateBrandingLogoURL(ctx, accountID, s3Key); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// GetAccountLogoURL returns a presigned S3 URL for the account's logo.
func (s *accountSvcImpl) GetAccountLogoURL(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_logo_url")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	logoKey, apiErr := s.accountRepo.GetBrandingLogoKey(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if logoKey == nil {
		return nil, nil
	}

	exists, apiErr := s.s3Client.FileExists(ctx, s.accountPhotosBucket, *logoKey)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !exists {
		return nil, nil
	}

	url, apiErr := s.s3Client.GetPresignedURL(ctx, s.accountPhotosBucket, *logoKey, time.Hour)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &url, nil
}

// UploadAccountFavicon uploads a customer-portal favicon to S3 and updates the branding record.
func (s *accountSvcImpl) UploadAccountFavicon(ctx context.Context, accountID string, file []byte, contentType string) *apierror.APIError {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.upload_account_favicon")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if accountID != identity.Target.AccountID {
		return tracing.Trace(span, apierror.NewAuthorizationError("You can only update your own account."))
	}

	if contentType == "" {
		contentType = "image/png"
	}

	s3Key := accountID + "/favicon.png"

	if apiErr := s.s3Client.Upload(ctx, s.accountPhotosBucket, s3Key, bytes.NewReader(file), contentType); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := s.accountRepo.UpdateBrandingFaviconURL(ctx, accountID, s3Key); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// GetAccountFaviconURL returns a presigned S3 URL for the account's customer-portal favicon.
func (s *accountSvcImpl) GetAccountFaviconURL(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := accountSvcTracer.Start(ctx, "service.account.get_account_favicon_url")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	faviconKey, apiErr := s.accountRepo.GetBrandingFaviconKey(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if faviconKey == nil {
		return nil, nil
	}

	exists, apiErr := s.s3Client.FileExists(ctx, s.accountPhotosBucket, *faviconKey)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !exists {
		return nil, nil
	}

	url, apiErr := s.s3Client.GetPresignedURL(ctx, s.accountPhotosBucket, *faviconKey, time.Hour)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &url, nil
}
