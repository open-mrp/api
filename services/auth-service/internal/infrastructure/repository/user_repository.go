package repository

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
	"go.opentelemetry.io/otel/trace"
)

var userRepoTracer = tracing.GetTracer("auth-service.user_repository")

type userRepoImpl struct {
	db *sqlc.Queries
}

func NewUserRepo(db *sqlc.Queries) domain.UserRepo {
	return &userRepoImpl{db: db}
}

func (r *userRepoImpl) Find(ctx context.Context, identifier string) (*types.User, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.find")
	defer span.End()

	// Every authenticated request funnels through this lookup, so a dropped
	// database connection (Vitess tablet failover, killed connection) must not
	// surface as a 500. The read is idempotent, making a short retry safe.
	var userModel sqlc.User
	err := db.WithConnRetry(ctx, nil, "user.find_by_identifier", func() error {
		var findErr error
		userModel, findErr = r.db.FindUserByIdentifier(ctx, sqlc.FindUserByIdentifierParams{
			ID:       identifier,
			Username: db.NullString(identifier),
			Email:    db.NullString(identifier),
		})
		return findErr
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &types.User{
		ID:             userModel.ID,
		Email:          db.StringFromNullString(userModel.Email),
		Name:           db.StringFromNullString(userModel.Name),
		Username:       db.StringFromNullString(userModel.Username),
		HashedPassword: db.StringFromNullString(userModel.HashedPassword),
		EmailVerified:  db.TimeFromNullTime(userModel.EmailVerified),
		ImageUrl:       db.StringFromNullString(userModel.ImageUrl),
		StatusCode:     string(types.UserStatusActive),
		CreatedAt:      userModel.CreatedAt,
		UpdatedAt:      userModel.UpdatedAt,
	}, nil
}

func (r *userRepoImpl) Create(ctx context.Context, userID, email, name, hashedPassword string) (*types.User, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.create")
	defer span.End()

	err := r.db.CreateUser(ctx, sqlc.CreateUserParams{
		ID:             userID,
		Email:          db.NullString(email),
		Name:           db.NullString(name),
		HashedPassword: db.NullString(hashedPassword),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Find(ctx, userID)
}

func (r *userRepoImpl) UpdatePassword(ctx context.Context, userID string, hashedPassword string) *apierror.APIError {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.update_password")
	defer span.End()

	err := r.db.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:             userID,
		HashedPassword: db.NullString(hashedPassword),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *userRepoImpl) FindLastUsedAccountID(ctx context.Context, userID string) (*string, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.find_last_used_account_id")
	defer span.End()

	accountID, err := r.db.FindLastUsedAccountID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return &accountID, nil
}

func (r *userRepoImpl) FindAccountUser(ctx context.Context, accountID, accountUserID string) (*domain.SigningAccountUser, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.find_account_user")
	defer span.End()

	row, err := r.db.FindAccountUserInAccount(ctx, sqlc.FindAccountUserInAccountParams{
		ID:        accountUserID,
		AccountID: accountID,
	})
	return signingAccountUserFromRow(span, row, err)
}

func (r *userRepoImpl) FindAccountUserByUserID(ctx context.Context, accountID, userID string) (*domain.SigningAccountUser, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.find_account_user_by_user_id")
	defer span.End()

	row, err := r.db.FindAccountUserByUserIDInAccount(ctx, sqlc.FindAccountUserByUserIDInAccountParams{
		UserID:    userID,
		AccountID: accountID,
	})
	return signingAccountUserFromRow(span, sqlc.FindAccountUserInAccountRow(row), err)
}

func signingAccountUserFromRow(span trace.Span, row sqlc.FindAccountUserInAccountRow, err error) (*domain.SigningAccountUser, *apierror.APIError) {
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Account user not found.")
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.SigningAccountUser{
		ID:           row.ID,
		UserID:       row.UserID,
		AccountID:    row.AccountID,
		DepartmentID: db.StringFromNullString(row.DepartmentID),
		StatusCode:   row.StatusCode,
		Name:         db.StringFromNullString(row.Name),
		Username:     db.StringFromNullString(row.Username),
	}, nil
}
