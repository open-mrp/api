package domain

import (
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthAccountRelation struct {
	ID                      string
	CounterpartyAccountID   string
	AccountRelationRoleCode types.IdentityRelationType
	IsOwnerSide             bool
}

type RefreshToken struct {
	Token     string
	UserID    string     `audit:"user_id"`
	ExpiresAt time.Time  `audit:"expires_at"`
	RevokedAt *time.Time `audit:"revoked_at"`
}

// IsRevoked reports whether the refresh token has been revoked.
func (m *RefreshToken) IsRevoked() bool {
	return m.RevokedAt != nil
}

// IsExpired reports whether the refresh token has passed its expiration time.
func (m *RefreshToken) IsExpired() bool {
	return m.ExpiresAt.Before(time.Now().UTC())
}

type RegistrationSession struct {
	ID                      int64
	TypeID                  string
	Email                   string                     `audit:"email"`
	PlanCode                string                     `audit:"plan_code"`
	Step                    constants.RegistrationStep `audit:"step"`
	VerificationToken       string
	IsEmailVerified         bool    `audit:"is_email_verified"`
	IsExistingUser          *bool   `audit:"is_existing_user"`
	UserID                  *string `audit:"user_id"`
	AccountID               *string `audit:"account_id"`
	StripeCustomerID        *string
	StripeCheckoutSessionID *string
	StripeSubscriptionID    *string
	PaymentCompleted        bool                    `audit:"payment_completed"`
	SessionData             RegistrationSessionData `audit:"session_data"`
	CompletedAt             *time.Time              `audit:"completed_at"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type RegistrationSessionData struct {
	UserName                 string
	AccountName              string
	BillingAddressLine1      string
	BillingAddressLine2      string
	BillingAddressCity       string
	BillingAddressState      string
	BillingAddressPostalCode string
	BillingAddressCountry    string
}

func (s *RegistrationSession) ToProto() *pb.RegistrationSessionInfo {
	if s == nil {
		return nil
	}

	var completedAt *timestamppb.Timestamp
	if s.CompletedAt != nil {
		completedAt = timestamppb.New(*s.CompletedAt)
	}

	return &pb.RegistrationSessionInfo{
		Id:                      s.TypeID,
		Email:                   s.Email,
		PlanCode:                s.PlanCode,
		Step:                    string(s.Step),
		IsEmailVerified:         s.IsEmailVerified,
		IsExistingUser:          s.IsExistingUser,
		UserId:                  s.UserID,
		AccountId:               s.AccountID,
		StripeCustomerId:        s.StripeCustomerID,
		StripeCheckoutSessionId: s.StripeCheckoutSessionID,
		StripeSubscriptionId:    s.StripeSubscriptionID,
		PaymentCompleted:        s.PaymentCompleted,
		SessionData: &pb.RegistrationSessionData{
			UserName:                 new(s.SessionData.UserName),
			AccountName:              new(s.SessionData.AccountName),
			BillingAddressLine1:      new(s.SessionData.BillingAddressLine1),
			BillingAddressLine2:      new(s.SessionData.BillingAddressLine2),
			BillingAddressCity:       new(s.SessionData.BillingAddressCity),
			BillingAddressState:      new(s.SessionData.BillingAddressState),
			BillingAddressPostalCode: new(s.SessionData.BillingAddressPostalCode),
			BillingAddressCountry:    new(s.SessionData.BillingAddressCountry),
		},
		CompletedAt: completedAt,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
}

type IdempotencyKey struct {
	ID             int64
	TypeID         string
	ServiceName    string
	Handler        string
	IdempotencyKey string
	ActorID        *string
	IdentityType   string
	ScopeHash      string
	ResponseCode   *int
	ResponseBody   json.RawMessage
	RecoveryPoint  RecoveryPoint
}

func (k *IdempotencyKey) HasResponse() bool {
	return k.ResponseCode != nil
}

func (k *IdempotencyKey) IsFinished() bool {
	return k.RecoveryPoint.IsFinished()
}

// SigningPin is a person's signing PIN as stored; PinHash never leaves auth-service.
type SigningPin struct {
	UserID         string
	PinHash        string
	IsTemporary    bool
	FailedAttempts int
	LockedUntil    *time.Time
	SetAt          time.Time
}

// IsLocked reports whether too many wrong PINs have locked signing at now.
func (p *SigningPin) IsLocked(now time.Time) bool {
	return p.LockedUntil != nil && p.LockedUntil.After(now)
}

// SigningAccountUser is an account member whose signing PIN is being set or checked.
type SigningAccountUser struct {
	ID           string
	UserID       string
	AccountID    string
	DepartmentID *string
	StatusCode   string
	Name         *string
	Username     *string
}

// IsActive reports whether the member may still sign in the account.
func (u *SigningAccountUser) IsActive() bool {
	return u.StatusCode == "active"
}

// DisplayName is the name copied onto what they sign; members without a name sign as their username.
func (u *SigningAccountUser) DisplayName() string {
	if u.Name != nil && *u.Name != "" {
		return *u.Name
	}
	if u.Username != nil {
		return *u.Username
	}
	return ""
}

// VerifiedSigner is who a correct signing PIN identified, as of the check.
type VerifiedSigner struct {
	AccountUserID string
	UserID        string
	Name          string
	RoleID        *string
	RoleName      *string
	RoleTypeCode  *string
	DepartmentID  *string
	// Permissions are "<domain>:<action>", sorted; empty for admins, who hold every permission.
	Permissions []string
}
