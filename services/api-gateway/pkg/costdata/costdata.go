// Package costdata registers OpenMRP's two classes of restricted response data with apikit's sensitive package: the
// seller's cost data (tag `sensitive:"cost"`, and `sensitive:"cost_keys"` on a map whose cost-named keys are
// restricted), visible only to callers who may read costs, and internal data (`sensitive:"internal"`), visible only
// to the account's own actors. Cost data is also kept out of request logs. Importing the package registers both
// classes; apiresource imports it, so every endpoint does.
package costdata

import (
	"context"
	"reflect"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/redact"
	"github.com/open-mrp/apikit/sensitive"
)

const (
	// TagCost marks a field holding the seller's cost data.
	TagCost = "cost"
	// TagCostKeys marks a map whose keys name what they hold, so the entries whose key reads as cost data (IsCostName) are restricted.
	TagCostKeys = "cost_keys"
	// TagInternal marks a field only the account's own actors may see.
	TagInternal = "internal"
)

var costNameTokens = map[string]bool{
	"cost": true, "costs": true, "cogs": true, "margin": true, "margins": true,
	"profit": true, "profits": true, "valuation": true, "markup": true,
}

var costNames = map[string]bool{
	"labor_rate":            true,
	"overhead_rate":         true,
	"changeover_labor_rate": true,
	"inventory_value":       true,
}

// IsCostName reports whether a snake_case field name reads as the seller's cost or margin data.
func IsCostName(name string) bool {
	if costNames[name] {
		return true
	}
	for token := range strings.SplitSeq(name, "_") {
		if costNameTokens[token] {
			return true
		}
	}
	return false
}

func identity(ctx context.Context) *types.Identity {
	id, _ := appctx.Identity[*types.Identity](ctx)
	return id
}

func init() {
	sensitive.Register(sensitive.Class{
		Tag:      TagCost,
		KeyedTag: TagCostKeys,
		MatchKey: IsCostName,
		Visible: func(ctx context.Context) bool {
			id := identity(ctx)
			return id != nil && id.CanReadCosts()
		},
	})
	sensitive.Register(sensitive.Class{
		Tag: TagInternal,
		Visible: func(ctx context.Context) bool {
			id := identity(ctx)
			return id != nil && id.IsInternalActor()
		},
	})
	redact.RegisterSensitiveTag(TagCost)
	redact.RegisterKeyedMapTag(TagCostKeys, IsCostName)
}

// Visible reports whether the caller in ctx may see cost data.
func Visible(ctx context.Context) bool {
	return sensitive.Visible(ctx, TagCost)
}

// HasCostFields reports whether a value of type t can carry cost data.
func HasCostFields(t reflect.Type) bool {
	return sensitive.HasFields(t, TagCost)
}

// HasInternalFields reports whether a value of type t can carry internal data.
func HasInternalFields(t reflect.Type) bool {
	return sensitive.HasFields(t, TagInternal)
}

// IsCostField reports whether sf is tagged as cost data.
func IsCostField(sf reflect.StructField) bool {
	return sf.Tag.Get(sensitive.TagKey) == TagCost
}

// IsCostKeysField reports whether sf is a map whose cost-named keys are cost data.
func IsCostKeysField(sf reflect.StructField) bool {
	return sf.Tag.Get(sensitive.TagKey) == TagCostKeys
}

// IsInternalField reports whether sf is tagged as internal data.
func IsInternalField(sf reflect.StructField) bool {
	return sf.Tag.Get(sensitive.TagKey) == TagInternal
}
