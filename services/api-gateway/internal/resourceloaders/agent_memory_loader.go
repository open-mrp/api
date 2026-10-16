package resourceloaders

import (
	"context"
	"slices"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var agentMemoryLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.agent_memory")

// noteClient is the CoreNoteService client. Agent memories are notes, read in their preview.9 shape. Set at startup.
var noteClient pb.CoreNoteServiceClient

// SetNoteClient is called once at startup with the CoreNoteService client.
func SetNoteClient(c pb.CoreNoteServiceClient) {
	noteClient = c
}

// memoryNoteTypes are the note types a memory can be. Comments were never memories, so the memory routes don't show them.
var memoryNoteTypes = []string{string(constants.NoteTypePreference), string(constants.NoteTypeFact), string(constants.NoteTypeInstruction)}

// MemoryNoteTypes is memoryNoteTypes for a list request.
func MemoryNoteTypes() []string { return slices.Clone(memoryNoteTypes) }

// LoadAgentMemories reads notes by ID in the preview.9 agent memory shape, leaving out comments.
func LoadAgentMemories(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, agentMemoryLoaderTracer, "loader.agent_memories.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetNotesByIDsResponse, error) {
			return noteClient.BatchGetNotesByIDs(ctx, &pb.BatchGetNotesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Notes))
	entities := make([]*apiresource.Entity, 0, len(resp.Notes))
	for _, n := range resp.Notes {
		mem := AgentMemoryFromNote(n)
		if mem == nil {
			continue
		}
		out[n.Id] = mem
		entities = append(entities, mem.Entity)
	}
	HydrateCustomerEntities(ctx, entities)
	return out, nil
}

// AgentMemoryFromNote is a note as preview.9 showed a memory: type as category, body as content, the record as entity
// and importance as a number (low 0.2, normal 0.5, high 0.8). Nil for a comment, which no memory could be.
func AgentMemoryFromNote(n *pb.NoteInfo) *apiresource.AgentMemory {
	if !slices.Contains(memoryNoteTypes, n.Type) {
		return nil
	}
	mem := &apiresource.AgentMemory{
		ID:         n.Id,
		Object:     constants.ObjectTypeAgentMemory,
		Category:   constants.AgentMemoryCategory(n.Type),
		Content:    n.Body,
		Importance: MemoryImportance(constants.NoteImportance(n.Importance)),
		CreatedAt:  n.CreatedAt.AsTime(),
		UpdatedAt:  n.UpdatedAt.AsTime(),
	}
	if n.Record != nil {
		mem.Entity = apiresource.NewEntity(n.Record.Id, constants.ObjectType(n.Record.Type), nil, nil)
	}
	if n.ExpiresAt != nil {
		t := n.ExpiresAt.AsTime()
		mem.ExpiresAt = &t
	}
	return mem
}

// MemoryImportance is a note importance as a memory's 0–1 number.
func MemoryImportance(i constants.NoteImportance) float64 {
	switch i {
	case constants.NoteImportanceLow:
		return 0.2
	case constants.NoteImportanceHigh:
		return 0.8
	default:
		return 0.5
	}
}

// NoteImportanceOfMemory is a memory's 0–1 importance as a note importance.
func NoteImportanceOfMemory(v float64) constants.NoteImportance {
	return constants.NoteImportanceFromScore(v)
}
