package domain

import (
	"encoding/json"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthAccountRelation struct {
	ID                      string
	CounterpartyAccountID   string
	AccountRelationRoleCode types.IdentityActorType
}

type RefreshToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
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
	Email                   string
	PlanCode                string
	Step                    constants.RegistrationStep
	VerificationToken       string
	IsEmailVerified         bool
	IsExistingUser          *bool
	UserID                  *string
	AccountID               *string
	StripeCustomerID        *string
	StripeCheckoutSessionID *string
	StripeSubscriptionID    *string
	PaymentCompleted        bool
	SessionData             RegistrationSessionData
	CompletedAt             *time.Time
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
