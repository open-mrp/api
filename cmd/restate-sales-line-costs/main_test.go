package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun_RejectsIncompleteFlags(t *testing.T) {
	base := []string{"--account", "ac_cost", "--item", "itm_boxed", "--stale-cost", "8.40", "--since", "2026-01-01"}
	without := func(flag string) []string {
		var out []string
		for i := 0; i < len(base); i += 2 {
			if base[i] != flag {
				out = append(out, base[i], base[i+1])
			}
		}
		return out
	}

	cases := map[string]struct {
		args []string
		want string
	}{
		"no account":         {without("--account"), "--account"},
		"no item":            {without("--item"), "--item"},
		"no stale cost":      {without("--stale-cost"), "--stale-cost"},
		"no since":           {without("--since"), "--since"},
		"zero stale cost":    {append(without("--stale-cost"), "--stale-cost", "0"), "positive"},
		"bad since":          {append(without("--since"), "--since", "09/03/2026"), "YYYY-MM-DD"},
		"until before since": {append(base, "--until", "2025-12-01"), "before --until"},
		"no dsn":             {base, "DATABASE_DSN"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			getenv := func(string) string { return "" }
			err := Run(context.Background(), append([]string{"restate-sales-line-costs"}, tc.args...), getenv, strings.NewReader(""), &stdout, &stderr)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestParseFlags_SplitsItemsAndDefaultsUntil(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := parseFlags([]string{"x", "--account", "ac_cost", "--item", "itm_a, itm_b,", "--stale-cost", "8.401234567890123456789", "--since", "2026-09-03"}, &stderr)
	require.NoError(t, err)
	require.Equal(t, []string{"itm_a", "itm_b"}, opts.itemIDs)
	require.Equal(t, "8.401234567890123456789", opts.staleCost.String())
	require.True(t, opts.since.Before(opts.until))
	require.False(t, opts.apply)
}
