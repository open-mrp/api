package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type noteGRPCHandler struct {
	pb.UnimplementedCoreNoteServiceServer

	noteSvc domain.NoteSvc
}

func RegisterNoteService(server *grpc.Server, noteSvc domain.NoteSvc) {
	pb.RegisterCoreNoteServiceServer(server, &noteGRPCHandler{noteSvc: noteSvc})
}

func noteToProto(n *domain.Note) *pb.NoteInfo {
	info := &pb.NoteInfo{
		Id:         n.ID,
		Body:       n.Body,
		Type:       string(n.Type),
		Importance: string(n.Importance),
		Actor:      n.Actor.ToProto(),
		CreatedAt:  timestamppb.New(n.CreatedAt),
		UpdatedAt:  timestamppb.New(n.UpdatedAt),
	}
	if n.RecordType != nil && n.RecordID != nil {
		info.Record = &pb.NoteRecordRef{Type: string(*n.RecordType), Id: *n.RecordID}
	}
	if n.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*n.ExpiresAt)
	}
	if n.EditedAt != nil {
		info.EditedAt = timestamppb.New(*n.EditedAt)
	}
	return info
}

func notesToProto(notes []*domain.Note) []*pb.NoteInfo {
	out := make([]*pb.NoteInfo, len(notes))
	for i, n := range notes {
		out[i] = noteToProto(n)
	}
	return out
}

func noteRecordFromProto(r *pb.NoteRecordRef) *domain.NoteRecord {
	if r == nil {
		return nil
	}
	return &domain.NoteRecord{Type: constants.RecordType(r.Type), ID: r.Id}
}

func noteTypeFromProto(v *string) *constants.NoteType {
	if v == nil {
		return nil
	}
	t := constants.NoteType(*v)
	return &t
}

func noteImportanceFromProto(v *string) *constants.NoteImportance {
	if v == nil {
		return nil
	}
	i := constants.NoteImportance(*v)
	return &i
}

func convertStrings[T ~string](values []string) []T {
	if len(values) == 0 {
		return nil
	}
	out := make([]T, len(values))
	for i, v := range values {
		out[i] = T(v)
	}
	return out
}

func (h *noteGRPCHandler) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	result, apiErr := h.noteSvc.ListNotes(ctx, domain.ListNotesParams{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		RecordIDs:   req.RecordIds,
		RecordTypes: convertStrings[constants.RecordType](req.RecordTypes),
		Types:       convertStrings[constants.NoteType](req.Types),
		ActorTypes:  convertStrings[constants.ActorType](req.ActorTypes),
		Statuses:    convertStrings[constants.NoteStatus](req.Statuses),
		Query:       req.Query,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ListNotesResponse{
		Notes: notesToProto(result.Notes),
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *noteGRPCHandler) GetNote(ctx context.Context, req *pb.GetNoteRequest) (*pb.GetNoteResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	n, apiErr := h.noteSvc.GetNote(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetNoteResponse{Note: noteToProto(n)}, nil
}

func (h *noteGRPCHandler) BatchGetNotesByIDs(ctx context.Context, req *pb.BatchGetNotesByIDsRequest) (*pb.BatchGetNotesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	notes, apiErr := h.noteSvc.BatchGetNotesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BatchGetNotesByIDsResponse{Notes: notesToProto(notes)}, nil
}

func (h *noteGRPCHandler) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.CreateNoteResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	params := domain.CreateNoteParams{
		Record:     noteRecordFromProto(req.Record),
		Body:       req.Body,
		Type:       noteTypeFromProto(req.Type),
		Importance: noteImportanceFromProto(req.Importance),
	}
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		params.ExpiresAt = &t
	}
	n, apiErr := h.noteSvc.CreateNote(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.CreateNoteResponse{Note: noteToProto(n)}, nil
}

func (h *noteGRPCHandler) UpdateNote(ctx context.Context, req *pb.UpdateNoteRequest) (*pb.UpdateNoteResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	params := domain.UpdateNoteParams{
		NoteID:      req.Id,
		Body:        req.Body,
		Type:        noteTypeFromProto(req.Type),
		Importance:  noteImportanceFromProto(req.Importance),
		Record:      noteRecordFromProto(req.Record),
		ClearRecord: req.ClearRecord,
	}
	if p := req.ExpiresAt; p != nil {
		if p.Clear || p.Value == nil {
			params.ClearExpiresAt = true
		} else {
			t := p.Value.AsTime()
			params.ExpiresAt = &t
		}
	}
	n, apiErr := h.noteSvc.UpdateNote(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.UpdateNoteResponse{Note: noteToProto(n)}, nil
}

func (h *noteGRPCHandler) DeleteNote(ctx context.Context, req *pb.DeleteNoteRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.noteSvc.DeleteNote(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}

func (h *noteGRPCHandler) ImportNotes(ctx context.Context, req *pb.ImportNotesRequest) (*pb.ImportNotesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	notes := make([]domain.ImportNote, len(req.Notes))
	for i, in := range req.Notes {
		a, err := actor.FromProto(in.Actor)
		if err != nil || a == nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("note import: invalid actor on " + in.Id))
		}
		n := domain.ImportNote{
			ID:         in.Id,
			AccountID:  in.AccountId,
			Record:     noteRecordFromProto(in.Record),
			Body:       in.Body,
			Type:       constants.NoteType(in.Type),
			Importance: constants.NoteImportance(in.Importance),
			Actor:      *a,
			CreatedAt:  in.CreatedAt.AsTime(),
			UpdatedAt:  in.UpdatedAt.AsTime(),
		}
		if in.ExpiresAt != nil {
			t := in.ExpiresAt.AsTime()
			n.ExpiresAt = &t
		}
		notes[i] = n
	}
	inserted, apiErr := h.noteSvc.ImportNotes(ctx, notes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ImportNotesResponse{Inserted: inserted}, nil
}

func (h *noteGRPCHandler) RecallNotes(ctx context.Context, req *pb.RecallNotesRequest) (*pb.RecallNotesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	notes, apiErr := h.noteSvc.RecallNotes(ctx, domain.RecallNotesParams{
		AccountID: req.AccountId,
		RecordIDs: req.RecordIds,
		Limit:     req.Limit,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.RecallNotesResponse{Notes: notesToProto(notes)}, nil
}
