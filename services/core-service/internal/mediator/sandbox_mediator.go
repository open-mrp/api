package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/calendarseed"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
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
// 2. Generate unique IDs for the new account and sandbox type.
// 3. Create the account record with sandbox type.
// 4. Create supporting records: business address, account-user link, portal, system products, and branding.
// 5. Insert the sandbox account record linking it to the owner.
// 6. Re-fetch and return the created sandbox with populated owner metadata.
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

	accountID, genErr := id.GenID(id.AccountIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}
	sandboxTypeID, genErr := id.GenID(id.SandboxAccountIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}

	if createErr := accountRepo.Create(ctx, accountID, name, domain.AccountTypeSandbox); createErr != nil {
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

	// A sandbox is where people try the product, so it has to resolve dates the same way production does.
	if seedErr := calendarseed.Seed(ctx, m.repos, accountID, time.Now()); seedErr != nil {
		return nil, tracing.Trace(span, seedErr)
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
