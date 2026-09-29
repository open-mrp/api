package main

import "testing"

func TestReplicaURLFallsBackToPrimary(t *testing.T) {
	env := map[string]string{"DB_URL": "primary"}
	getenv := func(k string) string { return env[k] }

	if got := (&config{}).withDefaults(getenv).DBReplicaURL; got != "primary" {
		t.Fatalf("DBReplicaURL without DB_REPLICA_URL = %q, want the primary URL", got)
	}
	env["DB_REPLICA_URL"] = "replica"
	if got := (&config{}).withDefaults(getenv).DBReplicaURL; got != "replica" {
		t.Fatalf("DBReplicaURL = %q, want %q", got, "replica")
	}
}
