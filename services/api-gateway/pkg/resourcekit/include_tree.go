package resourcekit

import "strings"

// IncludeNode represents one segment in a parsed include tree. The tree
// supports dot-paths: `?include[]=freight_preferences.carrier&include[]=child_accounts`
// parses to:
//
//	root
//	├── freight_preferences
//	│   └── carrier
//	└── child_accounts
//
// A node without Children is a leaf — the client asked for the parent but
// not any of its sub-resources.
type IncludeNode struct {
	Children map[string]*IncludeNode
}

// NewIncludeTree returns an empty tree.
func NewIncludeTree() *IncludeNode {
	return &IncludeNode{Children: map[string]*IncludeNode{}}
}

// ParseIncludeTree builds a tree from a flat list of dot-paths (e.g.
// ["freight_preferences.carrier", "child_accounts"]). Order of paths does not
// matter; longer paths and their prefixes collapse correctly.
//
// Empty path segments are tolerated (a stray "..") and ignored.
func ParseIncludeTree(keys []string) *IncludeNode {
	root := NewIncludeTree()
	for _, key := range keys {
		if key == "" {
			continue
		}
		root.Add(key)
	}
	return root
}

// Add inserts a single dot-path into the tree.
func (n *IncludeNode) Add(key string) {
	cur := n
	for seg := range strings.SplitSeq(key, ".") {
		if seg == "" {
			continue
		}
		next, ok := cur.Children[seg]
		if !ok {
			next = NewIncludeTree()
			cur.Children[seg] = next
		}
		cur = next
	}
}

// Child returns the sub-tree under `key` (which may be dot-separated for
// multi-segment lookups like "freight_preferences.carrier"), or nil if no
// such path exists.
func (n *IncludeNode) Child(key string) *IncludeNode {
	cur := n
	for seg := range strings.SplitSeq(key, ".") {
		if cur == nil {
			return nil
		}
		cur = cur.Children[seg]
	}
	return cur
}

// Has returns true if `key` was requested at any level under this node.
func (n *IncludeNode) Has(key string) bool {
	return n.Child(key) != nil
}

// HasChildren reports whether this node has any descendants. Safe on nil
// receivers — returns false.
func (n *IncludeNode) HasChildren() bool {
	return n != nil && len(n.Children) > 0
}

// Flatten returns every dot-path reachable from this node, sorted for
// deterministic output. Empty tree yields nil.
func (n *IncludeNode) Flatten() []string {
	if !n.HasChildren() {
		return nil
	}
	var out []string
	n.flattenInto("", &out)
	return out
}

func (n *IncludeNode) flattenInto(prefix string, out *[]string) {
	for seg, child := range n.Children {
		path := seg
		if prefix != "" {
			path = prefix + "." + seg
		}
		*out = append(*out, path)
		child.flattenInto(path, out)
	}
}
