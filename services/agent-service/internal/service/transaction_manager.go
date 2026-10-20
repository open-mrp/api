package service

import (
	"context"
	"database/sql"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
)

type TransactionManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *apierror.APIError) *apierror.APIError
}

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries) TransactionManager {
	return db.NewTransactionManager(sqlDB, queries, repository.NewRepoFactory)
}
