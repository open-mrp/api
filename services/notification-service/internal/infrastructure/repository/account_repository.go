package repository

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var accountRepoTracer = tracing.GetTracer("notification-service.account_repository")

type accountRepoImpl struct {
	db *sqlc.Queries
}

func NewAccountRepo(db *sqlc.Queries) domain.AccountRepo {
	return &accountRepoImpl{db: db}
}

// IsSandbox reports whether the account is a sandbox one.
//
// An account that cannot be read is reported as not-sandbox: this gates a send, and failing closed
// would silently swallow every production invoice the moment the lookup broke. The caller logs the
// error so a persistent failure is visible rather than quietly degrading either way.
func (r *accountRepoImpl) IsSandbox(ctx context.Context, accountID string) (bool, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.is_sandbox")
	defer span.End()

	typeCode, err := r.db.GetAccountTypeCode(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return constants.AccountTypeCode(typeCode) == constants.AccountTypeCodeSandbox, nil
}
