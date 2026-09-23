package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func fakeLookup(byRepository map[string][]string) tagLookup {
	return func(_ context.Context, repository string) (map[string]struct{}, error) {
		tags := make(map[string]struct{})
		for _, tag := range byRepository[repository] {
			tags[tag] = struct{}{}
		}
		return tags, nil
	}
}

func TestResolveImageTags_FallsBackToNewestOlderImage(t *testing.T) {
	t.Parallel()

	// Rolling back to v2.7.4, a release that built core-service alone.
	candidates := []string{"v2.7.4", "v2.7.3", "v2.7.2"}
	lookup := fakeLookup(map[string][]string{
		"augno/core-service": {"v2.7.4", "v2.7.3", "v2.7.2"},
		"augno/api-gateway":  {"v2.7.2"},
	})

	resolved, unresolved, err := resolveImageTags(
		context.Background(), "augno", []string{"core-service", "api-gateway"}, candidates, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unresolved) != 0 {
		t.Fatalf("unexpected unresolved services: %v", unresolved)
	}

	want := map[string]string{"core-service": "v2.7.4", "api-gateway": "v2.7.2"}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolved mismatch: got %v want %v", resolved, want)
	}
}

func TestResolveImageTags_IgnoresTagsNewerThanTheTarget(t *testing.T) {
	t.Parallel()

	// v2.8.0 is absent from candidates because the caller is rolling back to v2.7.4.
	candidates := []string{"v2.7.4", "v2.7.3"}
	lookup := fakeLookup(map[string][]string{"augno/api-gateway": {"v2.8.0", "v2.7.3"}})

	resolved, unresolved, err := resolveImageTags(
		context.Background(), "augno", []string{"api-gateway"}, candidates, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unresolved) != 0 {
		t.Fatalf("unexpected unresolved services: %v", unresolved)
	}
	if resolved["api-gateway"] != "v2.7.3" {
		t.Fatalf("api-gateway resolved to %q, want v2.7.3", resolved["api-gateway"])
	}
}

func TestResolveImageTags_ReportsServiceWithNoImage(t *testing.T) {
	t.Parallel()

	candidates := []string{"v2.7.4"}
	lookup := fakeLookup(map[string][]string{"augno/core-service": {"v2.7.4"}})

	_, unresolved, err := resolveImageTags(
		context.Background(), "augno", []string{"core-service", "billing-service"}, candidates, lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(unresolved, []string{"billing-service"}) {
		t.Fatalf("unresolved mismatch: got %v want [billing-service]", unresolved)
	}
}

func TestResolveImageTags_PropagatesLookupError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	lookup := func(_ context.Context, _ string) (map[string]struct{}, error) {
		return nil, wantErr
	}

	_, _, err := resolveImageTags(context.Background(), "augno", []string{"core-service"}, []string{"v2.7.4"}, lookup)
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
}

func TestGroupByTag_OrdersGroupsNewestTagFirst(t *testing.T) {
	t.Parallel()

	candidates := []string{"v2.8.0", "v2.7.4", "v2.7.3"}
	serviceNames := []string{"api-gateway", "auth-service", "core-service"}
	resolved := map[string]string{
		"api-gateway":  "v2.7.3",
		"auth-service": "v2.8.0",
		"core-service": "v2.7.3",
	}

	got := groupByTag(candidates, serviceNames, resolved)
	want := []group{
		{Tag: "v2.8.0", Services: []string{"auth-service"}},
		{Tag: "v2.7.3", Services: []string{"api-gateway", "core-service"}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("groups mismatch: got %v want %v", got, want)
	}
}

func TestReleaseTagPattern_RejectsNonReleaseTags(t *testing.T) {
	t.Parallel()

	for _, tag := range []string{"v2.7.4", "v2.7.4-rc1", "v10.0.0"} {
		if !releaseTagPattern.MatchString(tag) {
			t.Fatalf("expected %q to be a release tag", tag)
		}
	}
	for _, tag := range []string{"v2.7", "sandbox-v2.7.4", "v2.7.4/rollback"} {
		if releaseTagPattern.MatchString(tag) {
			t.Fatalf("expected %q not to be a release tag", tag)
		}
	}
}
