package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Owner describes the provenance of a resource.
type Owner struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=owner"`
	// The owner type: "system" for platform defaults, "account" for account-owned resources.
	Type constants.OwnerType `json:"type" validate:"required"`
	// The account that owns this resource. `null` if the object is system-owned.
	Account *Account `json:"account" expandable:"true"`
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
		Account: &Account{
			ID:     *accountID,
			Object: constants.ObjectTypeAccount,
		},
	}
}

// NewOwnerWithAccount constructs an Owner from an optional account ID and a
// resolved Account. When account is non-nil it is used directly; otherwise the
// Owner falls back to a stub Account containing only ID and Object.
func NewOwnerWithAccount(accountID *string, account *Account) *Owner {
	if accountID == nil || *accountID == "" {
		return SystemOwner()
	}
	if account != nil {
		return &Owner{
			Object:  constants.ObjectTypeOwner,
			Type:    constants.OwnerTypeAccount,
			Account: account,
		}
	}
	return NewOwner(accountID)
}

var SampleOwnerSystem = &Owner{
	Object: constants.ObjectTypeOwner,
	Type:   constants.OwnerTypeSystem,
}

var SampleOwnerAccount = &Owner{
	Object: constants.ObjectTypeOwner,
	Type:   constants.OwnerTypeAccount,
	Account: &Account{
		ID:        SampleAccountID,
		Object:    constants.ObjectTypeAccount,
		Name:      SampleAccountName,
		CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	},
}

func (*Owner) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOwnerAccount)
}
