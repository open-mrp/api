// Package recordstub loads `record` stubs: the `{id, object: "record", type, number, status, metadata}` references
// that `related` groups and polymorphic `record` fields carry instead of whole resources.
//
// Each record type registers its own stub loader from an init() in its own file, so no unit edits a shared list:
//
//	func init() {
//		recordstub.Register(constants.RecordTypeJob, loadJobStubs)
//	}
//
// Stubs reach a response through the include engine. A field that points at one record type uses One or List with
// that type; a polymorphic field stores each reference's type next to its ID and passes both to Key. Either way the
// include target is constants.ObjectTypeRecord, whose single loader (Load) groups the keys by type and calls each
// type's loader once.
package recordstub

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"
	"github.com/open-mrp/apikit/object"
)

// Loader returns stubs for the given IDs of one record type, keyed by ID. An ID that no longer exists is left out.
type Loader func(ctx context.Context, ids []string) (map[string]*apiresource.Record, *apierror.APIError)

var (
	mu      sync.RWMutex
	loaders = map[constants.RecordType]Loader{}
)

// Register sets the stub loader for one record type. It panics on an unknown type or a second registration, both of which are wiring mistakes.
func Register(t constants.RecordType, l Loader) {
	if !t.IsValid() {
		panic(fmt.Sprintf("recordstub: %q is not a record type", t))
	}
	if l == nil {
		panic(fmt.Sprintf("recordstub: nil loader for %s", t))
	}
	mu.Lock()
	defer mu.Unlock()
	if _, dup := loaders[t]; dup {
		panic(fmt.Sprintf("recordstub: duplicate loader for %s", t))
	}
	loaders[t] = l
}

// Registered reports whether a record type has a stub loader.
func Registered(t constants.RecordType) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := loaders[t]
	return ok
}

// Key is the include-engine ID of one record: its type and ID, so references of different types batch together.
func Key(t constants.RecordType, id string) string {
	return string(t) + ":" + id
}

// ParseKey splits a Key back into its type and ID.
func ParseKey(key string) (constants.RecordType, string, bool) {
	t, id, ok := strings.Cut(key, ":")
	if !ok || id == "" {
		return "", "", false
	}
	return constants.RecordType(t), id, true
}

// Load is the include loader for constants.ObjectTypeRecord. It groups keys by record type, calls each type's loader once and in parallel, and returns the stubs keyed by Key. A key whose type has no loader is an invariant violation: the field pointing at it was registered before its type's loader.
func Load(ctx context.Context, keys []string) (map[string]any, *apierror.APIError) {
	byType := map[constants.RecordType][]string{}
	for _, k := range keys {
		t, id, ok := ParseKey(k)
		if !ok {
			return nil, apierror.NewInvariantViolationError(fmt.Sprintf("recordstub: malformed record key %q", k))
		}
		if !slices.Contains(byType[t], id) {
			byType[t] = append(byType[t], id)
		}
	}

	mu.RLock()
	calls := make(map[constants.RecordType]Loader, len(byType))
	for t := range byType {
		l, ok := loaders[t]
		if !ok {
			mu.RUnlock()
			return nil, apierror.NewInvariantViolationError(fmt.Sprintf("recordstub: no stub loader registered for %s", t))
		}
		calls[t] = l
	}
	mu.RUnlock()

	var (
		wg    sync.WaitGroup
		outMu sync.Mutex
		out   = make(map[string]any, len(keys))
		first *apierror.APIError
	)
	for _, t := range slices.Sorted(maps.Keys(calls)) {
		wg.Go(func() {
			stubs, apiErr := calls[t](ctx, byType[t])
			outMu.Lock()
			defer outMu.Unlock()
			if apiErr != nil {
				if first == nil {
					first = apiErr
				}
				return
			}
			for id, s := range stubs {
				if s == nil {
					continue
				}
				s.Object = constants.ObjectTypeRecord
				s.Type = t
				if s.ID == "" {
					s.ID = id
				}
				out[Key(t, id)] = s
			}
		})
	}
	wg.Wait()
	if first != nil {
		return nil, first
	}
	return out, nil
}

// One is the stub for one record from the include engine's loaded map, or nil when the record is gone.
func One(loaded map[string]any, t constants.RecordType, id string) *apiresource.Record {
	if id == "" {
		return nil
	}
	if s, ok := loaded[Key(t, id)].(*apiresource.Record); ok {
		return s
	}
	return nil
}

// List is the stubs for ids, in order, skipping records that are gone. It returns nil when none are left, so the field reads null rather than an empty list the caller never asked to see.
func List(loaded map[string]any, t constants.RecordType, ids []string) []apiresource.Record {
	var out []apiresource.Record
	for _, id := range ids {
		if s := One(loaded, t, id); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// SubOne is the include sub-field for a field naming one record of type t. id returns the record's ID on parent (or ""), and set writes the stub.
func SubOne(key string, t constants.RecordType, id func(ctx context.Context, parent any) string, set func(parent any, stub *apiresource.Record)) include.SubField {
	return include.SubField{
		Key:         key,
		Target:      constants.ObjectTypeRecord,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if v := id(ctx, parent); v != "" {
				return []string{Key(t, v)}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			if s := One(loaded, t, id(ctx, parent)); s != nil {
				set(parent, s)
			}
		},
	}
}

// SubRefs is the include sub-field for a field naming any number of records of any types. refs returns the (type, ID) pairs on parent in order, and set receives their stubs, skipping records that are gone.
func SubRefs(key string, cardinality include.Cardinality, refs func(ctx context.Context, parent any) []Ref, set func(parent any, stubs []apiresource.Record)) include.SubField {
	return include.SubField{
		Key:         key,
		Target:      constants.ObjectTypeRecord,
		Cardinality: cardinality,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			rs := refs(ctx, parent)
			keys := make([]string, 0, len(rs))
			for _, r := range rs {
				if r.ID != "" {
					keys = append(keys, Key(r.Type, r.ID))
				}
			}
			return keys
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			var stubs []apiresource.Record
			for _, r := range refs(ctx, parent) {
				if s := One(loaded, r.Type, r.ID); s != nil {
					stubs = append(stubs, *s)
				}
			}
			if len(stubs) > 0 {
				set(parent, stubs)
			}
		},
	}
}

// Ref names one record by type and ID.
type Ref struct {
	Type constants.RecordType
	ID   string
}

// FromResource is a stub loader built on a resource's own include loader: it loads the full resources by ID and keeps what stub returns for each. Use it while a record type has no lighter RPC; the resource's Definition is looked up on each call, so registration order doesn't matter.
func FromResource[T any](objectType object.Type, stub func(ctx context.Context, r *T) *apiresource.Record) Loader {
	return func(ctx context.Context, ids []string) (map[string]*apiresource.Record, *apierror.APIError) {
		def := include.Lookup(objectType)
		if def == nil {
			return nil, apierror.NewInvariantViolationError(fmt.Sprintf("recordstub: %s has no include definition", objectType))
		}
		loaded, apiErr := def.Load(ctx, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		out := make(map[string]*apiresource.Record, len(loaded))
		for id, v := range loaded {
			r, ok := v.(*T)
			if !ok {
				return nil, apierror.NewInvariantViolationError(fmt.Sprintf("recordstub: %s loader returned %T", objectType, v))
			}
			if s := stub(ctx, r); s != nil {
				out[id] = s
			}
		}
		return out, nil
	}
}
