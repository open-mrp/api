package resourcekit

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// DefaultMaxIncludeDepth caps the recursion depth of include resolution, which bounds an include key at that many dot-separated segments. Cycles in the resource graph (e.g. child_accounts.parent_account) are also bounded by per-request memoization, but the depth cap protects against pathological client requests that pile up include paths. Clients cannot reach it on their own — every endpoint whitelists its include keys, and IncludesFor rejects a whitelisted key deeper than this at startup.
const DefaultMaxIncludeDepth = 8

// ResolveIncludes walks `tree` against the SubFields registered for
// `objectType`, batches loader calls per (level, target), and assigns the
// loaded children back onto `roots`.
//
// `roots` is a slice of `*Resource` pointer values typed as `any`. The
// resolver never inspects the concrete type — the SubField closures do that.
//
// The function is safe to call with an empty or nil tree (no-op) and with an
// empty roots slice (no-op).
func ResolveIncludes(ctx context.Context, roots []any, objectType constants.ObjectType, tree *IncludeNode) *apierror.APIError {
	return resolveIncludesAt(ctx, roots, objectType, tree, 0)
}

type loaderPlan struct {
	sub       *SubField
	ids       []string
	childTree *IncludeNode
}

func resolveIncludesAt(ctx context.Context, roots []any, objectType constants.ObjectType, tree *IncludeNode, depth int) *apierror.APIError {
	if !tree.HasChildren() || len(roots) == 0 {
		return nil
	}
	if depth >= DefaultMaxIncludeDepth {
		return apierror.NewInvariantViolationError(fmt.Sprintf(
			"resourcekit: include depth limit (%d) exceeded resolving %s",
			DefaultMaxIncludeDepth, objectType,
		))
	}
	def := Lookup(objectType)
	if def == nil {
		return apierror.NewInvariantViolationError(fmt.Sprintf(
			"resourcekit: no Definition registered for %s", objectType,
		))
	}
	cache := getOrCreateCache(ctx)

	var loaderPlans []loaderPlan
	missingByTarget := map[constants.ObjectType]map[string]struct{}{}
	for i := range def.Subs {
		sub := &def.Subs[i]
		if !tree.Has(sub.Key) {
			continue
		}
		if sub.Target == "" {
			continue
		}
		if Lookup(sub.Target) == nil {
			return apierror.NewInvariantViolationError(fmt.Sprintf(
				"resourcekit: %s sub %q targets unregistered %s",
				objectType, sub.Key, sub.Target,
			))
		}
		if sub.ExtractRefs != nil {
			continue
		}

		// Gather + dedup IDs across all roots.
		idSet := map[string]struct{}{}
		for _, r := range roots {
			for _, id := range sub.ExtractIDs(ctx, r) {
				if id != "" {
					idSet[id] = struct{}{}
				}
			}
		}
		// Split into cached vs. missing — cached entries skip the loader.
		ids := make([]string, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
			if _, cached := cache.get(sub.Target, id); !cached {
				if missingByTarget[sub.Target] == nil {
					missingByTarget[sub.Target] = map[string]struct{}{}
				}
				missingByTarget[sub.Target][id] = struct{}{}
			}
		}
		loaderPlans = append(loaderPlans, loaderPlan{sub: sub, ids: ids, childTree: tree.Child(sub.Key)})
	}

	if len(missingByTarget) > 0 {
		g, gctx := errgroup.WithContext(ctx)
		for target, idSet := range missingByTarget {
			target := target
			ids := make([]string, 0, len(idSet))
			for id := range idSet {
				ids = append(ids, id)
			}
			targetDef := Lookup(target)
			g.Go(func() error {
				fresh, apiErr := targetDef.Load(gctx, ids)
				if apiErr != nil {
					return apiErr
				}
				for id, v := range fresh {
					cache.set(target, id, v)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			if apiErr, ok := err.(*apierror.APIError); ok {
				return apiErr
			}
			return apierror.NewInternalError(err, "resourcekit: include load failed")
		}
	}

	planIdx := 0
	for i := range def.Subs {
		sub := &def.Subs[i]
		if !tree.Has(sub.Key) {
			continue
		}

		// No fetch: Populate runs with an empty loaded map. Used for include
		// keys that just toggle visibility on data the parent's loader already
		// supplied (e.g. `owner` exposing the deterministic type derived from
		// the parent's account_id).
		if sub.Target == "" {
			empty := map[string]any{}
			for _, r := range roots {
				sub.Populate(ctx, r, empty)
			}
			continue
		}

		// Traversal: the child objects are already on the parent (or will
		// be after Populate runs) and just need recursive include resolution.
		if sub.ExtractRefs != nil {
			if sub.Populate != nil {
				empty := map[string]any{}
				for _, r := range roots {
					sub.Populate(ctx, r, empty)
				}
			}
			childTree := tree.Child(sub.Key)
			if childTree.HasChildren() {
				var childRoots []any
				for _, r := range roots {
					childRoots = append(childRoots, sub.ExtractRefs(ctx, r)...)
				}
				if len(childRoots) > 0 {
					if apiErr := resolveIncludesAt(ctx, childRoots, sub.Target, childTree, depth+1); apiErr != nil {
						return apiErr
					}
				}
			}
			continue
		}

		plan := loaderPlans[planIdx]
		planIdx++

		loaded := make(map[string]any, len(plan.ids))
		for _, id := range plan.ids {
			if v, ok := cache.get(sub.Target, id); ok {
				loaded[id] = v
			}
		}

		// Recurse into nested includes before stitching, so children carry
		// their grandchildren by the time we attach them to parents.
		if plan.childTree.HasChildren() && len(loaded) > 0 {
			childRoots := make([]any, 0, len(loaded))
			for _, v := range loaded {
				childRoots = append(childRoots, v)
			}
			if apiErr := resolveIncludesAt(ctx, childRoots, sub.Target, plan.childTree, depth+1); apiErr != nil {
				return apiErr
			}
		}

		// Stitch loaded children back onto every root. Roots whose
		// ExtractIDs returned nothing remain unset (field stays nil/empty).
		for _, r := range roots {
			sub.Populate(ctx, r, loaded)
		}
	}
	return nil
}
