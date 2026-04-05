package domain

import "encoding/json"

// IdempotencyKey represents a service-level idempotency key.
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

// RequestIdentity contains the identity context for an idempotent operation.
type RequestIdentity struct {
	ActorID         string
	IdentityType    string
	TargetAccountID *string
}
