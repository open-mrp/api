package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	s3client "github.com/augno/api/shared/cloud/s3"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var userSvcTracer = tracing.GetTracer("core-service.user_service")

type userSvcImpl struct {
	repos            domain.RepoFactory
	mediatorFactory  domain.MediatorFactory
	txManager        TransactionManager
	s3Client         s3client.ObjectStore
	userPhotosBucket string
}

type UserSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// S3Client (required) is the object store client used for file storage.
	S3Client s3client.ObjectStore

	// UserPhotosBucket (optional; default: "") is the S3 bucket for user photos. It is not validated at construction.
	UserPhotosBucket string
}

func (c *UserSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("user service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("user service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("user service: tx manager is required")
	}
	if c.S3Client == nil {
		return fmt.Errorf("user service: s3 client is required")
	}
	return nil
}

func NewUserSvc(config *UserSvcConfig) domain.UserSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &userSvcImpl{
		repos:            config.Repos,
		mediatorFactory:  config.MediatorFactory,
		txManager:        config.TxManager,
		s3Client:         config.S3Client,
		userPhotosBucket: config.UserPhotosBucket,
	}
}

func (s *userSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *userSvcImpl) withTx(ctx context.Context, fn func(context.Context, *userSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &userSvcImpl{
			repos:            f,
			mediatorFactory:  s.mediatorFactory,
			txManager:        s.txManager,
			s3Client:         s.s3Client,
			userPhotosBucket: s.userPhotosBucket,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *userSvcImpl) GetUser(ctx context.Context, identifier string) (*domain.UserRecord, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.Actor == nil || identity.Actor.ID != identifier {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	userRepo := s.repos.NewUserRepo()

	// Try finding by ID first, then fall back to email and username. This matches the Dashboard behavior where the identifier can be an ID, email, or username.
	user, apiErr := userRepo.FindByID(ctx, identifier)
	if apiErr != nil && apierror.IsNotFound(apiErr) {
		user, apiErr = userRepo.FindByEmail(ctx, identifier)
	}
	if apiErr != nil && apierror.IsNotFound(apiErr) {
		user, apiErr = userRepo.FindByUsername(ctx, identifier)
	}
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	normalizeUserImageURL(user)

	return user, nil
}

func (s *userSvcImpl) BatchGetUsersByIDs(ctx context.Context, ids []string) ([]*domain.UserRecord, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	users, apiErr := s.repos.NewUserRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, user := range users {
		normalizeUserImageURL(user)
	}

	return users, nil
}

// normalizeUserImageURL converts legacy S3 signed URLs to the endpoint path format.
func normalizeUserImageURL(user *domain.UserRecord) {
	if user == nil || user.ImageURL == nil {
		return
	}
	if strings.Contains(*user.ImageURL, "augno-user-photos") {
		normalized := "/v1/core/users/" + user.ID + "/photo"
		user.ImageURL = &normalized
	}
}

func (s *userSvcImpl) UpdateUser(ctx context.Context, userID string, params domain.UpdateUserParams) (*domain.UserRecord, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.Actor == nil || identity.Actor.ID != userID {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionUpdate); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.UserRecord](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.UserRecord
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *userSvcImpl) *apierror.APIError {
			txUserRepo := txSvc.repos.NewUserRepo()

			old, apiErr := txUserRepo.FindByID(txCtx, userID)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txUserRepo.UpdateProfile(txCtx, userID, params.Name, nil, nil, params.ImageURL, params.EmailVerified); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txUserRepo.FindByID(txCtx, userID)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeUser,
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

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *userSvcImpl) UploadUserPhoto(ctx context.Context, userID string, file []byte, contentType string) *apierror.APIError {
	ctx, span := userSvcTracer.Start(ctx, "service.user.upload_photo")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if identity.Actor == nil || identity.Actor.ID != userID {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionUpdate); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountUserRepo := s.repos.NewAccountUserRepo()

	// The permission above only says the caller may manage users in their own account; it says
	// nothing about whether this user is one of them. Without this, any account could repoint
	// any user's photo at an image of its choosing.
	if _, apiErr := accountUserRepo.FindByAccountAndUserID(ctx, userID, identity.Target.AccountID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// The photo belongs to the user, not to the account that happened to upload it, so the key
	// is derived the same way on both sides. Deriving it from the calling account instead meant
	// a user who belongs to two accounts could upload a photo the read path never looked for.
	key, apiErr := s.userPhotoKey(ctx, userID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := s.s3Client.Upload(ctx, s.userPhotosBucket, key, bytes.NewReader(file), contentType); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	imageURL := "/v1/core/users/" + userID + "/photo"
	userRepo := s.repos.NewUserRepo()
	if apiErr := userRepo.UpdateImageURL(ctx, userID, &imageURL); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// userPhotoKey is where a user's photo lives, derived identically by the upload and the read.
// A user may belong to several accounts but has only one photo, so the account in the key is
// incidental — it just has to be the same one every time, or an upload lands somewhere the
// read never looks. Returns an empty key for a user who belongs to no account.
func (s *userSvcImpl) userPhotoKey(ctx context.Context, userID string) (string, *apierror.APIError) {
	accountID, apiErr := s.repos.NewAccountUserRepo().FindFirstAccountIDByUserID(ctx, userID)
	if apiErr != nil {
		return "", apiErr
	}
	if accountID == "" {
		return "", nil
	}

	return accountID + "/" + userID + ".png", nil
}

func (s *userSvcImpl) GetUserPhotoURL(ctx context.Context, userID string) (*string, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.get_photo_url")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// A user may always fetch their own photo; fetching another user's photo requires team_users:read. Mirrors GetUser / UpdateUser cross-user gating; without this any authenticated session could resolve any user's photo URL.
	if identity.Actor == nil || identity.Actor.ID != userID {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Same membership check as the upload path: a photo URL is a link to a person's face, and
	// resolving one for a user in another tenancy is not a read this caller is entitled to.
	accountUserRepo := s.repos.NewAccountUserRepo()
	if identity.Actor == nil || identity.Actor.ID != userID {
		if _, apiErr := accountUserRepo.FindByAccountAndUserID(ctx, userID, identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	key, apiErr := s.userPhotoKey(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if key == "" {
		return nil, nil
	}

	exists, _ := s.s3Client.FileExists(ctx, s.userPhotosBucket, key)
	if !exists {
		return nil, nil
	}

	url, apiErr := s.s3Client.GetPresignedURL(ctx, s.userPhotosBucket, key, time.Hour)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &url, nil
}
