package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithoutTokenCounts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"thinking step", `{"input_tokens":1200,"output_tokens":80}`, `{}`},
		{"compaction step", `{"input_tokens_before":90000,"tokens_freed":40000,"compaction_type":"prune"}`, `{"compaction_type":"prune"}`},
		{"no token counts", `{"tool_name":"list_items"}`, `{"tool_name":"list_items"}`},
		{"not an object", `[1,2]`, `[1,2]`},
		{"empty", ``, ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := string(withoutTokenCounts([]byte(tc.in)))
			if tc.want == "" || tc.want == `[1,2]` {
				assert.Equal(t, tc.want, got)
				return
			}
			assert.JSONEq(t, tc.want, got)
		})
	}
}
