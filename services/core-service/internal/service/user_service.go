package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	s3client "github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/imageupload"
	"github.com/open-mrp/api/shared/tracing"
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

	if apiErr := s.checkUserInTargetAccount(ctx, identity, user.ID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	normalizeUserImageURL(user)

	return user, nil
}

// checkUserInTargetAccount refuses a user who is neither the caller nor a member of the account the
// caller acts in. The team permission says the caller may manage users in their own account, not
// that the user named is one of them; a user outside it is reported as not found, so the answer does
// not reveal that an ID or email belongs to someone elsewhere.
func (s *userSvcImpl) checkUserInTargetAccount(ctx context.Context, identity *types.Identity, userID string) *apierror.APIError {
	if identity.Actor != nil && identity.Actor.ID == userID {
		return nil
	}
	if _, apiErr := s.repos.NewAccountUserRepo().FindByAccountAndUserID(ctx, userID, identity.Target.AccountID); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return apierror.NewResourceNotFoundError("User not found.")
		}
		return apiErr
	}
	return nil
}

func (s *userSvcImpl) BatchGetUsersByIDs(ctx context.Context, ids []string) ([]*domain.UserRecord, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	// The users returned are members of the target account, so reading them is reading that account's users: a merchant
	// reaching a customer's or supplier's contacts needs that domain's read permission, not the one for its own team.
	if apiErr := checkSellerStaff(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAccountUserReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.IsExternalTarget() {
		if apiErr := s.mediators().ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
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

// externalUserPhoto returns imageURL when it is an image hosted elsewhere, such as the avatar an
// identity provider supplied at sign-up, which a browser loads as is. A photo uploaded here is
// stored as a path or as a URL into the photos bucket, and only a presigned URL serves it.
func externalUserPhoto(imageURL *string) (string, bool) {
	if imageURL == nil || strings.Contains(*imageURL, "augno-user-photos") {
		return "", false
	}
	if strings.HasPrefix(*imageURL, "https://") || strings.HasPrefix(*imageURL, "http://") {
		return *imageURL, true
	}
	return "", false
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

	if apiErr := s.checkUserInTargetAccount(ctx, identity, userID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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

func (s *userSvcImpl) UploadUserPhoto(ctx context.Context, userID string, file []byte) *apierror.APIError {
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
		return tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if apiErr := s.checkUserInTargetAccount(ctx, identity, userID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	contentType, apiErr := imageupload.Photo(file)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	key := userPhotoKey(identity, userID)

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

// userPhotoKey is where a user's photo lives: {account}/{user}.png under the account the request
// targets. It is the key /me and the team list sign (tenancy and account-user services) and the one
// the dashboard API wrote, so a photo is per account, as it always was. Keying the upload and this read
// by the user's first account instead meant a user in two accounts uploaded where those reads never
// looked.
func userPhotoKey(identity *types.Identity, userID string) string {
	return identity.Target.AccountID + "/" + userID + ".png"
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

	// A photo URL is a link to a person's face, and resolving one for a user in another tenancy is
	// not a read this caller is entitled to.
	if apiErr := s.checkUserInTargetAccount(ctx, identity, userID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, nil
	}
	key := userPhotoKey(identity, userID)

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
