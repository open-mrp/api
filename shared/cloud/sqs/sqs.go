// Package sqs is a thin wrapper over the AWS SQS SDK for the consumers that poll a queue (currently the inbound-email bridge). It exposes long-poll receive + delete and nothing else.
package sqs

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var sqsTracer = tracing.GetTracer("shared.cloud.sqs")

// Message is a received SQS message: its body plus the receipt handle needed to delete it.
type Message struct {
	Body          string
	ReceiptHandle string
}

// Queue defines the SQS operations the consumers need. An interface so consumers can be unit-tested against a fake.
type Queue interface {
	Receive(ctx context.Context, maxMessages, waitSeconds int32) ([]Message, *apierror.APIError)
	Delete(ctx context.Context, receiptHandle string) *apierror.APIError
}

// Client wraps the AWS SQS SDK client bound to a single queue URL.
type Client struct {
	client   *sqs.Client
	queueURL string
}

// NewClient creates an SQS client for the given region, bound to queueURL.
//
// IMDS is disabled for the same reason as the S3 client: pods get credentials via IRSA (or static keys in dev), never the node instance role, and leaving IMDS in the chain blocks credential resolution for the full deadline on every call.
func NewClient(ctx context.Context, region, queueURL string) (*Client, *apierror.APIError) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithEC2IMDSClientEnableState(imds.ClientDisabled),
	)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to load AWS configuration for SQS.")
	}
	return &Client{client: sqs.NewFromConfig(cfg), queueURL: queueURL}, nil
}

// Receive long-polls the queue for up to maxMessages, waiting up to waitSeconds for a message.
func (c *Client) Receive(ctx context.Context, maxMessages, waitSeconds int32) ([]Message, *apierror.APIError) {
	ctx, span := sqsTracer.Start(ctx, "sqs.receive")
	defer span.End()

	out, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     waitSeconds,
	})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to receive SQS messages."))
	}
	msgs := make([]Message, 0, len(out.Messages))
	for _, m := range out.Messages {
		msgs = append(msgs, Message{Body: aws.ToString(m.Body), ReceiptHandle: aws.ToString(m.ReceiptHandle)})
	}
	return msgs, nil
}

// Delete acknowledges a message so it is not redelivered.
func (c *Client) Delete(ctx context.Context, receiptHandle string) *apierror.APIError {
	ctx, span := sqsTracer.Start(ctx, "sqs.delete")
	defer span.End()

	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to delete SQS message."))
	}
	return nil
}
