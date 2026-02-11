package domain

import (
	"encoding/json"
	"time"
)

type EmailLog struct {
	ID           string
	HasSent      bool
	AccountID    string
	SentByID     *string
	Subject      *string
	Filename     *string
	SesMessageID *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	RecoveryPoint  string
}

func (k *IdempotencyKey) HasResponse() bool {
	return k.ResponseCode != nil
}

func (k *IdempotencyKey) IsFinished() bool {
	return k.RecoveryPoint == string(RecoveryPointFinished)
}
