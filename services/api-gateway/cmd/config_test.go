package main

import (
	"strings"
	"testing"
)

// envFunc builds a getenv-style closure backed by a map. Mirrors how main()
// would normally hand os.Getenv to withDefaults.
func envFunc(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func TestConfigTrustedProxyHopsDefaultsToZero(t *testing.T) {
	t.Parallel()
	c := (&config{}).withDefaults(envFunc(map[string]string{
		"DB_URL": "mysql://x",
	}))
	if c.TrustedProxyHops != 0 {
		t.Fatalf("default TrustedProxyHops = %d; want 0 (XFF must NOT be trusted unless explicitly opted in)", c.TrustedProxyHops)
	}
}

func TestConfigTrustedProxyHopsParsedFromEnv(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want int
	}{
		{"1", 1},
		{"2", 2},
		{"  3  ", 3}, // env.GetEnv trims whitespace
	}
	for _, tc := range cases {
		t.Run("hops="+tc.raw, func(t *testing.T) {
			t.Parallel()
			c := (&config{}).withDefaults(envFunc(map[string]string{
				"DB_URL":             "mysql://x",
				"TRUSTED_PROXY_HOPS": tc.raw,
			}))
			if c.TrustedProxyHops != tc.want {
				t.Fatalf("TrustedProxyHops = %d; want %d (raw=%q)", c.TrustedProxyHops, tc.want, tc.raw)
			}
		})
	}
}

func TestConfigTrustedProxyHopsInvalidValueFallsBackToZero(t *testing.T) {
	t.Parallel()
	// Non-numeric env value should not crash; it falls back to the safe
	// default of 0 (XFF ignored).
	c := (&config{}).withDefaults(envFunc(map[string]string{
		"DB_URL":             "mysql://x",
		"TRUSTED_PROXY_HOPS": "not-a-number",
	}))
	if c.TrustedProxyHops != 0 {
		t.Fatalf("invalid TRUSTED_PROXY_HOPS should fall back to 0; got %d", c.TrustedProxyHops)
	}
}

func TestConfigValidateRejectsNegativeTrustedProxyHops(t *testing.T) {
	t.Parallel()
	c := (&config{}).withDefaults(envFunc(map[string]string{
		"DB_URL": "mysql://x",
	}))
	c.TrustedProxyHops = -1
	err := c.validate()
	if err == nil {
		t.Fatal("validate should reject negative TrustedProxyHops")
	}
	if !strings.Contains(err.Error(), "trusted proxy hops") {
		t.Fatalf("error should mention trusted proxy hops; got %v", err)
	}
}

func TestConfigValidatePassesWithDefaultTrustedProxyHops(t *testing.T) {
	t.Parallel()
	c := (&config{}).withDefaults(envFunc(map[string]string{
		"DB_URL": "mysql://x",
	}))
	if err := c.validate(); err != nil {
		t.Fatalf("validate returned unexpected error: %v", err)
	}
}
