// Package querytag carries SQLCommenter tags (https://google.github.io/sqlcommenter/spec/) on a context, so the SQL a request or job runs says which service, RPC, route, consumer or job it came from. PlanetScale Insights groups load by these tags.
//
// Values must be low-cardinality: a method, route pattern or job name, never an id. Insights stops recording a tag's values once it has seen too many distinct ones.
package querytag

import (
	"context"
	"net/url"
	"sort"
	"strings"
)

// Tag keys set across services.
const (
	App        = "app"
	GRPCMethod = "grpc_method"
	Route      = "route"
	Consumer   = "consumer"
	Job        = "job"
)

type ctxKey struct{}

// With returns ctx carrying the given key/value pairs on top of any it already carries. An empty value removes the key, so a job started from a request can drop the request's route.
func With(ctx context.Context, keyValues ...string) context.Context {
	if len(keyValues) < 2 {
		return ctx
	}
	parent, _ := ctx.Value(ctxKey{}).(map[string]string)
	tags := make(map[string]string, len(parent)+len(keyValues)/2)
	for k, v := range parent {
		tags[k] = v
	}
	for i := 0; i+1 < len(keyValues); i += 2 {
		if keyValues[i+1] == "" {
			delete(tags, keyValues[i])
			continue
		}
		tags[keyValues[i]] = keyValues[i+1]
	}
	return context.WithValue(ctx, ctxKey{}, tags)
}

// From returns the tags ctx carries. The map is shared; callers must not modify it.
func From(ctx context.Context) map[string]string {
	tags, _ := ctx.Value(ctxKey{}).(map[string]string)
	return tags
}

// Comment formats static tags, overridden by ctx's, as one SQLCommenter comment; empty when there are none.
func Comment(ctx context.Context, static map[string]string) string {
	dynamic := From(ctx)
	if len(static) == 0 && len(dynamic) == 0 {
		return ""
	}
	merged := make(map[string]string, len(static)+len(dynamic))
	for k, v := range static {
		merged[k] = v
	}
	for k, v := range dynamic {
		merged[k] = v
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("/*")
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(escape(k))
		b.WriteString("='")
		b.WriteString(escape(merged[k]))
		b.WriteByte('\'')
	}
	b.WriteString("*/")
	return b.String()
}

// escape URL-encodes per the spec. QueryEscape encodes the quote, '*' and '/', so a value can end neither its string nor the comment.
func escape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// Append adds comment to the end of query, before any terminating semicolon, where PlanetScale reads it.
func Append(query, comment string) string {
	if comment == "" {
		return query
	}
	trimmed := strings.TrimRight(query, " \t\r\n;")
	return trimmed + " " + comment + query[len(trimmed):]
}
