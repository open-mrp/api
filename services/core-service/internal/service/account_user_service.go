package service

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/event"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	s3client "github.com/augno/api/shared/cloud/s3"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var accountUserSvcTracer = tracing.GetTracer("core-service.account_user_service")

type accountUserSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	txManager             TransactionManager
	notificationPublisher domain.NotificationPublisher
	billingPublisher      domain.BillingPublisher
	s3Client              s3client.ObjectStore
	userPhotosBucket      string
}

type AccountUserSvcConfig struct {
	Repos                 domain.RepoFactory
	MediatorFactory       domain.MediatorFactory
	TxManager             TransactionManager
	NotificationPublisher domain.NotificationPublisher
	BillingPublisher      domain.BillingPublisher
	S3Client              s3client.ObjectStore
	UserPhotosBucket      string
	PlatformMode          constants.PlatformMode
}

func (c *AccountUserSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("account user service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("account user service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("account user service: tx manager is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("account user service: notification publisher is required")
	}
	if c.BillingPublisher == nil {
		return fmt.Errorf("account user service: billing publisher is required")
	}
	if c.S3Client == nil {
		return fmt.Errorf("account user service: s3 client is required")
	}
	if !c.PlatformMode.IsTest() && c.UserPhotosBucket == "" {
		return fmt.Errorf("account user service: user photos bucket is required")
	}
	return nil
}

func NewAccountUserSvc(config *AccountUserSvcConfig) domain.AccountUserSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountUserSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		txManager:             config.TxManager,
		notificationPublisher: config.NotificationPublisher,
		billingPublisher:      config.BillingPublisher,
		s3Client:              config.S3Client,
		userPhotosBucket:      config.UserPhotosBucket,
	}
}

func (s *accountUserSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *accountUserSvcImpl) withTx(ctx context.Context, fn func(context.Context, *accountUserSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &accountUserSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			txManager:             s.txManager,
			notificationPublisher: s.notificationPublisher,
			billingPublisher:      s.billingPublisher,
			s3Client:              s.s3Client,
			userPhotosBucket:      s.userPhotosBucket,
		}
		return fn(txCtx, txSvc)
	})
}

// resolveImageURL returns a presigned GET URL for the user's avatar, or nil if
// the object does not exist or cannot be signed. Mirrors the Express behavior
// at dashboard/apps/api/src/repositories/user.repo.ts:87-97 — the user-photos
// bucket is private and SSE-S3-encrypted, so clients cannot fetch directly
// without a short-lived signed URL. Returning nil on any error ensures a
// missing avatar never breaks the account-user response.
func (s *accountUserSvcImpl) resolveImageURL(ctx context.Context, accountID, userID string) *string {
	key := accountID + "/" + userID + ".png"
	exists, err := s.s3Client.FileExists(ctx, s.userPhotosBucket, key)
	if err != nil || !exists {
		return nil
	}
	url, err := s.s3Client.GetPresignedURL(ctx, s.userPhotosBucket, key, time.Hour)
	if err != nil {
		return nil
	}
	return &url
}

// ListAccountUsers returns a paginated list of account users.
func (s *accountUserSvcImpl) ListAccountUsers(ctx context.Context, params domain.ListAccountUsersParams) (*domain.ListAccountUsersResult, *apierror.APIError) {
	ctx, span := accountUserSvcTracer.Start(ctx, "service.account_user.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAccountUserReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewAccountUserRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, item := range result.Items {
		item.ImageURL = s.resolveImageURL(ctx, params.AccountID, item.UserID)
	}
	return result, nil
}

// GetAccountUser returns a single account user by account_user ID.
func (s *accountUserSvcImpl) GetAccountUser(ctx context.Context, accountUserID string) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserSvcTracer.Start(ctx, "service.account_user.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAccountUserReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	detail, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, identity.Target.AccountID, accountUserID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	detail.ImageURL = s.resolveImageURL(ctx, identity.Target.AccountID, detail.UserID)
	return detail, nil
}

// CreateAccountUser creates a new account user with idempotency support.
func (s *accountUserSvcImpl) CreateAccountUser(ctx context.Context, params domain.CreateAccountUserParams) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserSvcTracer.Start(ctx, "service.account_user.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAccountUserWritePermission(identity, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), params.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Validate: must have email or username
	if params.Email == nil && params.Username == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Either email or username is required."))
	}

	// Scanning station logic: username without email means this is a scanning station user.
	isScanningStation := params.Username != nil && params.Email == nil
	if isScanningStation {
		if params.Password == nil {
			return nil, tracing.Trace(span, apierror.NewValidationError("Password is required for scanning station users."))
		}
	}

	if !isScanningStation && params.Password != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Password cannot be set directly for non-scanning station users."))
	}

	// Check seat limits only for own account (not external customer accounts)
	isOwnAccount := identity.Actor != nil && identity.Actor.AccountID != nil && *identity.Actor.AccountID == params.AccountID
	if isOwnAccount {
		if apiErr := s.checkSeatLimit(ctx, params.AccountID); apiErr != nil {
			return nil, apiErr
		}
	}

	accountUserID, apiErr := id.GenID(id.AccountUserIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountUserDetail](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Data != nil {
			cached.Data.ImageURL = s.resolveImageURL(ctx, params.AccountID, cached.Data.UserID)
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountUserDetail
		var userID string
		var generatedPassword string

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountUserSvcImpl) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txSvc.repos)
			txUserRepo := txSvc.repos.NewUserRepo()
			txAccountUserRepo := txSvc.repos.NewAccountUserRepo()
			txRoleRepo := txSvc.repos.NewRoleRepo()

			// Scanning station users must always use the scanner role, regardless of what was provided.
			if isScanningStation {
				scannerRole, apiErr := txRoleRepo.FindByTypeCode(txCtx, string(constants.RoleTypeCodeScanner), params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				params.RoleID = &scannerRole.ID
			}

			// Sales-rep inference (mirrors Express): when the caller provides a role whose
			// type is sales_rep, normalize to the canonical sales-rep role for the account.
			// Skipped on the scanner path because the role has already been forced above.
			if !isScanningStation && params.RoleID != nil {
				providedRole, apiErr := txRoleRepo.Get(txCtx, *params.RoleID, params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				if providedRole.RoleTypeCode == string(constants.RoleTypeCodeSalesRep) {
					salesRepRole, apiErr := txRoleRepo.FindByTypeCode(txCtx, string(constants.RoleTypeCodeSalesRep), params.AccountID)
					if apiErr != nil {
						return apiErr
					}
					params.RoleID = &salesRepRole.ID
				}
			}

			// Try to find existing user by email or username.
			var existingUser *domain.UserRecord
			if params.Email != nil {
				found, apiErr := txUserRepo.FindByEmail(txCtx, *params.Email)
				if apiErr != nil && apiErr.Code != apierror.ErrorCodeResourceNotFound {
					return apiErr
				}
				existingUser = found
			}
			if existingUser == nil && params.Username != nil {
				found, apiErr := txUserRepo.FindByUsername(txCtx, *params.Username)
				if apiErr != nil && apiErr.Code != apierror.ErrorCodeResourceNotFound {
					return apiErr
				}
				existingUser = found
			}

			if existingUser != nil {
				userID = existingUser.ID

				// Check if already linked to this account.
				_, existErr := txAccountUserRepo.FindByAccountAndUserID(txCtx, userID, params.AccountID)
				if existErr == nil {
					return apierror.NewResourceConflictError("This user is already linked to this account.")
				}
				if existErr.Code != apierror.ErrorCodeResourceNotFound {
					return existErr
				}
			} else {
				// Create a new user.
				newUserID, apiErr := id.GenID(id.UserIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				userID = newUserID

				var hashedPassword *string
				if params.Password != nil {
					hp, err := crypto.HashBcrypt(*params.Password)
					if err != nil {
						return apierror.NewInternalError(err, "Failed to hash password.")
					}
					hashedPassword = &hp
				} else if !isScanningStation {
					// Generate a random password for non-scanning-station users.
					genPass, err := crypto.RandHexString(16)
					if err != nil {
						return apierror.NewInternalError(err, "Failed to generate random password.")
					}
					generatedPassword = genPass
					hp, err := crypto.HashBcrypt(generatedPassword)
					if err != nil {
						return apierror.NewInternalError(err, "Failed to hash password.")
					}
					hashedPassword = &hp
				}

				if apiErr := txUserRepo.CreateUser(txCtx, userID, domain.CreateUserRecordParams{
					Name:           params.Name,
					Email:          params.Email,
					Username:       params.Username,
					HashedPassword: hashedPassword,
				}); apiErr != nil {
					return apiErr
				}
			}

			// Create the account_user link.
			if apiErr := txAccountUserRepo.Create(txCtx, accountUserID, params.AccountID, userID, params.RoleID, params.DepartmentID); apiErr != nil {
				return apiErr
			}

			// Auto-disable external target users when the target account has an active billing plan.
			if identity.IsExternalTarget() {
				hasPlan, apiErr := txSvc.repos.NewAccountRepo().HasActiveBillingPlan(txCtx, params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				if hasPlan {
					if apiErr := txAccountUserRepo.UpdateStatus(txCtx, accountUserID, constants.AccountUserStatusDisabled); apiErr != nil {
						return apiErr
					}
				}
			}

			// Create notification preferences for external target users.
			if identity.IsExternalTarget() && len(params.NotificationPreferences) > 0 {
				for _, pref := range params.NotificationPreferences {
					if !constants.AccountRelationNotificationType(pref.NotificationTypeCode).IsValid() {
						return apierror.NewValidationError(fmt.Sprintf("Invalid notification type code: %s", pref.NotificationTypeCode))
					}
				}
				txRelationRepo := txSvc.repos.NewAccountRelationRepo()
				relationID, apiErr := txRelationRepo.FindRelationByOwnerAndCounterparty(txCtx, *identity.ActorAccountID(), params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				for _, pref := range params.NotificationPreferences {
					if !pref.Enabled {
						continue
					}
					prefID, apiErr := id.GenID(id.AccountRelationNotificationPreferenceIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txRelationRepo.CreateNotificationPreference(txCtx, prefID, relationID, accountUserID, pref.NotificationTypeCode); apiErr != nil {
						return apiErr
					}
				}
			}

			// Publish seat sync and seat change report via outbox.
			if apiErr := txSvc.billingPublisher.PublishSyncSeats(txCtx, params.AccountID); apiErr != nil {
				return apiErr
			}
			if apiErr := txSvc.billingPublisher.PublishReportSeatChange(txCtx, params.AccountID); apiErr != nil {
				return apiErr
			}

			// Send welcome email if user has an email and we generated a password for them.
			if params.Email != nil && generatedPassword != "" {
				emailParams := map[string]any{
					"Name":     stringOrDefault(params.Name, "there"),
					"Password": generatedPassword,
				}

				subject := "Welcome to Augno"

				if identity.IsExternalTarget() {
					actorAccountID := *identity.ActorAccountID()
					txAccountRepo := txSvc.repos.NewAccountRepo()
					accountName, _ := txAccountRepo.GetName(txCtx, actorAccountID)
					logoURL, _ := txAccountRepo.GetBrandingLogoURL(txCtx, actorAccountID)
					slug, _ := txAccountRepo.GetPortalSlug(txCtx, actorAccountID)

					emailParams["AccountName"] = accountName
					emailParams["IsBranded"] = true
					if logoURL != nil {
						emailParams["LogoURL"] = *logoURL
					}
					if slug != nil {
						emailParams["LoginLink"] = "https://app.augno.com/" + *slug + "/login"
					}
					subject = "Welcome to the " + accountName + " platform"
				}

				if apiErr := txSvc.notificationPublisher.PublishSendEmail(txCtx, messaging.EmailSendData{
					To:         []string{*params.Email},
					Subject:    subject,
					TemplateID: constants.EmailTemplateNewUserWelcome,
					Params:     emailParams,
				}); apiErr != nil {
					return apiErr
				}
			}

			// Fetch the created detail for response.
			detail, apiErr := txAccountUserRepo.GetDetailByAccountAndID(txCtx, params.AccountID, accountUserID)
			if apiErr != nil {
				return apiErr
			}
			result = detail

			changes := audit.ComputeChanges(nil, detail)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAccountUser,
				ResourceID:   detail.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		if result != nil {
			result.ImageURL = s.resolveImageURL(ctx, params.AccountID, result.UserID)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// UpdateAccountUser partially updates an account user with idempotency support.
func (s *accountUserSvcImpl) UpdateAccountUser(ctx context.Context, params domain.UpdateAccountUserParams) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserSvcTracer.Start(ctx, "service.account_user.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Resolve the account user to check self-edit and get the user ID.
	accountUserDetail, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, params.AccountID, params.AccountUserID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Allow self-edit or require customers:update permission.
	isSelfEdit := identity.Actor != nil && identity.Actor.ID == accountUserDetail.UserID
	if !isSelfEdit {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if identity.IsExternalTarget() {
		medsCheck := s.mediators()
		if apiErr := medsCheck.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), params.AccountID); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountUserDetail](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Data != nil {
			cached.Data.ImageURL = s.resolveImageURL(ctx, params.AccountID, cached.Data.UserID)
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Validate notification preferences (if any) before opening the transaction.
		if params.NotificationPreferences != nil {
			if !identity.IsExternalTarget() {
				return nil, tracing.Trace(span, apierror.NewValidationError("Notification preferences can only be updated for external (cross-account) users."))
			}
			for _, pref := range params.NotificationPreferences {
				if !constants.AccountRelationNotificationType(pref.NotificationTypeCode).IsValid() {
					return nil, tracing.Trace(span, apierror.NewValidationError(fmt.Sprintf("Invalid notification type code: %s", pref.NotificationTypeCode)))
				}
			}
		}

		var result *domain.AccountUserDetail
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountUserSvcImpl) *apierror.APIError {
			txAccountUserRepo := txSvc.repos.NewAccountUserRepo()
			txUserRepo := txSvc.repos.NewUserRepo()

			// Fetch old state for audit diff.
			old, apiErr := txAccountUserRepo.GetDetailByAccountAndID(txCtx, params.AccountID, params.AccountUserID)
			if apiErr != nil {
				return apiErr
			}
			userID := old.UserID

			// Check for duplicate email before updating.
			if params.Email != nil {
				existing, apiErr := txUserRepo.FindByEmail(txCtx, *params.Email)
				if apiErr != nil && apiErr.Code != apierror.ErrorCodeResourceNotFound {
					return apiErr
				}
				if existing != nil && existing.ID != userID {
					return apierror.NewConflictErrorWithParam("A user with this email already exists.", "email")
				}
			}

			// Check for duplicate username before updating.
			if params.Username != nil {
				existing, apiErr := txUserRepo.FindByUsername(txCtx, *params.Username)
				if apiErr != nil && apiErr.Code != apierror.ErrorCodeResourceNotFound {
					return apiErr
				}
				if existing != nil && existing.ID != userID {
					return apierror.NewConflictErrorWithParam("A user with this username already exists.", "username")
				}
			}

			// Update user profile fields if provided.
			if params.Name != nil || params.Email != nil || params.Username != nil {
				if apiErr := txUserRepo.UpdateProfile(txCtx, userID, params.Name, params.Email, params.Username, nil, nil); apiErr != nil {
					return apiErr
				}
			}

			// Backfill unchanged nullable fields with existing values.
			// Since the SQL uses direct assignment (no COALESCE) for these fields,
			// we must provide the existing value when the field was not sent.
			if params.RoleID == nil {
				params.RoleID = old.RoleID
			}
			if params.DepartmentID == nil {
				params.DepartmentID = old.DepartmentID
			}

			// Update account user role and department.
			if apiErr := txAccountUserRepo.Update(txCtx, params.AccountUserID, params.RoleID, params.DepartmentID); apiErr != nil {
				return apiErr
			}

			// Apply notification preference toggles (external targets only).
			if params.NotificationPreferences != nil {
				txRelationRepo := txSvc.repos.NewAccountRelationRepo()
				relationID, apiErr := txRelationRepo.FindRelationByOwnerAndCounterparty(txCtx, *identity.ActorAccountID(), params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				existingPrefs, apiErr := txRelationRepo.ListNotificationPreferences(txCtx, relationID, params.AccountUserID)
				if apiErr != nil {
					return apiErr
				}
				existingSet := make(map[string]bool, len(existingPrefs))
				for _, p := range existingPrefs {
					existingSet[p.NotificationTypeCode] = true
				}
				for _, pref := range params.NotificationPreferences {
					if pref.Enabled && !existingSet[pref.NotificationTypeCode] {
						prefID, apiErr := id.GenID(id.AccountRelationNotificationPreferenceIDPrefix, nil)
						if apiErr != nil {
							return apiErr
						}
						if apiErr := txRelationRepo.CreateNotificationPreference(txCtx, prefID, relationID, params.AccountUserID, pref.NotificationTypeCode); apiErr != nil {
							return apiErr
						}
					} else if !pref.Enabled && existingSet[pref.NotificationTypeCode] {
						if apiErr := txRelationRepo.DeleteNotificationPreference(txCtx, relationID, params.AccountUserID, pref.NotificationTypeCode); apiErr != nil {
							return apiErr
						}
					}
				}
			}

			detail, apiErr := txAccountUserRepo.GetDetailByAccountAndID(txCtx, params.AccountID, params.AccountUserID)
			if apiErr != nil {
				return apiErr
			}
			result = detail

			changes := audit.ComputeChanges(old, detail)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAccountUser,
				ResourceID:   detail.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		if result != nil {
			result.ImageURL = s.resolveImageURL(ctx, params.AccountID, result.UserID)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// UpdateAccountUserStatus transitions an account user to the given target status.
// Consolidates lock (active→disabled), unlock (disabled→active), restore (removed→active),
// and delete (→removed). Idempotent: calling with the current status is a no-op.
func (s *accountUserSvcImpl) UpdateAccountUserStatus(ctx context.Context, accountUserID string, targetStatus constants.AccountUserStatus) *apierror.APIError {
	ctx, span := accountUserSvcTracer.Start(ctx, "service.account_user.update_status")
	defer span.End()

	if !targetStatus.IsValid() {
		return tracing.Trace(span, apierror.NewValidationError(fmt.Sprintf("Invalid status: %s", targetStatus)))
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Removal uses the delete permission; other transitions use update.
	requiredAction := types.ActionUpdate
	if targetStatus == constants.AccountUserStatusRemoved {
		requiredAction = types.ActionDelete
	}
	if apiErr := checkAccountUserWritePermission(identity, requiredAction); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), accountID); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Caller must not be disabled when performing status transitions.
	if identity.IsUser() {
		callerDetail, apiErr := s.repos.NewAccountUserRepo().GetDetail(ctx, accountID, identity.Actor.ID)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if callerDetail.StatusCode == constants.AccountUserStatusDisabled {
			return tracing.Trace(span, apierror.NewAuthorizationError("Your account is locked. You cannot perform this action."))
		}
	}

	// Resolve current state. Handle "already removed" like the old delete handler did.
	accountUser, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, accountID, accountUserID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeAccountUser, accountUserID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This account user has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	// Idempotent no-op when already in the target state.
	if accountUser.StatusCode == targetStatus {
		return nil
	}

	// Validate transition-specific rules.
	switch targetStatus {
	case constants.AccountUserStatusDisabled:
		if accountUser.StatusCode == constants.AccountUserStatusRemoved {
			return tracing.Trace(span, apierror.NewValidationError("Cannot lock a removed user. Restore the user first."))
		}
		if identity.IsUser() && identity.Actor != nil && identity.Actor.ID == accountUser.UserID {
			return tracing.Trace(span, apierror.NewValidationError("You cannot lock your own account."))
		}
		if accountUser.RoleTypeCode != nil && *accountUser.RoleTypeCode == string(constants.RoleTypeCodeAdmin) {
			return tracing.Trace(span, apierror.NewValidationError("Admin users cannot be locked."))
		}
	case constants.AccountUserStatusActive:
		// Reactivating (from disabled or removed) consumes a seat.
		if apiErr := s.checkSeatLimit(ctx, accountID); apiErr != nil {
			return apiErr
		}
	case constants.AccountUserStatusRemoved:
		// Nothing extra: any non-removed user may be removed.
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountUserSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txAccountUserRepo := txSvc.repos.NewAccountUserRepo()

		// Removal performs a soft-delete and records the deleted snapshot.
		if targetStatus == constants.AccountUserStatusRemoved {
			if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAccountUser, accountUser.ID, accountUser); apiErr != nil {
				return apiErr
			}
			if apiErr := txAccountUserRepo.SoftDelete(txCtx, accountUserID); apiErr != nil {
				return apiErr
			}
		} else {
			if apiErr := txAccountUserRepo.UpdateStatus(txCtx, accountUserID, targetStatus); apiErr != nil {
				return apiErr
			}
			// Locking revokes outstanding refresh tokens.
			if targetStatus == constants.AccountUserStatusDisabled {
				if apiErr := txAccountUserRepo.RevokeRefreshTokensByUserID(txCtx, accountUser.UserID); apiErr != nil {
					return apiErr
				}
			}
		}

		auditAction := constants.AuditActionUpdate
		var updated *domain.AccountUserDetail
		if targetStatus == constants.AccountUserStatusRemoved {
			auditAction = constants.AuditActionDelete
			updated = nil
		} else {
			next, apiErr := txAccountUserRepo.GetDetailByAccountAndID(txCtx, accountID, accountUserID)
			if apiErr != nil {
				return apiErr
			}
			updated = next
		}

		changes := audit.ComputeChanges(accountUser, updated)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       auditAction,
			ResourceType: constants.ObjectTypeAccountUser,
			ResourceID:   accountUser.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.billingPublisher.PublishReportSeatChange(txCtx, accountID); apiErr != nil {
			return apiErr
		}
		return txSvc.billingPublisher.PublishSyncSeats(txCtx, accountID)
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// UpdateAccountUserPassword updates the password for a scanner-role account user after verifying the requester's password.
func (s *accountUserSvcImpl) UpdateAccountUserPassword(ctx context.Context, accountUserID, requesterPassword, newPassword string) *apierror.APIError {
	ctx, span := accountUserSvcTracer.Start(ctx, "service.account_user.update_password")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// Scanner-password rotation verifies the caller's own password, which only
	// exists for user identities (session auth). API keys and other actor types
	// have no password to verify against.
	if !identity.IsUser() {
		return tracing.Trace(span, apierror.NewAuthorizationError("Scanner password updates require session authentication."))
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Verify requester's password first.
	if identity.Actor == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Actor not found in identity."))
	}

	requesterHashedPassword, apiErr := s.repos.NewUserRepo().GetHashedPassword(ctx, identity.Actor.ID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	matches, err := crypto.CompareBcryptHash(requesterHashedPassword, requesterPassword)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to verify password."))
	}
	if !matches {
		return tracing.Trace(span, apierror.NewAuthenticationError("Incorrect password."))
	}

	// Verify that the target account user belongs to the requester's account.
	targetAccountUser, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, identity.Target.AccountID, accountUserID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Passwords may only be rotated for scanner-role (scanning station) users.
	if targetAccountUser.RoleTypeCode == nil || *targetAccountUser.RoleTypeCode != string(constants.RoleTypeCodeScanner) {
		return tracing.Trace(span, apierror.NewValidationError("Password updates are only supported for scanner-role users."))
	}

	// Hash the new password.
	newHashedPassword, err := crypto.HashBcrypt(newPassword)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to hash new password."))
	}

	// Update the target user's password within a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountUserSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewUserRepo().UpdatePassword(txCtx, targetAccountUser.UserID, newHashedPassword); apiErr != nil {
			return apiErr
		}

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeAccountUser,
			ResourceID:   targetAccountUser.ID,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// checkSeatLimit checks whether the account has room for another active user.
func (s *accountUserSvcImpl) checkSeatLimit(ctx context.Context, accountID string) *apierror.APIError {
	accountRepo := s.repos.NewAccountRepo()
	accountUserRepo := s.repos.NewAccountUserRepo()

	planCode, apiErr := accountRepo.GetPlanCode(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}

	seatLimit, apiErr := accountRepo.GetSeatLimitByPlanCode(ctx, string(planCode))
	if apiErr != nil {
		return apiErr
	}

	// nil means unlimited seats.
	if seatLimit == nil {
		return nil
	}

	activeCount, apiErr := accountUserRepo.CountActive(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}

	if activeCount >= int64(*seatLimit) {
		return apierror.NewValidationError("You have reached the maximum number of users for your plan.")
	}

	return nil
}

func stringOrDefault(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// checkAccountUserReadPermission checks the appropriate read permission based on the target context.
// Internal actors targeting a customer account need customers:read; supplier account needs suppliers:read;
// own account needs team:read.
func checkAccountUserReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainTeamUsers, types.ActionRead)
}

// checkAccountUserWritePermission checks the appropriate write permission based on the target context.
// Internal actors targeting a customer account need customers:update; supplier account needs suppliers:update;
// own account needs team:{action}.
func checkAccountUserWritePermission(identity *types.Identity, action types.Action) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
	}
	return identity.CheckHasPermission(types.PermissionDomainTeamUsers, action)
}
