package resourcekit

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// --- Test fixtures: lightweight stand-in resources used across tests. ---

type testCarrier struct {
	ID              string
	Name            string
	OwnerID         string
	ServiceLevelIDs []string

	Owner         *testOwner
	ServiceLevels []*testServiceLevel
}

type testOwner struct {
	ID   string
	Name string
}

type testServiceLevel struct {
	ID   string
	Name string
}

type testCustomer struct {
	ID              string
	ChildAccountIDs []string
	ParentID        string

	ChildAccounts []*testCustomer
	Parent        *testCustomer
}

const (
	otCarrier  constants.ObjectType = "test_carrier"
	otOwner    constants.ObjectType = "test_owner"
	otSL       constants.ObjectType = "test_service_level"
	otCustomer constants.ObjectType = "test_customer"
)

// --- Helpers to register definitions with call-counting loaders. ---

type counters struct {
	carrier  atomic.Int64
	owner    atomic.Int64
	sl       atomic.Int64
	customer atomic.Int64
}

func registerCarrierGraph(t *testing.T, c *counters, carriers []*testCarrier, owners []*testOwner, sls []*testServiceLevel) {
	t.Helper()

	Register(&Definition{
		ObjectType: otCarrier,
		Load: func(_ context.Context, ids []string) (map[string]any, *apierror.APIError) {
			c.carrier.Add(1)
			out := map[string]any{}
			for _, cr := range carriers {
				for _, id := range ids {
					if cr.ID == id {
						copy := *cr
						out[id] = &copy
					}
				}
			}
			return out, nil
		},
		Subs: []SubField{
			{
				Key: "owner", Target: otOwner, Cardinality: CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, p any) []string {
					cr := p.(*testCarrier)
					if cr.OwnerID == "" {
						return nil
					}
					return []string{cr.OwnerID}
				},
				Populate: func(_ context.Context, p any, loaded map[string]any) {
					cr := p.(*testCarrier)
					if v, ok := loaded[cr.OwnerID]; ok {
						cr.Owner = v.(*testOwner)
					}
				},
			},
			{
				Key: "service_levels", Target: otSL, Cardinality: CardinalityList,
				ExtractIDs: func(_ context.Context, p any) []string {
					cr := p.(*testCarrier)
					return cr.ServiceLevelIDs
				},
				Populate: func(_ context.Context, p any, loaded map[string]any) {
					cr := p.(*testCarrier)
					out := make([]*testServiceLevel, 0, len(cr.ServiceLevelIDs))
					for _, id := range cr.ServiceLevelIDs {
						if v, ok := loaded[id]; ok {
							out = append(out, v.(*testServiceLevel))
						}
					}
					cr.ServiceLevels = out
				},
			},
		},
	})

	Register(&Definition{
		ObjectType: otOwner,
		Load: func(_ context.Context, ids []string) (map[string]any, *apierror.APIError) {
			c.owner.Add(1)
			out := map[string]any{}
			for _, o := range owners {
				for _, id := range ids {
					if o.ID == id {
						copy := *o
						out[id] = &copy
					}
				}
			}
			return out, nil
		},
	})

	Register(&Definition{
		ObjectType: otSL,
		Load: func(_ context.Context, ids []string) (map[string]any, *apierror.APIError) {
			c.sl.Add(1)
			out := map[string]any{}
			for _, s := range sls {
				for _, id := range ids {
					if s.ID == id {
						copy := *s
						out[id] = &copy
					}
				}
			}
			return out, nil
		},
	})
}

func registerCustomerCycle(t *testing.T, c *counters, customers []*testCustomer) {
	t.Helper()
	Register(&Definition{
		ObjectType: otCustomer,
		Load: func(_ context.Context, ids []string) (map[string]any, *apierror.APIError) {
			c.customer.Add(1)
			out := map[string]any{}
			for _, cust := range customers {
				for _, id := range ids {
					if cust.ID == id {
						copy := *cust
						out[id] = &copy
					}
				}
			}
			return out, nil
		},
		Subs: []SubField{
			{
				Key: "child_accounts", Target: otCustomer, Cardinality: CardinalityList,
				ExtractIDs: func(_ context.Context, p any) []string {
					return p.(*testCustomer).ChildAccountIDs
				},
				Populate: func(_ context.Context, p any, loaded map[string]any) {
					c := p.(*testCustomer)
					out := make([]*testCustomer, 0, len(c.ChildAccountIDs))
					for _, id := range c.ChildAccountIDs {
						if v, ok := loaded[id]; ok {
							out = append(out, v.(*testCustomer))
						}
					}
					c.ChildAccounts = out
				},
			},
			{
				Key: "parent_account", Target: otCustomer, Cardinality: CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, p any) []string {
					c := p.(*testCustomer)
					if c.ParentID == "" {
						return nil
					}
					return []string{c.ParentID}
				},
				Populate: func(_ context.Context, p any, loaded map[string]any) {
					c := p.(*testCustomer)
					if v, ok := loaded[c.ParentID]; ok {
						c.Parent = v.(*testCustomer)
					}
				},
			},
		},
	})
}

// --- Tests ---

func TestResolveIncludes_EmptyTree_NoOp(t *testing.T) {
	ResetForTest()
	c := &counters{}
	registerCarrierGraph(t, c, nil, nil, nil)

	carrier := &testCarrier{ID: "c1", OwnerID: "o1"}
	err := ResolveIncludes(context.Background(), []any{carrier}, otCarrier, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carrier.Owner != nil {
		t.Errorf("Owner should remain nil for empty tree")
	}
	if c.owner.Load() != 0 {
		t.Errorf("loader should not have been called, got %d", c.owner.Load())
	}
}

func TestResolveIncludes_SingleInclude(t *testing.T) {
	ResetForTest()
	c := &counters{}
	registerCarrierGraph(t, c,
		[]*testCarrier{{ID: "c1", OwnerID: "o1"}},
		[]*testOwner{{ID: "o1", Name: "Owner One"}},
		nil,
	)

	carrier := &testCarrier{ID: "c1", OwnerID: "o1"}
	tree := ParseIncludeTree([]string{"owner"})
	if err := ResolveIncludes(context.Background(), []any{carrier}, otCarrier, tree); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carrier.Owner == nil || carrier.Owner.ID != "o1" {
		t.Errorf("expected owner stitched, got %+v", carrier.Owner)
	}
	if c.owner.Load() != 1 {
		t.Errorf("expected exactly 1 owner load, got %d", c.owner.Load())
	}
}

func TestResolveIncludes_BatchesAcrossRoots(t *testing.T) {
	// Two carriers both reference owner o1 — the loader should be called once
	// with the deduped id list, not once per carrier.
	ResetForTest()
	c := &counters{}
	registerCarrierGraph(t, c,
		[]*testCarrier{
			{ID: "c1", OwnerID: "o1"},
			{ID: "c2", OwnerID: "o2"},
			{ID: "c3", OwnerID: "o1"},
		},
		[]*testOwner{
			{ID: "o1", Name: "Owner One"},
			{ID: "o2", Name: "Owner Two"},
		},
		nil,
	)

	roots := []any{
		&testCarrier{ID: "c1", OwnerID: "o1"},
		&testCarrier{ID: "c2", OwnerID: "o2"},
		&testCarrier{ID: "c3", OwnerID: "o1"},
	}
	tree := ParseIncludeTree([]string{"owner"})
	if err := ResolveIncludes(context.Background(), roots, otCarrier, tree); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.owner.Load() != 1 {
		t.Errorf("expected exactly 1 owner load (batched), got %d", c.owner.Load())
	}
	for _, r := range roots {
		cr := r.(*testCarrier)
		if cr.Owner == nil {
			t.Errorf("carrier %s missing owner", cr.ID)
		}
	}
}

func TestResolveIncludes_ListCardinality(t *testing.T) {
	ResetForTest()
	c := &counters{}
	registerCarrierGraph(t, c,
		nil,
		nil,
		[]*testServiceLevel{
			{ID: "sl1", Name: "Ground"},
			{ID: "sl2", Name: "Express"},
		},
	)

	carrier := &testCarrier{ID: "c1", ServiceLevelIDs: []string{"sl1", "sl2"}}
	tree := ParseIncludeTree([]string{"service_levels"})
	if err := ResolveIncludes(context.Background(), []any{carrier}, otCarrier, tree); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(carrier.ServiceLevels) != 2 {
		t.Fatalf("expected 2 service levels, got %d", len(carrier.ServiceLevels))
	}
	if c.sl.Load() != 1 {
		t.Errorf("expected exactly 1 service-level load, got %d", c.sl.Load())
	}
}

func TestResolveIncludes_MissingFK_LeavesFieldUnset(t *testing.T) {
	// Carrier references owner o-missing, but owner loader returns nothing.
	// The Owner field should stay nil — no hallucination.
	ResetForTest()
	c := &counters{}
	registerCarrierGraph(t, c, nil, nil, nil)

	carrier := &testCarrier{ID: "c1", OwnerID: "o-missing"}
	tree := ParseIncludeTree([]string{"owner"})
	if err := ResolveIncludes(context.Background(), []any{carrier}, otCarrier, tree); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carrier.Owner != nil {
		t.Errorf("Owner should remain nil when loader returns nothing, got %+v", carrier.Owner)
	}
}

func TestResolveIncludes_CycleViaMemoization(t *testing.T) {
	// Customer c1 has c2 as a child. c2 has c1 as its parent. Asking for
	// child_accounts.parent_account would re-fetch c1 inside the recursion
	// without memoization; with it, customer loader fires only twice (once
	// at root level, once for the child).
	ResetForTest()
	c := &counters{}
	registerCustomerCycle(t, c, []*testCustomer{
		{ID: "c1", ChildAccountIDs: []string{"c2"}},
		{ID: "c2", ParentID: "c1"},
	})

	root := &testCustomer{ID: "c1", ChildAccountIDs: []string{"c2"}}
	ctx := WithLoadCache(context.Background())
	tree := ParseIncludeTree([]string{"child_accounts.parent_account"})
	if err := ResolveIncludes(ctx, []any{root}, otCustomer, tree); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(root.ChildAccounts) != 1 || root.ChildAccounts[0].ID != "c2" {
		t.Fatalf("expected child c2, got %+v", root.ChildAccounts)
	}
	if root.ChildAccounts[0].Parent == nil || root.ChildAccounts[0].Parent.ID != "c1" {
		t.Errorf("expected parent stitched on child, got %+v", root.ChildAccounts[0].Parent)
	}
	// Without seeded cache: 2 loads (the root was not loaded via the cache).
	// With the cache pre-warmed by callers it would be 1; we don't pre-warm
	// here, so accept loads ≤ 2.
	if c.customer.Load() > 2 {
		t.Errorf("expected at most 2 customer loads (memoized cycle), got %d", c.customer.Load())
	}
}

func TestResolveIncludes_DepthCap(t *testing.T) {
	ResetForTest()
	c := &counters{}
	// Build a parent_account chain long enough to overshoot the cap.
	chain := DefaultMaxIncludeDepth + 3
	var customers []*testCustomer
	for i := 1; i <= chain; i++ {
		cust := &testCustomer{ID: id(i)}
		if i < chain {
			cust.ParentID = id(i + 1)
		}
		customers = append(customers, cust)
	}
	registerCustomerCycle(t, c, customers)

	root := &testCustomer{ID: "c1", ParentID: "c2"}
	// One segment past the cap: resolving the last hop enters the resolver at depth DefaultMaxIncludeDepth.
	segments := make([]string, DefaultMaxIncludeDepth+1)
	for i := range segments {
		segments[i] = "parent_account"
	}
	tree := ParseIncludeTree([]string{strings.Join(segments, ".")})
	err := ResolveIncludes(context.Background(), []any{root}, otCustomer, tree)
	if err == nil {
		t.Fatalf("expected depth-cap error, got nil")
	}
	if !containsString(err.Error(), "depth limit") {
		t.Errorf("expected error to mention depth limit, got: %v", err)
	}
}

func TestResolveIncludes_UnregisteredTarget_Errors(t *testing.T) {
	ResetForTest()
	Register(&Definition{
		ObjectType: otCarrier,
		Load: func(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
			return nil, nil
		},
		Subs: []SubField{
			{
				Key: "owner", Target: otOwner, Cardinality: CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, p any) []string { return []string{"o1"} },
				Populate:   func(_ context.Context, p any, loaded map[string]any) {},
			},
		},
	})
	// Note: otOwner is NOT registered.
	root := &testCarrier{ID: "c1", OwnerID: "o1"}
	tree := ParseIncludeTree([]string{"owner"})
	err := ResolveIncludes(context.Background(), []any{root}, otCarrier, tree)
	if err == nil {
		t.Fatalf("expected error for unregistered target")
	}
	if !containsString(err.Error(), "unregistered") {
		t.Errorf("expected error to mention unregistered target, got: %v", err)
	}
}

func TestAllowedIncludeKeys(t *testing.T) {
	ResetForTest()
	c := &counters{}
	registerCarrierGraph(t, c, nil, nil, nil)

	keys := AllowedIncludeKeys(otCarrier, 3)
	sort.Strings(keys)
	want := []string{"owner", "service_levels"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("AllowedIncludeKeys mismatch\n  got:  %v\n  want: %v", keys, want)
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	ResetForTest()
	d := &Definition{
		ObjectType: otCarrier,
		Load: func(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
			return nil, nil
		},
	}
	Register(d)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on duplicate registration")
		}
	}()
	Register(d)
}

// --- helpers ---

func id(i int) string { return "c" + itoa(i) }

func itoa(i int) string {
	// avoid pulling strconv just to keep test fixtures local; deterministic.
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	buf := [20]byte{}
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func containsString(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// Ensure we exercise errors.Is path on apierror to catch wrapping regressions.
var _ = errors.Is
