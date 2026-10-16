package resourceloaders

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func TestAgentMemoryFromNote(t *testing.T) {
	t.Parallel()
	at := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	mem := AgentMemoryFromNote(&pb.NoteInfo{
		Id: "nt_1", Body: "Prefers express.", Type: "preference", Importance: "high",
		Record: &pb.NoteRecordRef{Type: "customer", Id: "ac_c"}, CreatedAt: at, UpdatedAt: at,
	})
	require.NotNil(t, mem)
	assert.Equal(t, constants.ObjectTypeAgentMemory, mem.Object)
	assert.Equal(t, constants.AgentMemoryCategoryPreference, mem.Category)
	assert.Equal(t, "Prefers express.", mem.Content)
	assert.InDelta(t, 0.8, mem.Importance, 1e-9)
	require.NotNil(t, mem.Entity)
	assert.Equal(t, constants.ObjectTypeCustomer, mem.Entity.Type)
	assert.Equal(t, "ac_c", mem.Entity.ID)

	assert.Nil(t, AgentMemoryFromNote(&pb.NoteInfo{Id: "nt_2", Type: "comment", CreatedAt: at, UpdatedAt: at}), "a comment was never a memory")
}

func TestMemoryImportanceRoundTrips(t *testing.T) {
	t.Parallel()
	for _, i := range []constants.NoteImportance{constants.NoteImportanceLow, constants.NoteImportanceNormal, constants.NoteImportanceHigh} {
		assert.Equal(t, i, NoteImportanceOfMemory(MemoryImportance(i)))
	}
}
