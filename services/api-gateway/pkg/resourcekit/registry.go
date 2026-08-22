// Package resourcekit is the canonical registry for API resources. Each resource
// type registers exactly one Definition that names its loader (BatchGet by IDs)
// and its expandable sub-resources. The runtime walks the registered graph to
// resolve includes, batch loader calls per level, and stitch results back onto
// parents without any per-endpoint code.
//
// See api/wip/presenters-refactor.md for the full architecture.
package resourcekit

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Cardinality describes the shape of a sub-resource field on a parent.
type Cardinality int

const (
	// CardinalityOnePtr — a `*T` field with a single FK id (or zero).
	CardinalityOnePtr Cardinality = iota
	// CardinalityList — a `*List[T]` field backed by `[]string` FK ids.
	CardinalityList
)

// Loader fetches resources by ID. The returned map is keyed by the resource ID
// (the same string that appeared in the input slice). Missing IDs are simply
// absent from the result — the resolver leaves the parent's sub-field unset.
//
// Values in the returned map MUST be `*T` (pointer to the resource type), so the
// resolver can mutate them in place when recursing into deeper includes.
type Loader func(ctx context.Context, ids []string) (map[string]any, *apierror.APIError)

// SubField declares one include-gated field on a parent resource.
//
// Every SubField has a Populate closure that runs only when the client
// requests this field's Key. The closure may use data the loader already
// put on the parent (Owner.Type, derived from the parent's account_id) or
// data the resolver fetched via Target's loader (a full Account record).
//
// When Target is set, the resolver first batches ExtractIDs across all
// roots, calls Lookup(Target).Load once, and passes the result to Populate
// as `loaded`. When Target is empty, no fetch happens and Populate runs
// with an empty `loaded` map — used for fields whose data is fully
// determined by the parent's own row (e.g. Carrier.Owner.Type comes from
// carrier.account_id being null or not; nothing is hallucinated).
//
// Both closures take ctx so they can read foreign-key info from the
// request-scoped LoadMeta side-table — the apiresource Go structs stay
// clean (no FK stowaway fields).
type SubField struct {
	// Key is the dot-separated include path, relative to the parent. Examples:
	// "owner", "owner.account", "service_levels". The runtime matches client
	// `?include[]=<key>` values against this exact string.
	Key string

	// Target is the ObjectType of the loaded child resource. When set, must
	// itself be a registered Definition — the resolver looks it up to find
	// the child loader. Leave empty for include keys that don't require a
	// fetch.
	Target constants.ObjectType

	// Cardinality declares whether the resulting field is a *T (OnePtr) or
	// *List[T] (List). Used by the OpenAPI generator and by codegen.
	Cardinality Cardinality

	// ExtractIDs returns the FK IDs to load for this sub-field on `parent`.
	// Required when Target is set; ignored otherwise. IDs are deduplicated
	// and batched with peers across all roots before the loader is invoked
	// exactly once per (level, target). The closure may read parent FK info
	// from GetLoadMeta(ctx).
	ExtractIDs func(ctx context.Context, parent any) []string

	// Populate writes data onto `parent`. Always called when the include
	// key is requested. `loaded` is keyed by ID and holds *Resource pointers
	// the resolver fetched via Target's loader; empty when Target is unset.
	// The closure may read parent FK info from GetLoadMeta(ctx).
	Populate func(ctx context.Context, parent any, loaded map[string]any)

	// ExtractRefs returns pointers to child objects already present on `parent`
	// for traversal-only resolution. When set, the resolver skips ExtractIDs
	// and Load — it uses the returned references as child roots for recursive
	// include resolution. Target must still be set so the resolver knows the
	// child Definition to recurse into. Populate is not called.
	ExtractRefs func(ctx context.Context, parent any) []any
}

// Definition is the single registry entry for one resource type.
type Definition struct {
	// ObjectType is the resource's identity in the registry.
	ObjectType constants.ObjectType

	// Load fetches base records by ID. Sub-resources on the returned objects
	// must be nil/empty; the resolver fills them when includes request them.
	Load Loader

	// Subs is the ordered list of expandable relations on this resource.
	// Order is preserved by the resolver for deterministic loader fan-out.
	Subs []SubField
}

var (
	registryMu sync.RWMutex
	registry   = map[constants.ObjectType]*Definition{}
)

// Register adds a Definition to the registry. Intended to be called from
// init() in per-resource registration files. Panics on duplicate ObjectType,
// because a duplicate would silently shadow real loaders.
func Register(d *Definition) {
	if d == nil {
		panic("resourcekit: Register called with nil Definition")
	}
	if d.ObjectType == "" {
		panic("resourcekit: Register called with empty ObjectType")
	}
	if d.Load == nil {
		panic(fmt.Sprintf("resourcekit: Register(%s) called with nil Load", d.ObjectType))
	}
	for i, sub := range d.Subs {
		if sub.Key == "" {
			panic(fmt.Sprintf("resourcekit: Register(%s) sub[%d] has empty Key", d.ObjectType, i))
		}
		if sub.ExtractRefs != nil {
			if sub.Target == "" {
				panic(fmt.Sprintf("resourcekit: Register(%s) sub[%d] (%s) has ExtractRefs but no Target", d.ObjectType, i, sub.Key))
			}
			continue
		}
		if sub.Populate == nil {
			panic(fmt.Sprintf("resourcekit: Register(%s) sub[%d] (%s) has nil Populate", d.ObjectType, i, sub.Key))
		}
		if sub.Target != "" && sub.ExtractIDs == nil {
			panic(fmt.Sprintf("resourcekit: Register(%s) sub[%d] (%s) has Target but nil ExtractIDs", d.ObjectType, i, sub.Key))
		}
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[d.ObjectType]; exists {
		panic(fmt.Sprintf("resourcekit: duplicate registration for %s", d.ObjectType))
	}
	registry[d.ObjectType] = d
}

// Lookup returns the registered Definition for an ObjectType, or nil if none.
func Lookup(ot constants.ObjectType) *Definition {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[ot]
}

// AllowedIncludeKeys returns the transitive set of include paths reachable
// from `root`, capped at maxDepth segments. Used by OpenAPI generation to
// emit the `include[]` enum per endpoint and by the request parser to
// validate client-supplied keys.
//
// Cycles in the resource graph are broken by tracking the set of ObjectTypes
// currently on the traversal path — a Definition can be revisited on a
// different branch but not on the same one.
func AllowedIncludeKeys(root constants.ObjectType, maxDepth int) []string {
	def := Lookup(root)
	if def == nil {
		return nil
	}
	seen := map[string]struct{}{}
	walking := map[constants.ObjectType]struct{}{root: {}}
	walkAllowedKeys("", def, walking, seen, maxDepth)
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func walkAllowedKeys(prefix string, def *Definition, walking map[constants.ObjectType]struct{}, seen map[string]struct{}, remaining int) {
	if remaining <= 0 || def == nil {
		return
	}
	for _, sub := range def.Subs {
		key := sub.Key
		if prefix != "" {
			key = prefix + "." + sub.Key
		}
		seen[key] = struct{}{}
		if _, busy := walking[sub.Target]; busy {
			continue
		}
		childDef := Lookup(sub.Target)
		if childDef == nil {
			continue
		}
		walking[sub.Target] = struct{}{}
		walkAllowedKeys(key, childDef, walking, seen, remaining-1)
		delete(walking, sub.Target)
	}
}

// ResetForTest clears registry state between test cases. Production code
// must never call this.
func ResetForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[constants.ObjectType]*Definition{}
}
