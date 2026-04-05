package apiresource

import "github.com/augno/api/shared/constants"

// OwnerAccount is a minimal account reference within an Owner.
type OwnerAccount struct {
	// The unique identifier for the account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
}

// Owner describes the provenance of a resource.
type Owner struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=owner"`
	// The owner type: "system" for platform defaults, "account" for account-owned resources.
	Type constants.OwnerType `json:"type" validate:"required"`
	// The account that owns this resource. Present when type is "account".
	Account *OwnerAccount `json:"account"`
}

// SystemOwner returns an Owner representing a system-provided default.
func SystemOwner() *Owner {
	return &Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeSystem,
	}
}

// NewOwner constructs an Owner from an optional account ID.
// A nil or empty account ID produces a system owner; otherwise an account owner.
func NewOwner(accountID *string) *Owner {
	if accountID == nil || *accountID == "" {
		return SystemOwner()
	}
	return &Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeAccount,
		Account: &OwnerAccount{
			ID:     *accountID,
			Object: constants.ObjectTypeAccount,
		},
	}
}

var SampleOwnerSystem = &Owner{
	Object: constants.ObjectTypeOwner,
	Type:   constants.OwnerTypeSystem,
}

var SampleOwnerAccount = &Owner{
	Object: constants.ObjectTypeOwner,
	Type:   constants.OwnerTypeAccount,
	Account: &OwnerAccount{
		ID:     SampleAccountID,
		Object: constants.ObjectTypeAccount,
	},
}
