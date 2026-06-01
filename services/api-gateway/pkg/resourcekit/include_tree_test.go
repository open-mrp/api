package resourcekit

import (
	"reflect"
	"sort"
	"testing"
)

func TestParseIncludeTree_FlatPaths(t *testing.T) {
	tree := ParseIncludeTree([]string{"owner", "service_levels"})
	if !tree.Has("owner") {
		t.Errorf("expected owner present")
	}
	if !tree.Has("service_levels") {
		t.Errorf("expected service_levels present")
	}
	if tree.Has("nope") {
		t.Errorf("did not expect nope")
	}
}

func TestParseIncludeTree_NestedPaths(t *testing.T) {
	tree := ParseIncludeTree([]string{"freight_preferences.carrier", "child_accounts.parent_account"})
	if !tree.Has("freight_preferences") {
		t.Errorf("expected freight_preferences as ancestor")
	}
	if !tree.Has("freight_preferences.carrier") {
		t.Errorf("expected freight_preferences.carrier")
	}
	if tree.Child("freight_preferences.carrier").HasChildren() {
		t.Errorf("leaf node should have no children")
	}
	if !tree.Child("freight_preferences").HasChildren() {
		t.Errorf("parent of leaf should have children")
	}
}

func TestParseIncludeTree_PrefixAndLeafCollapse(t *testing.T) {
	// Asking for both the parent path and the child path should produce one
	// tree with the child path under the parent — not duplicate the parent.
	tree := ParseIncludeTree([]string{"freight_preferences", "freight_preferences.carrier"})
	fp := tree.Child("freight_preferences")
	if fp == nil {
		t.Fatalf("expected freight_preferences node")
	}
	if len(fp.Children) != 1 {
		t.Errorf("expected freight_preferences to have exactly 1 child, got %d", len(fp.Children))
	}
	if !fp.Has("carrier") {
		t.Errorf("expected carrier under freight_preferences")
	}
}

func TestParseIncludeTree_IgnoresEmpty(t *testing.T) {
	tree := ParseIncludeTree([]string{"", "owner", "..", "service_levels..."})
	if !tree.Has("owner") || !tree.Has("service_levels") {
		t.Errorf("expected owner and service_levels present, got tree=%v", tree.Flatten())
	}
}

func TestFlatten_DeterministicSortable(t *testing.T) {
	tree := ParseIncludeTree([]string{"freight_preferences.carrier", "owner", "child_accounts"})
	got := tree.Flatten()
	sort.Strings(got)
	want := []string{"child_accounts", "freight_preferences", "freight_preferences.carrier", "owner"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Flatten()\n  got:  %v\n  want: %v", got, want)
	}
}

func TestHasChildren_NilSafe(t *testing.T) {
	var n *IncludeNode
	if n.HasChildren() {
		t.Errorf("nil node should report HasChildren=false")
	}
	empty := NewIncludeTree()
	if empty.HasChildren() {
		t.Errorf("empty tree should report HasChildren=false")
	}
}
