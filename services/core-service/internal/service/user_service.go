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
	Repos            domain.RepoFactory
	MediatorFactory  domain.MediatorFactory
	TxManager        TransactionManager
	S3Client         s3client.ObjectStore
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

	// Try finding by ID first, then fall back to email and username.
	// This matches the Dashboard behavior where the identifier can be an ID, email, or username.
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

	key := identity.Target.AccountID + "/" + userID + ".png"
	apiErr := s.s3Client.Upload(ctx, s.userPhotosBucket, key, bytes.NewReader(file), contentType)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	imageURL := "/v1/core/users/" + userID + "/photo"
	userRepo := s.repos.NewUserRepo()
	if apiErr := userRepo.UpdateImageURL(ctx, userID, &imageURL); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (s *userSvcImpl) GetUserPhotoURL(ctx context.Context, userID string) (*string, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.get_photo_url")
	defer span.End()

	accountUserRepo := s.repos.NewAccountUserRepo()
	accountID, apiErr := accountUserRepo.FindFirstAccountIDByUserID(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if accountID == "" {
		return nil, nil
	}

	key := accountID + "/" + userID + ".png"
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
