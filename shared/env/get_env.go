package env

import (
	"strings"
)

// GetEnv gets the environment variable with the given key and returns it trimmed. If the environment variable is not set, it returns the empty string.
func GetEnv(key string, getenv func(string) string) string {
	return strings.TrimSpace(getenv(key))
}
