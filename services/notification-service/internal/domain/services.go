package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

type NotificationSvc interface {
	// SendEmail sends an email to the given recipients.
	//
	//  1. Calls the email sender to send the email.
	//  2. Returns the SES message ID.
	SendEmail(ctx context.Context, data EmailSendData) (*string, *apierror.APIError)

	// LogEmail logs an email to the database.
	//
	//  1. Checks if the email has already been logged.
	//  2. If the email has not been logged, creates a new email log.
	LogEmail(ctx context.Context, data EmailLogData) *apierror.APIError

	// SendEnterpriseRequest sends an enterprise upgrade request email to sales
	//
	//  1. Renders the email template.
	//  2. Sends the email to the sales team.
	//  3. Returns an error if the email sender returns an error.
	SendEnterpriseRequest(ctx context.Context, req *EnterpriseRequestData) *apierror.APIError
}

type EmailLogData struct {
	SesMessageID string  `json:"ses_message_id"`
	AccountID    *string `json:"account_id,omitempty"`
	SentByID     *string `json:"sent_by_id,omitempty"`
	Subject      string  `json:"subject"`
	Filename     *string `json:"filename,omitempty"`
}

type EmailSendData struct {
	To        []string `json:"to"`
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
	SendAs    *string  `json:"send_as,omitempty"`
	AccountID *string  `json:"account_id,omitempty"`
	SentByID  *string  `json:"sent_by_id,omitempty"`
}

// EnterpriseRequestData contains data for an enterprise upgrade request email
type EnterpriseRequestData struct {
	AccountID       string
	AccountName     string
	CurrentPlanName string
	RequesterName   string
	RequesterEmail  string
}

type EmailSender interface {
	Send(ctx context.Context, data EmailData) (*string, *apierror.APIError)
}

type EmailData struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	SendAs  *string  `json:"send_as,omitempty"`
}
