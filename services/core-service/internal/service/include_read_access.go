package service

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// checkExternalReadAccess runs CheckReadAccess when the caller reads another account. Loading what an authorized request includes, a customer or supplier reads its counterparty's records, so the relation may run either way.
func checkExternalReadAccess(ctx context.Context, med domain.ReadAccessMed, identity *types.Identity) *apierror.APIError {
	if !identity.IsExternalTarget() {
		return nil
	}
	if identity.IsIncludeRead() {
		return med.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID)
	}
	return med.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID)
}
