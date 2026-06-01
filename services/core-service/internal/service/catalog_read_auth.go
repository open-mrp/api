package service

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
	"go.opentelemetry.io/otel/trace"
)

// authorizeCatalogBatchRead checks that the caller may read catalog resources in
// the target account via batch-get loaders. Internal actors require the supplied
// permission check; customer and supplier actors require a valid counterparty
// relationship when accessing an external target account.
func authorizeCatalogBatchRead(
	ctx context.Context,
	identity *types.Identity,
	span trace.Span,
	meds domain.Mediators,
	internalPermissionCheck func() *apierror.APIError,
) *apierror.APIError {
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if identity.IsInternalActor() {
		if apiErr := internalPermissionCheck(); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}
	if identity.IsExternalTarget() {
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}
