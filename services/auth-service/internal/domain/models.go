package domain

import (
	"encoding/json"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type APIKey struct {
	ID             int64
	KeyID          string `json:"-"`
	TypeID         string
	Name           string
	LastFour       string
	SecretHash     []byte `json:"-"`
	OwnerAccountID string
	RoleID         string
	RoleName       string
	RoleTypeCode   string
	RedactedValue  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
	ExpiresAt      *time.Time
	RevokedAt      *time.Time
}

func (m *APIKey) ToProto() *pb.APIKeyInfo {
	if m == nil {
		return nil
	}

	info := &pb.APIKeyInfo{
		Id:            m.TypeID,
		Name:          m.Name,
		RedactedValue: m.RedactedValue,
		RoleId:        m.RoleID,
		RoleName:      m.RoleName,
		CreatedAt:     timestamppb.New(m.CreatedAt),
		UpdatedAt:     timestamppb.New(m.UpdatedAt),
	}

	if m.LastUsedAt != nil {
		info.LastUsedAt = timestamppb.New(*m.LastUsedAt)
	}
	if m.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*m.ExpiresAt)
	}
	if m.RevokedAt != nil {
		info.RevokedAt = timestamppb.New(*m.RevokedAt)
	}

	return info
}

type DocAPIKey struct {
	ID              int64
	TypeID          string
	APIKeyID        string
	EncryptedSecret string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	APIKeyExpiresAt *time.Time
	APIKeyRevokedAt *time.Time
}

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

type ParsedAPIKey struct {
	AccountMode constants.AccountMode
	ID          string
	Secret      string // #nosec G117 - Struct field, not a hardcoded credential
	Checksum    string
}

func (p *ParsedAPIKey) String() string {
	return string(types.APIKeyPrefixSecretKey) + string(p.AccountMode) + "_" + p.ID + "_" + p.Secret + p.Checksum
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

// func RegistrationSessionDataFromProto(p *pb.RegistrationSessionData) *RegistrationSessionData {
// 	if p == nil {
// 		return nil
// 	}
// 	return &RegistrationSessionData{
// 		UserName:                 p.GetUserName(),
// 		AccountName:              p.GetAccountName(),
// 		BillingAddressLine1:      p.GetBillingAddressLine1(),
// 		BillingAddressLine2:      p.GetBillingAddressLine2(),
// 		BillingAddressCity:       p.GetBillingAddressCity(),
// 		BillingAddressState:      p.GetBillingAddressState(),
// 		BillingAddressPostalCode: p.GetBillingAddressPostalCode(),
// 		BillingAddressCountry:    p.GetBillingAddressCountry(),
// 	}
// }

// func (s *RegistrationSession) ToProto() *pb.RegistrationSession {
// 	if s == nil {
// 		return nil
// 	}

// 	return &pb.RegistrationSession{
// 		Id:                      s.TypeID,
// 		Email:                   s.Email,
// 		PlanCode:                s.PlanCode,
// 		Step:                    string(s.Step),
// 		IsEmailVerified:         s.IsEmailVerified,
// 		IsExistingUser:          s.IsExistingUser,
// 		UserId:                  s.UserID,
// 		AccountId:               s.AccountID,
// 		StripeCustomerId:        s.StripeCustomerID,
// 		StripeCheckoutSessionId: s.StripeCheckoutSessionID,
// 		StripeSubscriptionId:    s.StripeSubscriptionID,
// 		PaymentCompleted:        s.PaymentCompleted,
// 		SessionData: &pb.RegistrationSessionData{
// 			UserName:                 ptrutil.String(s.SessionData.UserName),
// 			AccountName:              ptrutil.String(s.SessionData.AccountName),
// 			BillingAddressLine1:      ptrutil.String(s.SessionData.BillingAddressLine1),
// 			BillingAddressLine2:      ptrutil.String(s.SessionData.BillingAddressLine2),
// 			BillingAddressCity:       ptrutil.String(s.SessionData.BillingAddressCity),
// 			BillingAddressState:      ptrutil.String(s.SessionData.BillingAddressState),
// 			BillingAddressPostalCode: ptrutil.String(s.SessionData.BillingAddressPostalCode),
// 			BillingAddressCountry:    ptrutil.String(s.SessionData.BillingAddressCountry),
// 		},
// 		CompletedAt: ptrutil.TimePtrToProto(s.CompletedAt),
// 		CreatedAt:   timestamppb.New(s.CreatedAt),
// 		UpdatedAt:   timestamppb.New(s.UpdatedAt),
// 	}
// }

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
