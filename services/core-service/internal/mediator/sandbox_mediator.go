package mediator

import (
	"context"
	"fmt"
	"math"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var sandboxMedTracer = tracing.GetTracer("core-service.sandbox_mediator")

type sandboxMedImpl struct {
	repos domain.RepoFactory
}

type SandboxMedConfig struct {
	// Repos (required) is the repository factory for sandbox operations.
	Repos domain.RepoFactory
}

func (c *SandboxMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sandbox mediator: repos is required")
	}
	return nil
}

func NewSandboxMed(config *SandboxMedConfig) domain.SandboxMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &sandboxMedImpl{
		repos: config.Repos,
	}
}

// Create provisions a new sandbox account under the given owner account.
//
// 1. Verify the owner is a production account (not already a sandbox).
// 2. Fetch the owner's plan code and sandbox limit.
// 3. Check the current sandbox count against the plan limit; reject if at capacity.
// 4. Generate unique IDs for the new account and sandbox type.
// 5. Create the account record with sandbox type and the owner's plan code.
// 6. Create supporting records: business address, account-user link, portal, system products, and branding.
// 7. Insert the sandbox account record linking it to the owner.
// 8. Re-fetch and return the created sandbox with populated owner metadata.
func (m *sandboxMedImpl) Create(ctx context.Context, ownerAccountID, userID, name string) (*domain.SandboxAccount, *apierror.APIError) {
	ctx, span := sandboxMedTracer.Start(ctx, "mediator.sandbox.create")
	defer span.End()

	accountRepo := m.repos.NewAccountRepo()

	accountContext, ctxErr := accountRepo.GetAccountContext(ctx, ownerAccountID)
	if ctxErr != nil {
		return nil, tracing.Trace(span, ctxErr)
	}
	if err := domain.RequireNotSandboxAccount(accountContext); err != nil {
		return nil, tracing.Trace(span, err)
	}

	planCode, planErr := accountRepo.GetPlanCode(ctx, ownerAccountID)
	if planErr != nil {
		return nil, tracing.Trace(span, planErr)
	}

	limit, limitErr := accountRepo.GetSandboxLimit(ctx, ownerAccountID)
	if limitErr != nil {
		if apierror.IsNotFound(limitErr) {
			return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Sandbox limit not configured for account plan."))
		}
		return nil, tracing.Trace(span, limitErr)
	}

	if limit != nil {
		count, countErr := m.repos.NewSandboxAccountRepo().CountByOwnerAccountID(ctx, ownerAccountID)
		if countErr != nil {
			return nil, tracing.Trace(span, countErr)
		}
		if count >= int64(*limit) {
			return nil, tracing.Trace(span, apierror.NewLimitExceededError(
				fmt.Sprintf("Maximum of %d sandbox accounts per account reached.", *limit),
			).WithQuota(*limit, int32(min(count, math.MaxInt32)), nil)) // #nosec G115 -- capped at MaxInt32
		}
	}

	accountID, genErr := id.GenID(id.AccountIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}
	sandboxTypeID, genErr := id.GenID(id.SandboxAccountIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}

	if createErr := accountRepo.Create(ctx, accountID, name, domain.AccountTypeSandbox, planCode); createErr != nil {
		return nil, tracing.Trace(span, createErr)
	}

	regRepo := m.repos.NewRegistrationRepo()

	if addrErr := regRepo.CreateBusinessAddress(ctx, accountID, name, domain.RegistrationAddress{
		Country: "US",
	}); addrErr != nil {
		return nil, tracing.Trace(span, addrErr)
	}

	adminRoleID, roleErr := m.repos.NewAccountUserRepo().GetAdminRoleID(ctx)
	if roleErr != nil {
		return nil, tracing.Trace(span, roleErr)
	}

	if linkErr := regRepo.CreateAccountUser(ctx, accountID, userID, adminRoleID); linkErr != nil {
		return nil, tracing.Trace(span, linkErr)
	}

	if portalErr := regRepo.CreateAccountPortal(ctx, accountID); portalErr != nil {
		return nil, tracing.Trace(span, portalErr)
	}

	if sysErr := regRepo.CreateSystemProducts(ctx, accountID); sysErr != nil {
		return nil, tracing.Trace(span, sysErr)
	}

	if brandErr := regRepo.CreateAccountBranding(ctx, accountID); brandErr != nil {
		return nil, tracing.Trace(span, brandErr)
	}

	sandboxRepo := m.repos.NewSandboxAccountRepo()
	if createErr := sandboxRepo.Create(ctx, sandboxTypeID, ownerAccountID, accountID); createErr != nil {
		return nil, tracing.Trace(span, createErr)
	}

	// Re-fetch to populate owner account name from the JOIN.
	created, fetchErr := sandboxRepo.FindByTypeID(ctx, sandboxTypeID, nil)
	if fetchErr != nil {
		return nil, tracing.Trace(span, fetchErr)
	}

	return created, nil
}

// Delete removes a sandbox account and its underlying account record.
//
// 1. Find the sandbox by type ID and verify it belongs to the requesting owner account.
// 2. Confirm the underlying account is actually a sandbox account.
// 3. Delete the sandbox account record from the sandbox table.
// 4. Delete the underlying account record.
// 5. Return the deleted account ID for downstream purge processing.
func (m *sandboxMedImpl) Delete(ctx context.Context, ownerAccountID, sandboxTypeID string) (string, *apierror.APIError) {
	ctx, span := sandboxMedTracer.Start(ctx, "mediator.sandbox.delete")
	defer span.End()

	sandbox, findErr := m.repos.NewSandboxAccountRepo().FindByTypeID(ctx, sandboxTypeID, nil)
	if findErr != nil {
		return "", tracing.Trace(span, findErr)
	}
	if sandbox.OwnerAccountID != ownerAccountID {
		return "", tracing.Trace(span, apierror.NewResourceNotFoundError("Sandbox not found."))
	}

	accountCtx, ctxErr := m.repos.NewAccountRepo().GetAccountContext(ctx, sandbox.AccountID)
	if ctxErr != nil {
		return "", tracing.Trace(span, ctxErr)
	}
	if err := domain.RequireSandboxAccount(accountCtx); err != nil {
		return "", tracing.Trace(span, err)
	}

	if deleteErr := m.repos.NewSandboxAccountRepo().DeleteByID(ctx, sandbox.ID); deleteErr != nil {
		return "", tracing.Trace(span, deleteErr)
	}

	if deleteErr := m.repos.NewAccountRepo().Delete(ctx, sandbox.AccountID); deleteErr != nil {
		return "", tracing.Trace(span, deleteErr)
	}

	return sandbox.AccountID, nil
}
