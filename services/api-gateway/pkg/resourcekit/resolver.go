package resourcekit

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/open-mrp/api/shared/appctx"
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
//
// Every load runs under WithIncludeReads: the request that returned roots
// already authorized everything it includes.
func ResolveIncludes(ctx context.Context, roots []any, objectType constants.ObjectType, tree *IncludeNode) *apierror.APIError {
	return resolveIncludesAt(WithIncludeReads(ctx), roots, objectType, tree, 0)
}

// WithIncludeReads returns ctx with the caller's identity as it loads what an authorized request includes or embeds (types.Identity.ForIncludeReads). Never use it for the request's own records.
func WithIncludeReads(ctx context.Context) context.Context {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || identity.IncludeReads {
		return ctx
	}
	return appctx.WithIdentity(ctx, identity.ForIncludeReads())
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

	// Prefetch: union the IDs every loader sub can see now and load each target
	// concurrently. A sub whose ExtractIDs reads a field an earlier sub's
	// Populate sets (volume_discount categories.properties) sees nothing here;
	// the in-order pass below re-extracts and loads what the prefetch missed.
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

		for id := range extractIDSet(ctx, sub, roots) {
			if _, cached := cache.get(sub.Target, id); !cached {
				if missingByTarget[sub.Target] == nil {
					missingByTarget[sub.Target] = map[string]struct{}{}
				}
				missingByTarget[sub.Target][id] = struct{}{}
			}
		}
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

		idSet := extractIDSet(ctx, sub, roots)
		loaded := make(map[string]any, len(idSet))
		var missing []string
		for id := range idSet {
			if v, ok := cache.get(sub.Target, id); ok {
				loaded[id] = v
				continue
			}
			missing = append(missing, id)
		}
		if len(missing) > 0 {
			fresh, apiErr := Lookup(sub.Target).Load(ctx, missing)
			if apiErr != nil {
				return apiErr
			}
			for id, v := range fresh {
				loaded[id] = v
				cache.set(sub.Target, id, v)
			}
		}

		// Recurse into nested includes before stitching, so children carry
		// their grandchildren by the time we attach them to parents.
		childTree := tree.Child(sub.Key)
		if childTree.HasChildren() && len(loaded) > 0 {
			childRoots := make([]any, 0, len(loaded))
			for _, v := range loaded {
				childRoots = append(childRoots, v)
			}
			if apiErr := resolveIncludesAt(ctx, childRoots, sub.Target, childTree, depth+1); apiErr != nil {
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

// extractIDSet gathers and dedups sub's non-empty IDs across all roots.
func extractIDSet(ctx context.Context, sub *SubField, roots []any) map[string]struct{} {
	idSet := map[string]struct{}{}
	for _, r := range roots {
		for _, id := range sub.ExtractIDs(ctx, r) {
			if id != "" {
				idSet[id] = struct{}{}
			}
		}
	}
	return idSet
}
