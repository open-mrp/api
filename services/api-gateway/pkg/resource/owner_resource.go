package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

// Owner describes the provenance of a resource.
type Owner struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=owner"`
	// Where this resource came from.
	//
	// - `system`: a platform-provided default shared across all accounts; not editable.
	// - `account`: created and owned by a specific account; the `account` field identifies which.
	Type constants.OwnerType `json:"type" validate:"required"`
	// The account that owns this resource.
	//
	// Present only when `type` is `account`; system-owned resources have no owning account.
	Account *Account `json:"account" expandable:"true"`
}

// SystemOwner returns an Owner representing a system-provided default.
func SystemOwner() *Owner {
	return &Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeSystem,
	}
}

// NewOwner constructs an Owner from an optional account ID. A nil or empty account ID produces a system owner; otherwise an account owner.
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

// NewOwnerWithAccount constructs an Owner from an optional account ID and a resolved Account. When account is non-nil it is used directly; otherwise the Owner falls back to a stub Account containing only ID and Object.
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
