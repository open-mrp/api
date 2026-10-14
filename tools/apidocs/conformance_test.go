package main

import (
	"testing"

	"github.com/open-mrp/apikit/conformance"
)

// TestConformance holds every endpoint to the forge.1 rules it already meets. The forge.1 sweep adds the rest as
// each lands: ActionsArePost, DeleteReturnsStub, ListsPageWithListRequest, LongRunningReturnsAsyncJob and
// PublicDocstrings, until this asserts conformance.Rules.
func TestConformance(t *testing.T) {
	t.Parallel()
	conformance.Assert(t, openAPIEndpointGroups(),
		conformance.RouteShape,
		conformance.PathParamsRequired,
		conformance.OptionalFieldsOmitZero,
		conformance.ResponsesNeverOmit,
	)
}
