package types

import (
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
)

type IdentityActor struct {
	RelationType IdentityRelationType
	ID           string
	Name         *string
	AccountID    *string
	RoleID       *string
	RoleType     *string
	RoleName     *string
	Permissions  map[string]bool
}

type Identity struct {
	Type               IdentityActorType
	Target             *IdentityTarget
	Actor              *IdentityActor
	AccountMode        constants.AccountMode
	SubscriptionStatus *string
}

// IsAuthenticated checks that the identity exists and is not unauthenticated
func (i *Identity) IsAuthenticated() bool {
	return i != nil && i.Type != IdentityActorTypeUnauthenticated
}

// IsActorSet checks that the identity is authenticated, has a valid actor, and has a valid actor account
func (i *Identity) IsActorSet() bool {
	return i.IsAuthenticated() && i.Actor != nil && i.Actor.ID != "" && i.Actor.AccountID != nil && *i.Actor.AccountID != ""
}

// IsRoleSet checks that the identity has a valid actor and a valid role
func (i *Identity) IsRoleSet() bool {
	return i.IsActorSet() && i.Actor.RoleType != nil && *i.Actor.RoleType != ""
}

// IsAdmin checks that the identity is authenticated, has a valid actor, a valid role, and is of type admin
func (i *Identity) IsAdmin() bool {
	return i.IsRoleSet() && *i.Actor.RoleType == string(constants.RoleTypeAdmin)
}

// IsScanner checks that the identity is authenticated, has a valid actor, a valid role, and is of type scanner
func (i *Identity) IsScanner() bool {
	return i.IsRoleSet() && *i.Actor.RoleType == string(constants.RoleTypeScanner)
}

// IsSalesRep checks that the identity is authenticated, has a valid actor, a valid role, and is of type sales rep
func (i *Identity) IsSalesRep() bool {
	return i.IsRoleSet() && *i.Actor.RoleType == string(constants.RoleTypeSalesRep)
}

// IsSandbox checks that the target account is a sandbox
func (i *Identity) IsSandbox() bool {
	return i != nil && i.AccountMode == constants.AccountModeSandbox
}

// IsNotSandbox checks that the target account is not a sandbox
func (i *Identity) IsNotSandbox() bool {
	return i != nil && i.AccountMode != constants.AccountModeSandbox
}

// IsAPIKey checks that the identity is an API key and a valid actor was found
func (i *Identity) IsAPIKey() bool {
	return i.IsActorSet() && i.Type == IdentityActorTypeAPIKey
}

// IsUser checks that the identity is a user and a valid actor was found
func (i *Identity) IsUser() bool {
	return i.IsActorSet() && i.Type == IdentityActorTypeUser
}

// HasUserActor checks that the identity is an authenticated user with a valid
// actor ID, regardless of whether an account is assigned. Unlike IsUser, this
// does not require an actor account, so it suits account-agnostic user
// endpoints (e.g. tenancy discovery, called before an account is selected).
func (i *Identity) HasUserActor() bool {
	return i.IsAuthenticated() && i.Type == IdentityActorTypeUser && i.Actor != nil && i.Actor.ID != ""
}

// IsInternalActor checks that the identity has a valid actor of type internal, regardless of target account.
func (i *Identity) IsInternalActor() bool {
	return i.IsActorSet() && i.Actor.RelationType == IdentityRelationTypeInternal
}

// IsRelationActor reports whether the actor reaches the target account through a
// customer or supplier account relation rather than as an internal member. Such
// actors may carry their OWN-account role permissions (used by downstream services
// to authorize customer-side capabilities, e.g. purchase_orders:create for a
// portal order); those permissions apply to the actor's own account, not the
// target owner's. Their RoleID/RoleType are cleared (no admin bypass). The coarse
// gateway permission gate declares owner-side domains, so it must not reject
// relation actors — their access is authorized in the downstream service.
func (i *Identity) IsRelationActor() bool {
	return i.IsActorSet() &&
		(i.Actor.RelationType == IdentityRelationTypeCustomer || i.Actor.RelationType == IdentityRelationTypeSupplier)
}

// IsInternalUser checks that the identity is authenticated, has a valid actor, is of type internal, and has a valid actor account
func (i *Identity) IsInternalUser() bool {
	return i.IsActorSet() && i.IsTargetAccountSet() && i.Actor.RelationType == IdentityRelationTypeInternal && *i.Actor.AccountID == i.Target.AccountID
}

// IsSupplierUser checks that the identity is authenticated, has a valid actor, is of type supplier
func (i *Identity) IsSupplierUser() bool {
	return i.IsActorSet() && i.IsTargetAccountSet() && i.Actor.RelationType == IdentityRelationTypeSupplier
}

// IsCustomerUser checks that the identity is authenticated, has a valid actor, is of type customer
func (i *Identity) IsCustomerUser() bool {
	return i.IsActorSet() && i.IsTargetAccountSet() && i.Actor.RelationType == IdentityRelationTypeCustomer
}

// IsUnassignedUser checks that the identity is authenticated, has a valid actor, and is of type unassigned
func (i *Identity) IsUnassignedUser() bool {
	return i.IsActorSet() && i.Actor.RelationType == IdentityRelationTypeUnassigned
}

// IsExternalTarget checks that the identity has a valid actor, a valid actor account, and a valid target account, and that the actor's account differs from the target account
func (i *Identity) IsExternalTarget() bool {
	return i.IsActorSet() && i.ActorAccountID() != nil && i.IsTargetAccountSet() && *i.ActorAccountID() != i.Target.AccountID
}

// IsTargetCustomerAccount returns true when the target account is a customer account relative to the actor.
func (i *Identity) IsTargetCustomerAccount() bool {
	return i.Target != nil && i.Target.RelationType != nil && *i.Target.RelationType == IdentityRelationTypeCustomer
}

// IsTargetSupplierAccount returns true when the target account is a supplier account relative to the actor.
func (i *Identity) IsTargetSupplierAccount() bool {
	return i.Target != nil && i.Target.RelationType != nil && *i.Target.RelationType == IdentityRelationTypeSupplier
}

// IsTargetAccountSet returns true when a target account ID is present.
func (i *Identity) IsTargetAccountSet() bool {
	return i != nil && i.Target != nil && i.Target.AccountID != ""
}

// ActorAccountID returns the actor's account ID, or nil if unavailable.
func (i *Identity) ActorAccountID() *string {
	if i == nil || i.Actor == nil || i.Actor.AccountID == nil || *i.Actor.AccountID == "" {
		return nil
	}
	return i.Actor.AccountID
}

func GetUnauthenticatedIdentity(targetAccountID *string) *Identity {
	return GetUnauthenticatedIdentityWithMode(targetAccountID, constants.AccountModeProduction)
}

func GetUnauthenticatedIdentityWithMode(targetAccountID *string, accountMode constants.AccountMode) *Identity {
	identity := &Identity{
		Type:        IdentityActorTypeUnauthenticated,
		Actor:       nil,
		AccountMode: accountMode,
	}
	if targetAccountID != nil {
		identity.Target = &IdentityTarget{AccountID: *targetAccountID}
	}
	return identity
}

func (i *Identity) ToProto() *pb.Identity {
	if i == nil {
		return nil
	}

	var actor *pb.IdentityActor
	if i.Actor != nil {
		actor = &pb.IdentityActor{
			RelationType: i.Actor.RelationType.toProto(),
			Id:           i.Actor.ID,
			Name:         i.Actor.Name,
			AccountId:    i.Actor.AccountID,
			RoleId:       i.Actor.RoleID,
			RoleTypeCode: i.Actor.RoleType,
			RoleName:     i.Actor.RoleName,
			Permissions:  i.Actor.Permissions,
		}
	}

	var target *pb.IdentityTarget
	if i.Target != nil {
		target = &pb.IdentityTarget{
			AccountId: i.Target.AccountID,
		}
		if i.Target.RelationType != nil {
			s := string(*i.Target.RelationType)
			target.RelationType = &s
		}
	}

	return &pb.Identity{
		Type:               i.Type.toProto(),
		Actor:              actor,
		AccountMode:        convertAccountModeToProto(i.AccountMode),
		SubscriptionStatus: i.SubscriptionStatus,
		Target:             target,
	}
}

func (t IdentityActorType) toProto() pb.IdentityActorType {
	switch t {
	case IdentityActorTypeUser:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_USER
	case IdentityActorTypeAPIKey:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_API_KEY
	case IdentityActorTypeUnauthenticated:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNAUTHENTICATED
	case IdentityActorTypeAgent:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_AGENT
	default:
		return pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNSPECIFIED
	}
}

func (t IdentityRelationType) toProto() pb.IdentityRelationType {
	switch t {
	case IdentityRelationTypeInternal:
		return pb.IdentityRelationType_IDENTITY_RELATION_TYPE_INTERNAL
	case IdentityRelationTypeCustomer:
		return pb.IdentityRelationType_IDENTITY_RELATION_TYPE_CUSTOMER
	case IdentityRelationTypeSupplier:
		return pb.IdentityRelationType_IDENTITY_RELATION_TYPE_SUPPLIER
	case IdentityRelationTypeUnassigned:
		return pb.IdentityRelationType_IDENTITY_RELATION_TYPE_UNASSIGNED
	default:
		return pb.IdentityRelationType_IDENTITY_RELATION_TYPE_UNSPECIFIED
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
			RelationType: convertIdentityRelationTypeFromProto(pbIdentity.Actor.RelationType),
			ID:           pbIdentity.Actor.Id,
			Name:         pbIdentity.Actor.Name,
			AccountID:    pbIdentity.Actor.AccountId,
			RoleID:       pbIdentity.Actor.RoleId,
			RoleType:     pbIdentity.Actor.RoleTypeCode,
			RoleName:     pbIdentity.Actor.RoleName,
			Permissions:  pbIdentity.Actor.Permissions,
		}
	}

	var target *IdentityTarget
	if pbIdentity.Target != nil {
		target = &IdentityTarget{
			AccountID: pbIdentity.Target.AccountId,
		}
		if pbIdentity.Target.RelationType != nil {
			parsed, ok := ParseIdentityRelationType(*pbIdentity.Target.RelationType)
			if ok {
				target.RelationType = &parsed
			}
		}
	}

	return &Identity{
		Type:               convertIdentityActorTypeFromProto(pbIdentity.Type),
		Target:             target,
		Actor:              actor,
		AccountMode:        convertAccountModeFromProto(pbIdentity.AccountMode),
		SubscriptionStatus: pbIdentity.SubscriptionStatus,
	}
}

func convertIdentityActorTypeFromProto(t pb.IdentityActorType) IdentityActorType {
	switch t {
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_USER:
		return IdentityActorTypeUser
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_API_KEY:
		return IdentityActorTypeAPIKey
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_UNAUTHENTICATED:
		return IdentityActorTypeUnauthenticated
	case pb.IdentityActorType_IDENTITY_ACTOR_TYPE_AGENT:
		return IdentityActorTypeAgent
	default:
		return IdentityActorTypeUnauthenticated
	}
}

func convertIdentityRelationTypeFromProto(t pb.IdentityRelationType) IdentityRelationType {
	switch t {
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_INTERNAL:
		return IdentityRelationTypeInternal
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_CUSTOMER:
		return IdentityRelationTypeCustomer
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_SUPPLIER:
		return IdentityRelationTypeSupplier
	case pb.IdentityRelationType_IDENTITY_RELATION_TYPE_UNASSIGNED:
		return IdentityRelationTypeUnassigned
	default:
		return IdentityRelationTypeUnassigned
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
