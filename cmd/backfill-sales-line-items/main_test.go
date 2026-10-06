package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun_RejectsIncompleteFlags(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"no account": {[]string{}, "--account"},
		"zero batch": {[]string{"--account", "ac_cost", "--batch", "0"}, "--batch"},
		"huge batch": {[]string{"--account", "ac_cost", "--batch", "10000"}, "--batch"},
		"no dsn":     {[]string{"--account", "ac_cost"}, "DATABASE_DSN"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			getenv := func(string) string { return "" }
			err := Run(context.Background(), append([]string{"backfill-sales-line-items"}, tc.args...), getenv, strings.NewReader(""), &stdout, &stderr)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := parseFlags([]string{"x", "--account", "ac_cost"}, &stderr)
	require.NoError(t, err)
	require.Equal(t, 500, opts.batch)
	require.False(t, opts.apply)
}
