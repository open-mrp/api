package types

import (
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/auth"
	apierror "github.com/open-mrp/apikit/apierror"
)

func TestNewSystemIdentity_AttributesToTheSourceAndTarget(t *testing.T) {
	t.Parallel()
	identity := NewSystemIdentity(constants.SystemActorStripe, "ac_target")

	if !identity.IsSystem() {
		t.Fatal("a system identity must report IsSystem")
	}
	if identity.Actor == nil || identity.Actor.ID != "system:stripe" {
		t.Fatalf("actor ID must name the source, got %+v", identity.Actor)
	}
	if !identity.IsTargetAccountSet() || identity.Target.AccountID != "ac_target" {
		t.Fatalf("target account must be the one given, got %+v", identity.Target)
	}
	if identity.ActorAccountID() != nil {
		t.Fatal("a system identity belongs to no account")
	}
}

// A system identity exists so a background write can be audited. If it satisfied any check, a
// forged one sent over internal metadata would be a skeleton key, so every gate must refuse it.
func TestNewSystemIdentity_PassesNoAuthorizationCheck(t *testing.T) {
	t.Parallel()
	identity := NewSystemIdentity(constants.SystemActorScheduler, "ac_target")
	// Even a forged permission map grants nothing: the identity is not authenticated.
	identity.Actor.Permissions = map[string]bool{"products:read": true}
	identity.Actor.RoleType = new(string(constants.RoleTypeAdmin))

	checks := map[string]func() bool{
		"IsAuthenticated":       identity.IsAuthenticated,
		"IsActorSet":            identity.IsActorSet,
		"IsAdmin":               identity.IsAdmin,
		"IsUser":                identity.IsUser,
		"IsAPIKey":              identity.IsAPIKey,
		"IsAgent":               identity.IsAgent,
		"HasUserActor":          identity.HasUserActor,
		"IsInternalActor":       identity.IsInternalActor,
		"IsInternalUser":        identity.IsInternalUser,
		"IsRelationActor":       identity.IsRelationActor,
		"IsIncludeRead":         identity.ForIncludeReads().IsIncludeRead,
		"IsExternalTarget":      identity.IsExternalTarget,
		"IsCustomerUser":        identity.IsCustomerUser,
		"IsUnassignedUser":      identity.IsUnassignedUser,
		"IsTargetCustomer":      identity.IsTargetCustomerAccount,
		"IsTargetSupplier":      identity.IsTargetSupplierAccount,
		"IsSalesRepOrScanner":   func() bool { return identity.IsSalesRep() || identity.IsScanner() },
		"IsSupplierUser":        identity.IsSupplierUser,
		"IsRoleSet":             identity.IsRoleSet,
		"PortalAccountIDNotNil": func() bool { return identity.PortalAccountID() != nil },
	}
	for name, check := range checks {
		if check() {
			t.Errorf("%s must be false for a system identity", name)
		}
	}

	errs := map[string]func() *apierror.APIError{
		"CheckIsAuthenticated": func() *apierror.APIError { return identity.CheckIsAuthenticated() },
		"CheckIsAssignedActor": func() *apierror.APIError { return identity.CheckIsAssignedActor() },
		"CheckIsInternalActor": func() *apierror.APIError { return identity.CheckIsInternalActor() },
		"CheckIsAdmin":         func() *apierror.APIError { return identity.CheckIsAdmin() },
		"CheckIsUser":          func() *apierror.APIError { return identity.CheckIsUser() },
		"CheckHasUserActor":    func() *apierror.APIError { return identity.CheckHasUserActor() },
		"CheckIsAPIKey":        func() *apierror.APIError { return identity.CheckIsAPIKey() },
		"CheckIsAgent":         func() *apierror.APIError { return identity.CheckIsAgent() },
		"CheckAPIKeyAccess":    func() *apierror.APIError { return identity.CheckAPIKeyAccess() },
		"CheckHasPermission":   func() *apierror.APIError { return identity.CheckHasPermission(PermissionDomainProducts, ActionRead) },
		"CheckHasRoleType":     func() *apierror.APIError { return identity.CheckHasRoleType(constants.RoleTypeAdmin) },
		"CheckHasRelationCap": func() *apierror.APIError {
			return identity.CheckHasRelationCapability(PermissionDomainProducts, ActionRead)
		},
		"CheckIsInternalForRead": func() *apierror.APIError { return identity.ForIncludeReads().CheckIsInternalActorForRead() },
	}
	for name, check := range errs {
		if check() == nil {
			t.Errorf("%s must refuse a system identity", name)
		}
	}
}

// The gateway builds identities from the auth-service protocol, which has no system value, so no
// response from it can turn into a system identity.
func TestIdentityFromProto_NeverProducesASystemIdentity(t *testing.T) {
	t.Parallel()
	for value := range pb.IdentityActorType_name {
		got := IdentityFromProto(&pb.Identity{Type: pb.IdentityActorType(value)})
		if got.IsSystem() {
			t.Errorf("proto actor type %d decoded to a system identity", value)
		}
	}
	if got := IdentityFromProto(&pb.Identity{Type: pb.IdentityActorType(999)}); got.IsSystem() {
		t.Error("an unknown proto actor type decoded to a system identity")
	}
}

// Services pass identity to each other as JSON, so a system identity has to survive the trip to
// stay attributed.
func TestNewSystemIdentity_SurvivesJSONPropagation(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(NewSystemIdentity(constants.SystemActorEmailIngest, "ac_target"))
	if err != nil {
		t.Fatal(err)
	}
	var got Identity
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !got.IsSystem() || got.Actor.ID != "system:email_ingest" || got.Target.AccountID != "ac_target" {
		t.Errorf("round-tripped identity lost its attribution: %+v", got)
	}
	if got.IsAuthenticated() {
		t.Error("a round-tripped system identity must still not be authenticated")
	}
}
