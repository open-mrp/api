package repository

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var userRepoTracer = tracing.GetTracer("core-service.user_repository")

type userRepoImpl struct {
	queries *sqlc.Queries
}

func NewUserRepo(queries *sqlc.Queries) domain.UserRepo {
	return &userRepoImpl{queries: queries}
}

func (r *userRepoImpl) FindByEmail(ctx context.Context, email string) (*domain.UserRecord, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.find_by_email")
	defer span.End()

	row, err := r.queries.FindUserByEmail(ctx, db.NullString(email))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapUserRow(row), nil
}

func (r *userRepoImpl) FindByUsername(ctx context.Context, username string) (*domain.UserRecord, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.find_by_username")
	defer span.End()

	row, err := r.queries.FindUserByUsername(ctx, db.NullString(username))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapUserRow(row), nil
}

func (r *userRepoImpl) CreateUser(ctx context.Context, id string, params domain.CreateUserRecordParams) *apierror.APIError {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.create")
	defer span.End()

	err := r.queries.InsertUser(ctx, sqlc.InsertUserParams{
		ID:             id,
		Name:           db.NullStringPtr(params.Name),
		Email:          db.NullStringPtr(params.Email),
		Username:       db.NullStringPtr(params.Username),
		HashedPassword: db.NullStringPtr(params.HashedPassword),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *userRepoImpl) UpdateProfile(ctx context.Context, userID string, name, email, username, imageURL *string, emailVerified *time.Time) *apierror.APIError {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.update_profile")
	defer span.End()

	err := r.queries.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:            userID,
		Name:          db.NullStringPtr(name),
		Email:         db.NullStringPtr(email),
		Username:      db.NullStringPtr(username),
		ImageUrl:      db.NullStringPtr(imageURL),
		EmailVerified: db.NullTimePtr(emailVerified),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *userRepoImpl) UpdatePassword(ctx context.Context, userID, hashedPassword string) *apierror.APIError {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.update_password")
	defer span.End()

	err := r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:             userID,
		HashedPassword: db.NullString(hashedPassword),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *userRepoImpl) GetHashedPassword(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.get_hashed_password")
	defer span.End()

	hp, err := r.queries.GetUserHashedPassword(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	result := db.StringFromNullString(hp)
	if result == nil {
		return "", nil
	}
	return *result, nil
}

func (r *userRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.UserRecord, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetUsersByIDs(ctx, sqlc.GetUsersByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.UserRecord, len(rows))
	for i, row := range rows {
		items[i] = &domain.UserRecord{
			ID:            row.ID,
			Email:         db.StringFromNullString(row.Email),
			Name:          db.StringFromNullString(row.Name),
			Username:      db.StringFromNullString(row.Username),
			EmailVerified: db.TimeFromNullTime(row.EmailVerified),
			ImageURL:      db.StringFromNullString(row.ImageUrl),
			StatusCode:    row.StatusCode,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
	}

	return items, nil
}

func (r *userRepoImpl) FindByID(ctx context.Context, userID string) (*domain.UserRecord, *apierror.APIError) {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.find_by_id")
	defer span.End()

	row, err := r.queries.FindUserByID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapUserRow(row), nil
}

func (r *userRepoImpl) UpdateImageURL(ctx context.Context, userID string, imageURL *string) *apierror.APIError {
	ctx, span := userRepoTracer.Start(ctx, "repository.user.update_image_url")
	defer span.End()

	err := r.queries.UpdateUserImageURL(ctx, sqlc.UpdateUserImageURLParams{
		ID:       userID,
		ImageUrl: db.NullStringPtr(imageURL),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func mapUserRow(row sqlc.User) *domain.UserRecord {
	return &domain.UserRecord{
		ID:             row.ID,
		Email:          db.StringFromNullString(row.Email),
		Name:           db.StringFromNullString(row.Name),
		Username:       db.StringFromNullString(row.Username),
		HashedPassword: db.StringFromNullString(row.HashedPassword),
		EmailVerified:  db.TimeFromNullTime(row.EmailVerified),
		ImageURL:       db.StringFromNullString(row.ImageUrl),
		StatusCode:     row.StatusCode,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
