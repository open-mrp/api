package agentmemoryep

import (
	"context"
	"fmt"
	"time"

	noteep "github.com/open-mrp/api/services/api-gateway/endpoints/notes"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Agent memories are notes. These routes answer only 1.0.forge-preview.9 callers, reading and writing notes in the
// memory shape; on 1.0.forge they 404.
const memoriesRemoved = "Agent memories are notes: use the Notes endpoints (`/v1/core/notes`)."

type AgentMemorySvc interface {
	ListMemories(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError)
	GetMemory(ctx context.Context, req *RetrieveMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError)
	CreateMemory(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError)
	UpdateMemory(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError)
	DeleteMemory(ctx context.Context, req *DeleteMemoryRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AgentMemorySvcConfig struct {
	// NoteClient (required) is the core-service notes gRPC client.
	NoteClient pb.CoreNoteServiceClient
}

type agentMemorySvcImpl struct {
	noteClient pb.CoreNoteServiceClient
}

var memorySvcTracer = tracing.GetTracer("api-gateway.endpoints.agent_memories.service")

func (c *AgentMemorySvcConfig) validate() error {
	if c.NoteClient == nil {
		return fmt.Errorf("agent memory endpoint service: note client is required")
	}
	return nil
}

func NewAgentMemorySvc(config *AgentMemorySvcConfig) AgentMemorySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &agentMemorySvcImpl{noteClient: config.NoteClient}
}

func requirePreview9(ctx context.Context) *apierror.APIError {
	if v, ok := appctx.GetAPIVersionFromContext(ctx); ok && v.Equal(apiversion.V1_0_Forge_Preview9) {
		return nil
	}
	return apierror.NewNotFoundError("This route was removed in 1.0.forge. " + memoriesRemoved)
}

// memoryRecord is the note record a memory's entity names: nil for none or for the account itself. Products, materials
// and parts are items.
func memoryRecord(entityType, entityID string) (*pb.NoteRecordRef, *apierror.APIError) {
	if entityType == "" || entityID == "" || entityType == "account" {
		return nil, nil
	}
	t := constants.RecordType(entityType)
	switch entityType {
	case "product", "material", "part":
		t = constants.RecordTypeItem
	}
	if !t.IsValid() {
		return nil, apierror.NewFieldError("entity_type", apierror.CodeInvalidFormat, "Unknown entity type.")
	}
	return &pb.NoteRecordRef{Type: string(t), Id: entityID}, nil
}

func parseMemoryTime(param, v string) (*timestamppb.Timestamp, *apierror.APIError) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Must be an ISO 8601 timestamp.")
	}
	return timestamppb.New(t), nil
}

func (m *agentMemorySvcImpl) ListMemories(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError) {
	if apiErr := requirePreview9(ctx); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.ListNotesRequest{
		Limit:    req.Limit,
		Cursor:   req.Cursor,
		Query:    req.Query,
		Types:    resourceloaders.MemoryNoteTypes(),
		Statuses: []string{string(constants.NoteStatusActive)},
	}
	if req.Category != nil {
		pbReq.Types = []string{string(*req.Category)}
	}
	if req.EntityType != nil {
		if r, apiErr := memoryRecord(*req.EntityType, "-"); apiErr != nil {
			// preview.9 filtered by any string, so a type no record has matches nothing.
			return apiresource.NewList([]apiresource.AgentMemory{}, apiresource.PageInfo{}), nil
		} else if r != nil {
			pbReq.RecordTypes = []string{r.Type}
		}
	}
	resp, apiErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListNotesResponse, error) {
			return m.noteClient.ListNotes(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.AgentMemory, 0, len(resp.Notes))
	entities := make([]*apiresource.Entity, 0, len(resp.Notes))
	for _, n := range resp.Notes {
		if mem := resourceloaders.AgentMemoryFromNote(n); mem != nil {
			items = append(items, *mem)
		}
	}
	for i := range items {
		entities = append(entities, items[i].Entity)
	}
	resourceloaders.HydrateCustomerEntities(ctx, entities)
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *agentMemorySvcImpl) GetMemory(ctx context.Context, req *RetrieveMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	if apiErr := requirePreview9(ctx); apiErr != nil {
		return nil, apiErr
	}
	return loadMemoryByID(ctx, req.ID)
}

func (m *agentMemorySvcImpl) CreateMemory(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	if apiErr := requirePreview9(ctx); apiErr != nil {
		return nil, apiErr
	}
	noteType := string(req.Category)
	// A memory created without an importance was stored at 0, the first left out.
	importance := string(constants.NoteImportanceLow)
	if v, ok := req.Importance.Value(); ok {
		importance = string(resourceloaders.NoteImportanceOfMemory(v))
	}
	pbReq := &pb.CreateNoteRequest{Body: req.Content, Type: &noteType, Importance: &importance}
	entityType, _ := req.EntityType.Value()
	entityID, _ := req.EntityID.Value()
	record, apiErr := memoryRecord(entityType, entityID)
	if apiErr != nil {
		return nil, apiErr
	}
	if record != nil {
		if apiErr := noteep.RequireRecord(ctx, "entity_id", constants.RecordType(record.Type), record.Id); apiErr != nil {
			return nil, apiErr
		}
		pbReq.Record = record
	}
	if v, ok := req.ExpiresAt.Value(); ok {
		ts, apiErr := parseMemoryTime("expires_at", v)
		if apiErr != nil {
			return nil, apiErr
		}
		pbReq.ExpiresAt = ts
	}
	resp, apiErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateNoteResponse, error) {
			return m.noteClient.CreateNote(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadMemoryByID(ctx, resp.Note.Id)
}

func (m *agentMemorySvcImpl) UpdateMemory(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	if apiErr := requirePreview9(ctx); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.UpdateNoteRequest{Id: req.ID}
	if v, ok := req.Category.Value(); ok {
		pbReq.Type = v.StringPtr()
	}
	if v, ok := req.Content.Value(); ok {
		pbReq.Body = &v
	}
	if v, ok := req.Importance.Value(); ok {
		i := string(resourceloaders.NoteImportanceOfMemory(v))
		pbReq.Importance = &i
	}
	// Either entity field sent as null unscopes the memory.
	if req.EntityType.IsClear() || req.EntityID.IsClear() {
		pbReq.ClearRecord = true
	} else {
		entityType, hasType := req.EntityType.Value()
		entityID, hasID := req.EntityID.Value()
		if hasType || hasID {
			if !hasType || !hasID {
				current, apiErr := loadMemoryByID(ctx, req.ID)
				if apiErr != nil {
					return nil, apiErr
				}
				if current.Entity != nil {
					if !hasType {
						entityType = string(current.Entity.Type)
					}
					if !hasID {
						entityID = current.Entity.ID
					}
				}
			}
			record, apiErr := memoryRecord(entityType, entityID)
			if apiErr != nil {
				return nil, apiErr
			}
			if record == nil {
				pbReq.ClearRecord = true
			} else {
				if apiErr := noteep.RequireRecord(ctx, "entity_id", constants.RecordType(record.Type), record.Id); apiErr != nil {
					return nil, apiErr
				}
				pbReq.Record = record
			}
		}
	}
	if req.ExpiresAt.IsClear() {
		pbReq.ExpiresAt = &pb.TimestampPatch{Clear: true}
	} else if v, ok := req.ExpiresAt.Value(); ok {
		ts, apiErr := parseMemoryTime("expires_at", v)
		if apiErr != nil {
			return nil, apiErr
		}
		pbReq.ExpiresAt = &pb.TimestampPatch{Value: ts}
	}
	resp, apiErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateNoteResponse, error) {
			return m.noteClient.UpdateNote(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadMemoryByID(ctx, resp.Note.Id)
}

func (m *agentMemorySvcImpl) DeleteMemory(ctx context.Context, req *DeleteMemoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	if apiErr := requirePreview9(ctx); apiErr != nil {
		return nil, apiErr
	}
	_, apiErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.noteClient.DeleteNote(ctx, &pb.DeleteNoteRequest{Id: req.ID}, opts...)
		})
	if apiErr != nil {
		// A memory deleted before reads as already deleted, as it did when memories lived in the agent service.
		if apiErr.Code == apierror.CodeResourceGone || apiErr.Code == apierror.CodeResourceNotFound {
			return &apiresource.EmptyResource{}, nil
		}
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadMemoryByID is one note in the memory shape. A comment reads as not found: it was never a memory.
func loadMemoryByID(ctx context.Context, id string) (*apiresource.AgentMemory, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadAgentMemories(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewNotFoundError("Agent memory not found.")
	}
	return v.(*apiresource.AgentMemory), nil
}
