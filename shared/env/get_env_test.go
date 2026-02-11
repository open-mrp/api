package env

import (
	"testing"
)

func TestGetEnv(t *testing.T) {
	getenv := func(key string) string {
		m := map[string]string{"FOO": "  bar  ", "EMPTY": "", "MISSING": ""}
		return m[key]
	}
	if got := GetEnv("FOO", getenv); got != "bar" {
		t.Errorf("GetEnv(FOO) = %q, want %q", got, "bar")
	}
	if got := GetEnv("EMPTY", getenv); got != "" {
		t.Errorf("GetEnv(EMPTY) = %q, want %q", got, "")
	}
	if got := GetEnv("MISSING", getenv); got != "" {
		t.Errorf("GetEnv(MISSING) = %q, want %q", got, "")
	}
}
