package querytag

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommentSortsAndEncodes(t *testing.T) {
	ctx := With(context.Background(), GRPCMethod, "/core.CoreService/AnalyzeSalesSummary", Route, "GET /v1/items/{id}")
	require.Equal(t,
		`/*app='core-service',grpc_method='%2Fcore.CoreService%2FAnalyzeSalesSummary',route='GET%20%2Fv1%2Fitems%2F%7Bid%7D'*/`,
		Comment(ctx, map[string]string{App: "core-service"}))
}

func TestCommentEscapesQuotes(t *testing.T) {
	require.Equal(t, `/*job='it%27s'*/`, Comment(With(context.Background(), Job, "it's"), nil))
}

func TestContextOverridesStaticAndEmptyRemoves(t *testing.T) {
	ctx := With(context.Background(), App, "override", Route, "GET /x")
	ctx = With(ctx, Route, "", Job, "sweep")
	require.Equal(t, `/*app='override',job='sweep'*/`, Comment(ctx, map[string]string{App: "core-service"}))
}

func TestWithDoesNotLeakIntoParent(t *testing.T) {
	parent := With(context.Background(), Job, "a")
	_ = With(parent, Job, "b")
	require.Equal(t, "a", From(parent)[Job])
}

func TestNoTagsNoComment(t *testing.T) {
	require.Equal(t, "", Comment(context.Background(), nil))
	require.Equal(t, "SELECT 1", Append("SELECT 1", ""))
}

func TestAppendGoesBeforeTheSemicolon(t *testing.T) {
	require.Equal(t, "SELECT 1 /*a='b'*/;\n", Append("SELECT 1;\n", "/*a='b'*/"))
	require.Equal(t, "-- name: X :one\nSELECT 1 /*a='b'*/", Append("-- name: X :one\nSELECT 1", "/*a='b'*/"))
	require.Equal(t, "SELECT /*+ MAX_EXECUTION_TIME(5) */ 1 /*a='b'*/\n", Append("SELECT /*+ MAX_EXECUTION_TIME(5) */ 1\n", "/*a='b'*/"))
}
