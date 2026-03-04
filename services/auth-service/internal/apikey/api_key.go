package apikey

import (
	"time"

	pb "github.com/augno/api/shared/proto/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type APIKey struct {
	ID             int64
	KeyID          string `json:"-"`
	TypeID         string
	Name           string
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

// IsExpired reports whether the API key has passed its expiration time.
func (m *APIKey) IsExpired() bool {
	return m.ExpiresAt != nil && m.ExpiresAt.Before(time.Now().UTC())
}

// IsRevoked reports whether the API key has been revoked.
func (m *APIKey) IsRevoked() bool {
	return m.RevokedAt != nil
}

// ShouldTouch reports whether the API key's last-used timestamp should be
// updated. It returns true when the key has never been used or was last used
// before the given threshold time.
func (m *APIKey) ShouldTouch(threshold time.Time) bool {
	return m.LastUsedAt == nil || m.LastUsedAt.Before(threshold)
}

func (m *APIKey) ToProto() *pb.APIKeyInfo {
	if m == nil {
		return nil
	}

	info := &pb.APIKeyInfo{
		Id:            m.TypeID,
		Name:          m.Name,
		RedactedValue: m.RedactedValue,
		CreatedAt:     timestamppb.New(m.CreatedAt),
		UpdatedAt:     timestamppb.New(m.UpdatedAt),
	}

	if m.RoleID != "" {
		info.RoleId = &m.RoleID
	}
	if m.RoleName != "" {
		info.RoleName = &m.RoleName
	}
	if m.RoleTypeCode != "" {
		info.RoleTypeCode = &m.RoleTypeCode
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
