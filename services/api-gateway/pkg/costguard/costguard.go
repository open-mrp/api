// Package costguard keeps the seller's cost and margin figures from callers who may not see them.
//
// A response field carrying cost data is tagged `sensitive:"cost"`. Redact nulls every such field reachable from a response, through nested resources, lists, and resolved includes, unless the caller is an internal actor holding costs:read (or an admin). Customer and supplier portal actors never see them.
package costguard

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/open-mrp/api/shared/appctx"
)

const (
	// TagKey is the struct tag key that marks a field as data a caller may be denied.
	TagKey = "sensitive"
	// TagCost marks a field carrying the seller's cost or margin data.
	TagCost = "cost"
)

// IsCostField reports whether sf is tagged as cost data.
func IsCostField(sf reflect.StructField) bool {
	return sf.Tag.Get(TagKey) == TagCost
}

// Redactor is implemented by a resource whose cost data cannot be marked field by field, such as an audited change whose field name says what it holds. Redact calls it on every such value it reaches.
type Redactor interface {
	RedactCosts()
}

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

// Visible reports whether the caller in ctx may see cost data.
func Visible(ctx context.Context) bool {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	return ok && identity.CanReadCosts()
}

// Redact nulls every cost field reachable from v unless the caller in ctx may see cost data, and returns the value to serialize. Pointers, slices and maps are redacted in place; a bare struct value comes back as a redacted copy.
func Redact(ctx context.Context, v any) any {
	if Visible(ctx) {
		return v
	}
	return Strip(v)
}

// Strip nulls every cost field reachable from v regardless of the caller. See Redact for what is changed in place.
func Strip(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	p := planFor(rv.Type())
	if p == nil {
		return v
	}
	if rv.Kind() == reflect.Struct || rv.Kind() == reflect.Array {
		cp := reflect.New(rv.Type()).Elem()
		cp.Set(rv)
		p.apply(cp)
		return cp.Interface()
	}
	p.apply(rv)
	return v
}

// HasCostFields reports whether a value of type t can carry a cost field.
func HasCostFields(t reflect.Type) bool {
	return planFor(t) != nil
}

// plan is what to visit in a value of one type: the cost fields to clear, and the children that can lead to more. A nil plan means the type can never carry cost data, so the walk skips it without reflecting over it.
type plan struct {
	kind     reflect.Kind
	clear    []int
	fields   []fieldPlan
	elem     *plan
	dynamic  bool
	redactor bool
}

type fieldPlan struct {
	index int
	plan  *plan
}

func (p *plan) apply(v reflect.Value) {
	switch p.kind {
	case reflect.Pointer:
		if !v.IsNil() {
			p.elem.apply(v.Elem())
		}
	case reflect.Interface:
		if !v.IsNil() {
			applyDynamic(v)
		}
	case reflect.Struct:
		for _, i := range p.clear {
			v.Field(i).SetZero()
		}
		for _, f := range p.fields {
			f.plan.apply(v.Field(f.index))
		}
		if p.redactor {
			v.Addr().Interface().(Redactor).RedactCosts()
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			p.elem.apply(v.Index(i))
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			val := iter.Value()
			switch val.Kind() {
			case reflect.Pointer, reflect.Slice, reflect.Map:
				p.elem.apply(val)
			default:
				cp := reflect.New(val.Type()).Elem()
				cp.Set(val)
				p.elem.apply(cp)
				v.SetMapIndex(iter.Key(), cp)
			}
		}
	}
}

// applyDynamic redacts the concrete value held by an interface, writing a copy back when that value is not addressable.
func applyDynamic(v reflect.Value) {
	inner := v.Elem()
	p := planFor(inner.Type())
	if p == nil {
		return
	}
	switch inner.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map:
		p.apply(inner)
	default:
		if !v.CanSet() {
			return
		}
		cp := reflect.New(inner.Type()).Elem()
		cp.Set(inner)
		p.apply(cp)
		v.Set(cp)
	}
}

var (
	plans   sync.Map // reflect.Type -> *plan (nil when the type carries no cost data)
	buildMu sync.Mutex
)

func planFor(t reflect.Type) *plan {
	if p, ok := plans.Load(t); ok {
		return p.(*plan)
	}
	buildMu.Lock()
	defer buildMu.Unlock()
	if p, ok := plans.Load(t); ok {
		return p.(*plan)
	}
	b := &builder{nodes: map[reflect.Type]*node{}}
	root := b.node(t)
	b.finish()
	return root.plan
}

// node is a type under construction. Types refer to each other in cycles (an account's child accounts are accounts), so whether a type leads to a cost field is settled for the whole graph at once in finish, not while descending.
type node struct {
	t        reflect.Type
	kind     reflect.Kind
	clear    []int
	fields   []fieldNode
	elem     *node
	dynamic  bool
	redactor bool
	live     bool
	plan     *plan
	cached   bool
}

type fieldNode struct {
	index int
	node  *node
}

type builder struct {
	nodes map[reflect.Type]*node
}

func (b *builder) node(t reflect.Type) *node {
	if n, ok := b.nodes[t]; ok {
		return n
	}
	n := &node{t: t, kind: t.Kind()}
	b.nodes[t] = n
	if p, ok := plans.Load(t); ok {
		n.cached = true
		n.plan = p.(*plan)
		n.live = n.plan != nil
		return n
	}
	switch n.kind {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		n.elem = b.node(t.Elem())
	case reflect.Interface:
		n.dynamic = true
	case reflect.Struct:
		n.redactor = reflect.PointerTo(t).Implements(redactorType)
		for i := range t.NumField() {
			sf := t.Field(i)
			if !sf.IsExported() || sf.Tag.Get("json") == "-" {
				continue
			}
			if IsCostField(sf) {
				n.clear = append(n.clear, i)
				continue
			}
			if canNest(sf.Type) {
				n.fields = append(n.fields, fieldNode{index: i, node: b.node(sf.Type)})
			}
		}
	}
	return n
}

// canNest reports whether a value of type t can hold another value, so a cost field could sit below it.
func canNest(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map, reflect.Interface, reflect.Struct:
		return true
	default:
		return false
	}
}

var redactorType = reflect.TypeFor[Redactor]()

func (b *builder) finish() {
	for _, n := range b.nodes {
		if !n.cached && (len(n.clear) > 0 || n.dynamic || n.redactor) {
			n.live = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, n := range b.nodes {
			if n.live || n.cached {
				continue
			}
			if (n.elem != nil && n.elem.live) || anyLive(n.fields) {
				n.live = true
				changed = true
			}
		}
	}
	for _, n := range b.nodes {
		if !n.cached && n.live {
			n.plan = &plan{kind: n.kind, clear: n.clear, dynamic: n.dynamic, redactor: n.redactor}
		}
	}
	for _, n := range b.nodes {
		if n.cached || !n.live {
			continue
		}
		if n.elem != nil {
			n.plan.elem = n.elem.plan
		}
		for _, f := range n.fields {
			if f.node.live {
				n.plan.fields = append(n.plan.fields, fieldPlan{index: f.index, plan: f.node.plan})
			}
		}
	}
	for t, n := range b.nodes {
		if !n.cached {
			plans.Store(t, n.plan)
		}
	}
}

func anyLive(fields []fieldNode) bool {
	for _, f := range fields {
		if f.node.live {
			return true
		}
	}
	return false
}
