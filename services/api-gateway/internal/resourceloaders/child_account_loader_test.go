package resourceloaders

import (
	"testing"
	"time"

	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The child's account carries the account's own timestamps, not the relation's.
func TestChildAccountFromProtoCarriesTheAccountsTimestamps(t *testing.T) {
	relationAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	accountCreated := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	accountUpdated := time.Date(2025, 5, 6, 0, 0, 0, 0, time.UTC)

	got := ChildAccountFromProto(&pb.ChildAccountProto{
		RelationId:       "acre_child",
		AccountId:        "ac_child",
		AccountName:      "Store 12",
		CreatedAt:        timestamppb.New(relationAt),
		UpdatedAt:        timestamppb.New(relationAt),
		AccountCreatedAt: timestamppb.New(accountCreated),
		AccountUpdatedAt: timestamppb.New(accountUpdated),
	})

	if !got.Account.CreatedAt.Equal(accountCreated) || !got.Account.UpdatedAt.Equal(accountUpdated) {
		t.Fatalf("account timestamps = %v / %v, want %v / %v", got.Account.CreatedAt, got.Account.UpdatedAt, accountCreated, accountUpdated)
	}
	if !got.CreatedAt.Equal(relationAt) {
		t.Fatalf("relation created_at = %v, want %v", got.CreatedAt, relationAt)
	}
}
