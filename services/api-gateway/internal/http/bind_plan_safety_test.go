package httptransport

import (
	"net/url"
	"testing"
)

type bindCycleA struct {
	B *bindCycleB
}

type bindCycleB struct {
	A *bindCycleA
	Q string `query:"q"`
}

type bindCycleReq struct {
	A bindCycleA
}

type bindPruneInner struct {
	X int
}

type bindPruneReq struct {
	Name  string `query:"name"`
	Inner bindPruneInner
}

type bindDepthL33 struct {
	Q string `query:"deep"`
}

type bindDepthL32 struct {
	N bindDepthL33
}

type bindDepthL31 struct {
	N bindDepthL32
}

type bindDepthL30 struct {
	N bindDepthL31
}

type bindDepthL29 struct {
	N bindDepthL30
}

type bindDepthL28 struct {
	N bindDepthL29
}

type bindDepthL27 struct {
	N bindDepthL28
}

type bindDepthL26 struct {
	N bindDepthL27
}

type bindDepthL25 struct {
	N bindDepthL26
}

type bindDepthL24 struct {
	N bindDepthL25
}

type bindDepthL23 struct {
	N bindDepthL24
}

type bindDepthL22 struct {
	N bindDepthL23
}

type bindDepthL21 struct {
	N bindDepthL22
}

type bindDepthL20 struct {
	N bindDepthL21
}

type bindDepthL19 struct {
	N bindDepthL20
}

type bindDepthL18 struct {
	N bindDepthL19
}

type bindDepthL17 struct {
	N bindDepthL18
}

type bindDepthL16 struct {
	N bindDepthL17
}

type bindDepthL15 struct {
	N bindDepthL16
}

type bindDepthL14 struct {
	N bindDepthL15
}

type bindDepthL13 struct {
	N bindDepthL14
}

type bindDepthL12 struct {
	N bindDepthL13
}

type bindDepthL11 struct {
	N bindDepthL12
}

type bindDepthL10 struct {
	N bindDepthL11
}

type bindDepthL9 struct {
	N bindDepthL10
}

type bindDepthL8 struct {
	N bindDepthL9
}

type bindDepthL7 struct {
	N bindDepthL8
}

type bindDepthL6 struct {
	N bindDepthL7
}

type bindDepthL5 struct {
	N bindDepthL6
}

type bindDepthL4 struct {
	N bindDepthL5
}

type bindDepthL3 struct {
	N bindDepthL4
}

type bindDepthL2 struct {
	N bindDepthL3
}

type bindDepthL1 struct {
	N bindDepthL2
}

type bindDepthL0 struct {
	N bindDepthL1
}

type bindDepthReq struct {
	Root bindDepthL0
	Name string `query:"name"`
}

func TestBuildBindPlan_cycleSafe(t *testing.T) {
	t.Parallel()

	p, err := planFor(&bindCycleReq{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.allowedQuery["q"]; !ok {
		t.Fatalf("expected q in allowedQuery, got %#v", p.allowedQuery)
	}

	u := mustParseURL(t, "/x?q=ok")
	dst := &bindCycleReq{A: bindCycleA{B: &bindCycleB{}}}
	if err := BindFromQuery(u, dst); err != nil {
		t.Fatal(err)
	}
	if dst.A.B.Q != "ok" {
		t.Fatalf("Q=%q", dst.A.B.Q)
	}
}

func TestBuildBindPlan_prunesUntaggedSubtree(t *testing.T) {
	t.Parallel()

	p, err := planFor(&bindPruneReq{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.allowedQuery["name"]; !ok {
		t.Fatalf("expected name in allowedQuery, got %#v", p.allowedQuery)
	}
	if len(p.allowedQuery) != 1 {
		t.Fatalf("pruned subtree should leave only top-level bind keys, got %#v", p.allowedQuery)
	}

	u := mustParseURL(t, "/x?name=alice&ignored=1")
	dst := &bindPruneReq{}
	if err := BindFromQuery(u, dst); err != nil {
		t.Fatal(err)
	}
	if dst.Name != "alice" {
		t.Fatalf("Name=%q", dst.Name)
	}
	if apiErr := RejectUnknownQueryParams(u, dst, false); apiErr == nil {
		t.Fatal("expected unknown query parameter error for ignored")
	}
}

func TestBuildBindPlan_maxDepthExcludesDeepQuery(t *testing.T) {
	t.Parallel()

	p, err := planFor(&bindDepthReq{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.allowedQuery["name"]; !ok {
		t.Fatalf("expected name, got %#v", p.allowedQuery)
	}
	if _, ok := p.allowedQuery["deep"]; ok {
		t.Fatal("query key beyond maxBindPlanStructDepth should not be registered")
	}

	u, err := url.Parse("/x?name=n&deep=should-reject")
	if err != nil {
		t.Fatal(err)
	}
	if apiErr := RejectUnknownQueryParams(u, &bindDepthReq{}, false); apiErr == nil {
		t.Fatal("expected unknown parameter for deep")
	}
}
