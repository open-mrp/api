package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestSiteOf(t *testing.T) {
	t.Parallel()

	p := func(s string) *string { return &s }
	nodes := map[string]domain.LocationTreeNode{
		"main":   {ID: "main", TypeCode: "site"},
		"bldg":   {ID: "bldg", ParentID: p("main"), TypeCode: "building"},
		"bin":    {ID: "bin", ParentID: p("bldg"), TypeCode: "bin"},
		"plant2": {ID: "plant2", TypeCode: "site"},
		"rack":   {ID: "rack", ParentID: p("plant2"), TypeCode: "rack"},
		"loose":  {ID: "loose", TypeCode: "building"},
		"cyc1":   {ID: "cyc1", ParentID: p("cyc2"), TypeCode: "bin"},
		"cyc2":   {ID: "cyc2", ParentID: p("cyc1"), TypeCode: "bin"},
	}
	for loc, want := range map[string]string{
		"main":    "main",
		"bin":     "main",
		"rack":    "plant2",
		"loose":   "main",
		"cyc1":    "main",
		"unknown": "main",
	} {
		assert.Equal(t, want, domain.SiteOf(nodes, loc, "main"), loc)
	}
	assert.Equal(t, "", domain.SiteOf(nodes, "loose", ""))
}
