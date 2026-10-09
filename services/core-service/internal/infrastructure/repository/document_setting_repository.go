package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var documentSettingRepoTracer = tracing.GetTracer("core-service.document_setting_repository")

type documentSettingRepoImpl struct {
	queries *sqlc.Queries
}

func NewDocumentSettingRepo(queries *sqlc.Queries) domain.DocumentSettingRepo {
	return &documentSettingRepoImpl{queries: queries}
}

func documentSettingFromRow(row sqlc.DocumentSetting) *domain.DocumentSetting {
	return &domain.DocumentSetting{
		ID:             row.ID,
		AccountID:      row.AccountID,
		DocumentType:   constants.DocumentType(row.DocumentType),
		ProcessOwner:   db.StringFromNullString(row.ProcessOwner),
		DocumentNumber: db.StringFromNullString(row.DocumentNumber),
		Revision:       db.StringFromNullString(row.Revision),
		FooterText:     db.StringFromNullString(row.FooterText),
		CreatedAt:      &row.CreatedAt,
		UpdatedAt:      &row.UpdatedAt,
	}
}

func (r *documentSettingRepoImpl) List(ctx context.Context, accountID string) ([]*domain.DocumentSetting, *apierror.APIError) {
	ctx, span := documentSettingRepoTracer.Start(ctx, "repository.document_setting.list")
	defer span.End()

	rows, err := r.queries.ListDocumentSettings(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.DocumentSetting, len(rows))
	for i, row := range rows {
		out[i] = documentSettingFromRow(row)
	}
	return out, nil
}

func (r *documentSettingRepoImpl) Get(ctx context.Context, accountID string, documentType constants.DocumentType) (*domain.DocumentSetting, *apierror.APIError) {
	ctx, span := documentSettingRepoTracer.Start(ctx, "repository.document_setting.get")
	defer span.End()

	row, err := r.queries.GetDocumentSetting(ctx, sqlc.GetDocumentSettingParams{
		AccountID:    accountID,
		DocumentType: string(documentType),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return documentSettingFromRow(row), nil
}

func (r *documentSettingRepoImpl) Upsert(ctx context.Context, settingID string, setting domain.DocumentSetting) *apierror.APIError {
	ctx, span := documentSettingRepoTracer.Start(ctx, "repository.document_setting.upsert")
	defer span.End()

	err := r.queries.UpsertDocumentSetting(ctx, sqlc.UpsertDocumentSettingParams{
		ID:             settingID,
		AccountID:      setting.AccountID,
		DocumentType:   string(setting.DocumentType),
		ProcessOwner:   db.NullStringPtr(setting.ProcessOwner),
		DocumentNumber: db.NullStringPtr(setting.DocumentNumber),
		Revision:       db.NullStringPtr(setting.Revision),
		FooterText:     db.NullStringPtr(setting.FooterText),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
