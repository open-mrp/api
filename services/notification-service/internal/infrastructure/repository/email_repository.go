package repository

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var emailRepoTracer = tracing.GetTracer("notification-service.email_repository")

// marshalStringSlice encodes a string slice as JSON for a nullable JSON column; nil/empty → NULL.
func marshalStringSlice(vals []string) db.NullableRawMessage {
	if len(vals) == 0 {
		return nil
	}
	b, err := json.Marshal(vals)
	if err != nil {
		return nil
	}
	return db.NullableRawMessage(b)
}

// unmarshalStringSlice decodes a nullable JSON array column back to a string slice; NULL → nil.
func unmarshalStringSlice(raw db.NullableRawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// ── email_domain ──

type emailDomainRepoImpl struct {
	db *sqlc.Queries
}

func NewEmailDomainRepo(db *sqlc.Queries) domain.EmailDomainRepo {
	return &emailDomainRepoImpl{db: db}
}

func (r *emailDomainRepoImpl) Create(ctx context.Context, id, accountID string, input *domain.CreateEmailDomainInput) *apierror.APIError {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.create")
	defer span.End()
	err := r.db.CreateEmailDomain(ctx, sqlc.CreateEmailDomainParams{
		ID:         id,
		AccountID:  accountID,
		Domain:     input.Domain,
		DkimTokens: marshalStringSlice(input.DkimTokens),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *emailDomainRepoImpl) GetByID(ctx context.Context, id, accountID string) (*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.get_by_id")
	defer span.End()
	row, err := r.db.GetEmailDomainByID(ctx, sqlc.GetEmailDomainByIDParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return emailDomainFromRow(row), nil
}

func (r *emailDomainRepoImpl) GetByDomain(ctx context.Context, dom string) (*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.get_by_domain")
	defer span.End()
	row, err := r.db.GetEmailDomainByDomain(ctx, dom)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return emailDomainFromRow(row), nil
}

func (r *emailDomainRepoImpl) ListByAccount(ctx context.Context, accountID string) ([]*domain.EmailDomain, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.list_by_account")
	defer span.End()
	rows, err := r.db.ListEmailDomainsByAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.EmailDomain, 0, len(rows))
	for _, row := range rows {
		out = append(out, emailDomainFromRow(row))
	}
	return out, nil
}

func (r *emailDomainRepoImpl) MarkVerified(ctx context.Context, id, accountID string) *apierror.APIError {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.mark_verified")
	defer span.End()
	err := r.db.MarkEmailDomainVerified(ctx, sqlc.MarkEmailDomainVerifiedParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *emailDomainRepoImpl) UpdateStatus(ctx context.Context, id, accountID, status string, dkimTokens []string) *apierror.APIError {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.update_status")
	defer span.End()
	err := r.db.UpdateEmailDomainStatus(ctx, sqlc.UpdateEmailDomainStatusParams{
		Status:     status,
		DkimTokens: marshalStringSlice(dkimTokens),
		ID:         id,
		AccountID:  accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *emailDomainRepoImpl) Delete(ctx context.Context, id, accountID string) (bool, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.delete")
	defer span.End()
	n, err := r.db.DeleteEmailDomain(ctx, sqlc.DeleteEmailDomainParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return n > 0, nil
}

func emailDomainFromRow(row sqlc.EmailDomain) *domain.EmailDomain {
	return &domain.EmailDomain{
		ID:             row.ID,
		AccountID:      row.AccountID,
		Domain:         row.Domain,
		Status:         row.Status,
		DkimTokens:     unmarshalStringSlice(row.DkimTokens),
		MailFromDomain: db.StringFromNullString(row.MailFromDomain),
		VerifiedAt:     db.TimeFromNullTime(row.VerifiedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (r *emailDomainRepoImpl) SetMailFromDomain(ctx context.Context, id, accountID, mailFromDomain string) *apierror.APIError {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_domain.set_mail_from_domain")
	defer span.End()
	err := r.db.SetEmailDomainMailFrom(ctx, sqlc.SetEmailDomainMailFromParams{
		MailFromDomain: db.NullString(mailFromDomain),
		ID:             id,
		AccountID:      accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// ── email_sender ──

type accountEmailSenderRepoImpl struct {
	db *sqlc.Queries
}

func NewAccountEmailSenderRepo(db *sqlc.Queries) domain.AccountEmailSenderRepo {
	return &accountEmailSenderRepoImpl{db: db}
}

func (r *accountEmailSenderRepoImpl) Upsert(ctx context.Context, id, accountID string, input *domain.UpsertAccountEmailSenderInput) *apierror.APIError {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_sender.upsert")
	defer span.End()
	err := r.db.UpsertEmailSender(ctx, sqlc.UpsertEmailSenderParams{
		ID:            id,
		AccountID:     accountID,
		EmailDomainID: input.EmailDomainID,
		LocalPart:     input.LocalPart,
		FromName:      db.NullStringPtr(input.FromName),
		ReplyTo:       db.NullStringPtr(input.ReplyTo),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// GetByAccount runs on every merchant-facing send, and most accounts have no sender configured. No rows is therefore the ordinary case and maps to (nil, nil) rather than a 404 the send path would have to unwrap.
func (r *accountEmailSenderRepoImpl) GetByAccount(ctx context.Context, accountID string) (*domain.AccountEmailSender, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_sender.get_by_account")
	defer span.End()
	row, err := r.db.GetEmailSenderByAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.AccountEmailSender{
		ID:             row.EmailSender.ID,
		AccountID:      row.EmailSender.AccountID,
		EmailDomainID:  row.EmailSender.EmailDomainID,
		LocalPart:      row.EmailSender.LocalPart,
		FromName:       db.StringFromNullString(row.EmailSender.FromName),
		ReplyTo:        db.StringFromNullString(row.EmailSender.ReplyTo),
		Domain:         row.Domain,
		DomainStatus:   row.Status,
		MailFromDomain: db.StringFromNullString(row.MailFromDomain),
		CreatedAt:      row.EmailSender.CreatedAt,
		UpdatedAt:      row.EmailSender.UpdatedAt,
	}, nil
}

func (r *accountEmailSenderRepoImpl) Delete(ctx context.Context, accountID string) (bool, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_sender.delete")
	defer span.End()
	rows, err := r.db.DeleteEmailSender(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *accountEmailSenderRepoImpl) DeleteByDomain(ctx context.Context, emailDomainID, accountID string) (bool, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_sender.delete_by_domain")
	defer span.End()
	rows, err := r.db.DeleteEmailSenderByDomain(ctx, sqlc.DeleteEmailSenderByDomainParams{EmailDomainID: emailDomainID, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

// ── email_inbox ──

type emailInboxRepoImpl struct {
	db *sqlc.Queries
}

func NewEmailInboxRepo(db *sqlc.Queries) domain.EmailInboxRepo {
	return &emailInboxRepoImpl{db: db}
}

func (r *emailInboxRepoImpl) Create(ctx context.Context, id, accountID string, input *domain.CreateEmailInboxInput) *apierror.APIError {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.create")
	defer span.End()
	err := r.db.CreateEmailInbox(ctx, sqlc.CreateEmailInboxParams{
		ID:                   id,
		AccountID:            accountID,
		EmailDomainID:        input.EmailDomainID,
		Address:              input.Address,
		FromName:             db.NullStringPtr(input.FromName),
		AgentConfigID:        db.NullStringPtr(input.AgentConfigID),
		AgentTriggerPolicy:   db.NullStringPtr(input.AgentTriggerPolicy),
		AgentTriggerKeywords: marshalStringSlice(input.AgentTriggerKeywords),
		GroupID:              db.NullStringPtr(input.GroupID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *emailInboxRepoImpl) GetByID(ctx context.Context, id, accountID string) (*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.get_by_id")
	defer span.End()
	row, err := r.db.GetEmailInboxByID(ctx, sqlc.GetEmailInboxByIDParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return emailInboxFromRow(row), nil
}

func (r *emailInboxRepoImpl) GetByAddress(ctx context.Context, address string) (*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.get_by_address")
	defer span.End()
	row, err := r.db.GetEmailInboxByAddress(ctx, address)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return emailInboxFromRow(row), nil
}

func (r *emailInboxRepoImpl) GetByIDSystem(ctx context.Context, id string) (*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.get_by_id_system")
	defer span.End()
	row, err := r.db.GetEmailInboxByIDSystem(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return emailInboxFromRow(row), nil
}

func (r *emailInboxRepoImpl) ListByAccount(ctx context.Context, accountID string) ([]*domain.EmailInbox, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.list_by_account")
	defer span.End()
	rows, err := r.db.ListEmailInboxesByAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.EmailInbox, 0, len(rows))
	for _, row := range rows {
		out = append(out, emailInboxFromRow(row))
	}
	return out, nil
}

func (r *emailInboxRepoImpl) CountByDomain(ctx context.Context, accountID, emailDomainID string) (int64, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.count_by_domain")
	defer span.End()
	n, err := r.db.CountEmailInboxesByDomain(ctx, sqlc.CountEmailInboxesByDomainParams{AccountID: accountID, EmailDomainID: emailDomainID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return n, nil
}

func (r *emailInboxRepoImpl) Update(ctx context.Context, id, accountID string, input *domain.UpdateEmailInboxInput) *apierror.APIError {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.update")
	defer span.End()
	err := r.db.UpdateEmailInbox(ctx, sqlc.UpdateEmailInboxParams{
		FromName:             db.NullStringPtr(input.FromName),
		Status:               input.Status,
		AgentConfigID:        db.NullStringPtr(input.AgentConfigID),
		AgentTriggerPolicy:   db.NullStringPtr(input.AgentTriggerPolicy),
		AgentTriggerKeywords: marshalStringSlice(input.AgentTriggerKeywords),
		GroupID:              db.NullStringPtr(input.GroupID),
		ID:                   id,
		AccountID:            accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *emailInboxRepoImpl) Delete(ctx context.Context, id, accountID string) (bool, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_inbox.delete")
	defer span.End()
	n, err := r.db.DeleteEmailInbox(ctx, sqlc.DeleteEmailInboxParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return n > 0, nil
}

func emailInboxFromRow(row sqlc.EmailInbox) *domain.EmailInbox {
	return &domain.EmailInbox{
		ID:                   row.ID,
		AccountID:            row.AccountID,
		EmailDomainID:        row.EmailDomainID,
		Address:              row.Address,
		FromName:             db.StringFromNullString(row.FromName),
		Status:               row.Status,
		AgentConfigID:        db.StringFromNullString(row.AgentConfigID),
		AgentTriggerPolicy:   db.StringFromNullString(row.AgentTriggerPolicy),
		AgentTriggerKeywords: unmarshalStringSlice(row.AgentTriggerKeywords),
		GroupID:              db.StringFromNullString(row.GroupID),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

// ── email_message ──

type emailMessageRepoImpl struct {
	db *sqlc.Queries
}

func NewEmailMessageRepo(db *sqlc.Queries) domain.EmailMessageRepo {
	return &emailMessageRepoImpl{db: db}
}

func (r *emailMessageRepoImpl) TryInsert(ctx context.Context, input *domain.CreateEmailMessageInput) (bool, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_message.try_insert")
	defer span.End()
	n, err := r.db.TryInsertEmailMessage(ctx, sqlc.TryInsertEmailMessageParams{
		ID:             input.ID,
		AccountID:      input.AccountID,
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
		EmailInboxID:   input.EmailInboxID,
		Direction:      input.Direction,
		RfcMessageID:   input.RfcMessageID,
		InReplyTo:      db.NullStringPtr(input.InReplyTo),
		References:     db.NullStringPtr(input.References),
		FromAddr:       input.FromAddr,
		ToAddrs:        input.ToAddrs,
		CcAddrs:        db.NullStringPtr(input.CcAddrs),
		Subject:        db.NullStringPtr(input.Subject),
		RawS3Key:       db.NullStringPtr(input.RawS3Key),
		SesMessageID:   db.NullStringPtr(input.SesMessageID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return n > 0, nil
}

func (r *emailMessageRepoImpl) GetByRfcID(ctx context.Context, rfcMessageID string) (*domain.EmailMessage, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_message.get_by_rfc_id")
	defer span.End()
	row, err := r.db.GetEmailMessageByRfcID(ctx, rfcMessageID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return emailMessageFromRow(row), nil
}

func (r *emailMessageRepoImpl) FindThreadConversation(ctx context.Context, rfcMessageIDs []string) (*domain.EmailThreadMatch, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_message.find_thread_conversation")
	defer span.End()
	if len(rfcMessageIDs) == 0 {
		return nil, apierror.NewResourceNotFoundError("No thread match.")
	}
	row, err := r.db.FindEmailThreadConversation(ctx, rfcMessageIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.EmailThreadMatch{ConversationID: row.ConversationID, EmailInboxID: row.EmailInboxID}, nil
}

func (r *emailMessageRepoImpl) GetLatestInbound(ctx context.Context, conversationID string) (*domain.EmailMessage, *apierror.APIError) {
	ctx, span := emailRepoTracer.Start(ctx, "repository.email_message.get_latest_inbound")
	defer span.End()
	row, err := r.db.GetLatestInboundEmailMessage(ctx, conversationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return emailMessageFromRow(row), nil
}

func emailMessageFromRow(row sqlc.EmailMessage) *domain.EmailMessage {
	return &domain.EmailMessage{
		ID:             row.ID,
		AccountID:      row.AccountID,
		ConversationID: row.ConversationID,
		MessageID:      row.MessageID,
		EmailInboxID:   row.EmailInboxID,
		Direction:      row.Direction,
		RfcMessageID:   row.RfcMessageID,
		InReplyTo:      db.StringFromNullString(row.InReplyTo),
		References:     db.StringFromNullString(row.References),
		FromAddr:       row.FromAddr,
		ToAddrs:        row.ToAddrs,
		CcAddrs:        db.StringFromNullString(row.CcAddrs),
		Subject:        db.StringFromNullString(row.Subject),
		RawS3Key:       db.StringFromNullString(row.RawS3Key),
		SesMessageID:   db.StringFromNullString(row.SesMessageID),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
