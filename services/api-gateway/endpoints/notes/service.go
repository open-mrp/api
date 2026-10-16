package noteep

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const notesRoute = "/v1/core/notes"

type NoteSvc interface {
	ListNotes(ctx context.Context, req *ListNotesRequest) (*apiresource.List[apiresource.Note], *apierror.APIError)
	GetNote(ctx context.Context, req *RetrieveNoteRequest) (*apiresource.Note, *apierror.APIError)
	CreateNote(ctx context.Context, req *CreateNoteRequest) (*apiresource.Note, *apierror.APIError)
	UpdateNote(ctx context.Context, req *UpdateNoteRequest) (*apiresource.Note, *apierror.APIError)
	DeleteNote(ctx context.Context, req *DeleteNoteRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type NoteSvcConfig struct {
	// CoreClient (required) is the core-service notes gRPC client.
	CoreClient pb.CoreNoteServiceClient
}

func (c *NoteSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("note endpoint service: core client is required")
	}
	return nil
}

type noteSvcImpl struct {
	coreClient pb.CoreNoteServiceClient
}

var noteSvcTracer = tracing.GetTracer("api-gateway.endpoints.notes.service")

func NewNoteSvc(config *NoteSvcConfig) NoteSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &noteSvcImpl{coreClient: config.CoreClient}
}

func (s *noteSvcImpl) ListNotes(ctx context.Context, req *ListNotesRequest) (*apiresource.List[apiresource.Note], *apierror.APIError) {
	pbReq := &pb.ListNotesRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		RecordIds:   req.RecordIDs,
		RecordTypes: enumStrings(req.RecordTypes),
		Types:       enumStrings(req.Types),
		ActorTypes:  enumStrings(req.ActorTypes),
		Statuses:    enumStrings(req.Statuses),
		Query:       req.Query,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, noteSvcTracer, "service.notes.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListNotesResponse, error) {
			return s.coreClient.ListNotes(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	notes := Present(ctx, resp.Notes)
	items := make([]apiresource.Note, len(notes))
	for i, n := range notes {
		items[i] = *n
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *noteSvcImpl) GetNote(ctx context.Context, req *RetrieveNoteRequest) (*apiresource.Note, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, noteSvcTracer, "service.notes.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetNoteResponse, error) {
			return s.coreClient.GetNote(ctx, &pb.GetNoteRequest{Id: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return Present(ctx, []*pb.NoteInfo{resp.Note})[0], nil
}

func (s *noteSvcImpl) CreateNote(ctx context.Context, req *CreateNoteRequest) (*apiresource.Note, *apierror.APIError) {
	pbReq := &pb.CreateNoteRequest{
		Body:       req.Body,
		Type:       req.Type.Ptr().StringPtr(),
		Importance: req.Importance.Ptr().StringPtr(),
	}
	if r, ok := req.Record.Value(); ok {
		if apiErr := RequireRecord(ctx, "record.id", r.Type, r.ID); apiErr != nil {
			return nil, apiErr
		}
		pbReq.Record = &pb.NoteRecordRef{Type: string(r.Type), Id: r.ID}
	}
	if v, ok := req.ExpiresAt.Value(); ok {
		pbReq.ExpiresAt = timestamppb.New(v)
	}
	resp, apiErr := grpcutil.CallRPC(ctx, noteSvcTracer, "service.notes.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateNoteResponse, error) {
			return s.coreClient.CreateNote(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return Present(ctx, []*pb.NoteInfo{resp.Note})[0], nil
}

func (s *noteSvcImpl) UpdateNote(ctx context.Context, req *UpdateNoteRequest) (*apiresource.Note, *apierror.APIError) {
	pbReq := &pb.UpdateNoteRequest{
		Id:         req.ID,
		Body:       req.Body.Ptr(),
		Type:       req.Type.Ptr().StringPtr(),
		Importance: req.Importance.Ptr().StringPtr(),
	}
	if req.ExpiresAt.IsClear() {
		pbReq.ExpiresAt = &pb.TimestampPatch{Clear: true}
	} else if v, ok := req.ExpiresAt.Value(); ok {
		pbReq.ExpiresAt = &pb.TimestampPatch{Value: timestamppb.New(v)}
	}
	resp, apiErr := grpcutil.CallRPC(ctx, noteSvcTracer, "service.notes.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateNoteResponse, error) {
			return s.coreClient.UpdateNote(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return Present(ctx, []*pb.NoteInfo{resp.Note})[0], nil
}

func (s *noteSvcImpl) DeleteNote(ctx context.Context, req *DeleteNoteRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, noteSvcTracer, "service.notes.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.coreClient.DeleteNote(ctx, &pb.DeleteNoteRequest{Id: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func enumStrings[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// RequireRecord rejects a note on a record that doesn't exist, for every record type that has a stub loader. param names
// the field in the error.
func RequireRecord(ctx context.Context, param string, t constants.RecordType, id string) *apierror.APIError {
	if !t.IsValid() || !recordstub.Registered(t) {
		return nil
	}
	loaded, apiErr := recordstub.Load(include.WithIncludeReads(ctx), []string{recordstub.Key(t, id)})
	if apiErr != nil {
		return apiErr
	}
	if recordstub.One(loaded, t, id) == nil {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, fmt.Sprintf("No %s with this ID exists.", t))
	}
	return nil
}

// Present is the response shape of notes core returned. Each note's record is its stub, filled from the record's own
// loader where its type has one; a record of another type, or one that no longer exists, keeps its type and ID only.
// Actor names are filled in one batch.
func Present(ctx context.Context, infos []*pb.NoteInfo) []*apiresource.Note {
	out := make([]*apiresource.Note, len(infos))
	var keys []string
	var actors []*apiresource.ActorRef
	now := time.Now()
	for i, info := range infos {
		n := &apiresource.Note{
			ID:         info.Id,
			Object:     constants.ObjectTypeNote,
			Body:       info.Body,
			Type:       constants.NoteType(info.Type),
			Importance: constants.NoteImportance(info.Importance),
			Status:     constants.NoteStatusActive,
			Actor:      resourceloaders.ActorRefFromProto(info.Actor),
			CreatedAt:  info.CreatedAt.AsTime(),
			UpdatedAt:  info.UpdatedAt.AsTime(),
		}
		if info.ExpiresAt != nil {
			t := info.ExpiresAt.AsTime()
			n.ExpiresAt = &t
			if !t.After(now) {
				n.Status = constants.NoteStatusExpired
			}
		}
		if info.EditedAt != nil {
			t := info.EditedAt.AsTime()
			n.EditedAt = &t
		}
		if r := info.Record; r != nil {
			rt := constants.RecordType(r.Type)
			stub := apiresource.NewRecord(r.Id, rt)
			stub.Metadata = map[string]string{}
			n.Record = stub
			if recordstub.Registered(rt) {
				keys = append(keys, recordstub.Key(rt, r.Id))
			}
		}
		actors = append(actors, n.Actor)
		out[i] = n
	}

	if len(keys) > 0 {
		if loaded, apiErr := recordstub.Load(include.WithIncludeReads(ctx), keys); apiErr == nil {
			for _, n := range out {
				if n.Record == nil {
					continue
				}
				if s := recordstub.One(loaded, n.Record.Type, n.Record.ID); s != nil {
					if s.Metadata == nil {
						s.Metadata = map[string]string{}
					}
					n.Record = s
				}
			}
		}
	}
	resourceloaders.ResolveActorRefs(ctx, actors)
	return out
}
