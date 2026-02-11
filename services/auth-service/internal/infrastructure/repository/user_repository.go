package repository

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
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

	userModel, err := r.db.FindUserByIdentifier(ctx, sqlc.FindUserByIdentifierParams{
		ID:       identifier,
		Username: db.NullString(identifier),
		Email:    db.NullString(identifier),
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
	ctx, span := userRepoTracer.Start(ctx, "repository.user.updatePassword")
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
