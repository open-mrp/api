package types

import (
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/ptrutil"
)

type IdentityActor struct {
	Type         IdentityActorType
	ID           string
	Name         *string
	AccountID    *string
	RoleID       *string
	RoleTypeCode *string
	Permissions  map[string]bool
}

type Identity struct {
	Type            IdentityType
	TargetAccountID *string
	Actor           *IdentityActor
	AccountMode     constants.AccountMode
}

func (i *Identity) IsUser() bool {
	return i.Type == IdentityTypeUser
}

func (i *Identity) IsInternalUser() bool {
	return i.Type == IdentityTypeUser && i.Actor != nil && i.Actor.Type == IdentityActorTypeInternal
}

func (i *Identity) IsSupplierUser() bool {
	return i.Type == IdentityTypeUser && i.Actor != nil && i.Actor.Type == IdentityActorTypeSupplier
}

func (i *Identity) IsCustomerUser() bool {
	return i.Type == IdentityTypeUser && i.Actor != nil && i.Actor.Type == IdentityActorTypeCustomer
}

func (i *Identity) IsUnassignedUser() bool {
	return i.Type == IdentityTypeUser && i.Actor != nil && i.Actor.Type == IdentityActorTypeUnassigned
}

func (i *Identity) IsAPIKey() bool {
	return i.Type == IdentityTypeAPIKey
}

func GetUnauthenticatedIdentity(targetAccountID string) *Identity {
	targetPtr := (*string)(nil)
	if targetAccountID != "" {
		targetPtr = ptrutil.String(targetAccountID)
	}

	return &Identity{
		Type:            IdentityTypeUnauthenticated,
		Actor:           nil,
		AccountMode:     constants.AccountModeProduction, // TODO: This should be dynamic
		TargetAccountID: targetPtr,
	}
}

func (i *Identity) ToProto() *pb.Identity {
	if i == nil {
		return nil
	}

	var actor *pb.IdentityActor
	if i.Actor != nil {
		actor = &pb.IdentityActor{
			Type:         convertIdentityActorTypeToProto(i.Actor.Type),
			Id:           i.Actor.ID,
			Name:         i.Actor.Name,
			AccountId:    i.Actor.AccountID,
			RoleId:       i.Actor.RoleID,
			RoleTypeCode: i.Actor.RoleTypeCode,
			Permissions:  i.Actor.Permissions,
		}
	}

	return &pb.Identity{
		Type:            convertIdentityTypeToProto(i.Type),
		TargetAccountId: i.TargetAccountID,
		Actor:           actor,
		AccountMode:     convertAccountModeToProto(i.AccountMode),
	}
}

func convertIdentityTypeToProto(t IdentityType) pb.IdentityType {
	switch t {
	case IdentityTypeUser:
		return pb.IdentityType_IDENTITY_TYPE_USER
	case IdentityTypeAPIKey:
		return pb.IdentityType_IDENTITY_TYPE_API_KEY
	case IdentityTypeUnauthenticated:
		return pb.IdentityType_IDENTITY_TYPE_UNAUTHENTICATED
	default:
		return pb.IdentityType_IDENTITY_TYPE_UNSPECIFIED
	}
}

func convertIdentityActorTypeToProto(t IdentityActorType) pb.IdentityActorType {
	switch t {
	case IdentityActorTypeInternal:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_INTERNAL
	case IdentityActorTypeCustomer:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_CUSTOMER
	case IdentityActorTypeSupplier:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_SUPPLIER
	case IdentityActorTypeUnassigned:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNASSIGNED
	default:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNSPECIFIED
	}
}

func convertAccountModeToProto(m constants.AccountMode) pb.AccountMode {
	switch m {
	case constants.AccountModeProduction:
		return pb.AccountMode_ACCOUNT_MODE_PRODUCTION
	case constants.AccountModeSandbox:
		return pb.AccountMode_ACCOUNT_MODE_SANDBOX
	default:
		return pb.AccountMode_ACCOUNT_MODE_UNSPECIFIED
	}
}

func IdentityFromProto(pbIdentity *pb.Identity) *Identity {
	if pbIdentity == nil {
		return nil
	}

	var actor *IdentityActor
	if pbIdentity.Actor != nil {
		actor = &IdentityActor{
			Type:         convertIdentityActorTypeFromProto(pbIdentity.Actor.Type),
			ID:           pbIdentity.Actor.Id,
			Name:         pbIdentity.Actor.Name,
			AccountID:    pbIdentity.Actor.AccountId,
			RoleID:       pbIdentity.Actor.RoleId,
			RoleTypeCode: pbIdentity.Actor.RoleTypeCode,
			Permissions:  pbIdentity.Actor.Permissions,
		}
	}

	return &Identity{
		Type:            convertIdentityTypeFromProto(pbIdentity.Type),
		TargetAccountID: pbIdentity.TargetAccountId,
		Actor:           actor,
		AccountMode:     convertAccountModeFromProto(pbIdentity.AccountMode),
	}
}

func convertIdentityTypeFromProto(t pb.IdentityType) IdentityType {
	switch t {
	case pb.IdentityType_IDENTITY_TYPE_USER:
		return IdentityTypeUser
	case pb.IdentityType_IDENTITY_TYPE_API_KEY:
		return IdentityTypeAPIKey
	case pb.IdentityType_IDENTITY_TYPE_UNAUTHENTICATED:
		return IdentityTypeUnauthenticated
	default:
		return IdentityTypeUnauthenticated
	}
}

func convertIdentityActorTypeFromProto(t pb.IdentityActorType) IdentityActorType {
	switch t {
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_INTERNAL:
		return IdentityActorTypeInternal
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_CUSTOMER:
		return IdentityActorTypeCustomer
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_SUPPLIER:
		return IdentityActorTypeSupplier
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNASSIGNED:
		return IdentityActorTypeUnassigned
	default:
		return IdentityActorTypeUnassigned
	}
}

func convertAccountModeFromProto(m pb.AccountMode) constants.AccountMode {
	switch m {
	case pb.AccountMode_ACCOUNT_MODE_PRODUCTION:
		return constants.AccountModeProduction
	case pb.AccountMode_ACCOUNT_MODE_SANDBOX:
		return constants.AccountModeSandbox
	default:
		return constants.AccountModeProduction
	}
}
