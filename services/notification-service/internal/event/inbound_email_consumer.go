package event

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/augno/api/services/notification-service/internal/domain"
	s3client "github.com/augno/api/shared/cloud/s3"
	sqsclient "github.com/augno/api/shared/cloud/sqs"

	"go.opentelemetry.io/otel/trace"
)

// InboundEmailConsumer polls the inbound-email SQS queue. Each message is an S3 ObjectCreated event for a raw .eml the SES receipt rule stored; the consumer fetches it, parses the MIME, and hands the result to the conversation service to thread + dispatch. Dedup is by rfc Message-ID downstream.
type InboundEmailConsumer struct {
	queue   sqsclient.Queue
	store   s3client.ObjectStore
	chatSvc domain.ConversationSvc
	tracer  trace.Tracer
}

func NewInboundEmailConsumer(queue sqsclient.Queue, store s3client.ObjectStore, chatSvc domain.ConversationSvc, tracer trace.Tracer) *InboundEmailConsumer {
	return &InboundEmailConsumer{queue: queue, store: store, chatSvc: chatSvc, tracer: tracer}
}

// Listen starts the polling loop in a goroutine and returns immediately.
func (c *InboundEmailConsumer) Listen(ctx context.Context) error {
	go c.loop(ctx)
	return nil
}

func (c *InboundEmailConsumer) loop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		msgs, apiErr := c.queue.Receive(ctx, 10, 20) // long poll up to 20s
		if apiErr != nil {
			if ctx.Err() != nil {
				return
			}
			slog.ErrorContext(ctx, "inbound email: SQS receive failed", "error", apiErr)
			// Back off briefly so a persistent failure doesn't spin.
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		for _, m := range msgs {
			// handle returns true when the message is fully handled (or permanently undeliverable) and should be acked; false leaves it for redelivery (and eventually the DLQ).
			if c.handle(ctx, m.Body) {
				if delErr := c.queue.Delete(ctx, m.ReceiptHandle); delErr != nil {
					slog.ErrorContext(ctx, "inbound email: SQS delete failed", "error", delErr)
				}
			}
		}
	}
}

// s3EventNotification is the subset of an S3 event-notification payload we read.
type s3EventNotification struct {
	Event   string `json:"Event"` // set to "s3:TestEvent" for the validation ping S3 sends on setup
	Records []struct {
		S3 struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				Key string `json:"key"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}

func (c *InboundEmailConsumer) handle(ctx context.Context, body string) bool {
	ctx, span := c.tracer.Start(ctx, "consumer.inbound_email.handle")
	defer span.End()

	var evt s3EventNotification
	if err := json.Unmarshal([]byte(body), &evt); err != nil {
		slog.ErrorContext(ctx, "inbound email: bad event JSON, dropping", "error", err)
		return true // malformed — retry won't help
	}
	if evt.Event == "s3:TestEvent" {
		return true // S3 setup ping
	}

	ok := true
	for _, r := range evt.Records {
		key, err := url.QueryUnescape(r.S3.Object.Key)
		if err != nil {
			key = r.S3.Object.Key
		}
		if !c.processObject(ctx, r.S3.Bucket.Name, key) {
			ok = false // a transient failure on any record → leave the whole message for redelivery
		}
	}
	return ok
}

// processObject fetches and ingests one raw email. Returns false only for transient failures worth a retry; permanent problems (unparseable mail) are logged and treated as handled.
func (c *InboundEmailConsumer) processObject(ctx context.Context, bucket, key string) bool {
	raw, apiErr := c.store.Get(ctx, bucket, key)
	if apiErr != nil {
		slog.ErrorContext(ctx, "inbound email: S3 get failed", "bucket", bucket, "key", key, "error", apiErr)
		return false // transient (e.g. eventual consistency / permissions) — retry
	}
	in, err := parseInboundEmail(raw, key)
	if err != nil {
		slog.ErrorContext(ctx, "inbound email: parse failed, dropping", "key", key, "error", err)
		return true // unparseable — won't improve on retry
	}
	if ingestErr := c.chatSvc.IngestInboundEmail(ctx, in); ingestErr != nil {
		slog.ErrorContext(ctx, "inbound email: ingest failed", "key", key, "error", ingestErr)
		return false // DB/thread error — retry
	}
	return true
}

// parseInboundEmail turns a raw rfc822 message into the structured ingest input: headers + a best-effort plain-text body.
func parseInboundEmail(raw []byte, s3Key string) (domain.IngestInboundEmailInput, error) {
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return domain.IngestInboundEmailInput{}, err
	}
	h := msg.Header
	dec := new(mime.WordDecoder)
	decode := func(s string) string {
		if out, derr := dec.DecodeHeader(s); derr == nil {
			return out
		}
		return s
	}

	fromAddr, fromName := "", ""
	if addr, perr := mail.ParseAddress(h.Get("From")); perr == nil {
		fromAddr = addr.Address
		fromName = decode(addr.Name)
	} else {
		fromAddr = strings.TrimSpace(h.Get("From"))
	}

	body := extractText(msg, h.Get("Content-Type"))

	return domain.IngestInboundEmailInput{
		Recipients:   candidateRecipients(h),
		From:         fromAddr,
		FromName:     fromName,
		Subject:      decode(h.Get("Subject")),
		TextBody:     body,
		RfcMessageID: strings.Trim(strings.TrimSpace(h.Get("Message-Id")), "<>"),
		InReplyTo:    strings.Trim(strings.TrimSpace(h.Get("In-Reply-To")), "<>"),
		References:   parseReferences(h.Get("References")),
		RawS3Key:     s3Key,
	}, nil
}

// candidateRecipients collects every address the mail could have been delivered to — across the
// delivery headers (Delivered-To, X-Original-To) and the visible recipient headers (To, Cc) — lowercased
// and de-duplicated. Ingestion matches these against known inboxes rather than trusting one "delivered"
// header: a forwarding hop (M365/Barracuda) rewrites Delivered-To/X-Original-To to its own forward target,
// so the original inbox address survives only in To/Cc, while the per-inbox forwarding address survives
// only in the delivery headers. Collecting all of them lets either routing path resolve.
func candidateRecipients(h mail.Header) []string {
	seen := map[string]bool{}
	var out []string
	add := func(addr string) {
		addr = strings.ToLower(strings.TrimSpace(addr))
		if addr != "" && !seen[addr] {
			seen[addr] = true
			out = append(out, addr)
		}
	}
	for _, hdr := range []string{"Delivered-To", "X-Original-To", "To", "Cc"} {
		v := h.Get(hdr)
		if v == "" {
			continue
		}
		if addrs, err := mail.ParseAddressList(v); err == nil {
			for _, a := range addrs {
				add(a.Address)
			}
			continue
		}
		add(v)
	}
	return out
}

// parseReferences splits the References header (whitespace-separated message-ids) and strips the angle brackets.
func parseReferences(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	var out []string
	for _, f := range strings.Fields(v) {
		if id := strings.Trim(f, "<>"); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// extractText returns a best-effort plain-text body: the first text/plain part of a multipart message, falling back to the raw decoded body (HTML stripped) otherwise.
func extractText(msg *mail.Message, contentType string) string {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		raw, _ := io.ReadAll(msg.Body)
		return cleanText(string(raw), mediaType)
	}
	boundary := params["boundary"]
	if boundary == "" {
		raw, _ := io.ReadAll(msg.Body)
		return cleanText(string(raw), mediaType)
	}
	mr := multipart.NewReader(msg.Body, boundary)
	var htmlFallback string
	for {
		part, perr := mr.NextPart()
		if perr != nil {
			break
		}
		pt, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		data, _ := io.ReadAll(part)
		switch {
		case strings.HasPrefix(pt, "text/plain"):
			return strings.TrimSpace(string(data))
		case strings.HasPrefix(pt, "text/html") && htmlFallback == "":
			htmlFallback = stripHTML(string(data))
		}
	}
	return strings.TrimSpace(htmlFallback)
}

func cleanText(body, mediaType string) string {
	if strings.HasPrefix(mediaType, "text/html") {
		return stripHTML(body)
	}
	return strings.TrimSpace(body)
}

var (
	htmlTagRe   = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlAnyTag  = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlSpaceRe = regexp.MustCompile(`[ \t]*\n[ \t\n]*`)
)

// stripHTML reduces an HTML body to readable text: drop script/style blocks and tags, collapse runs of blank lines. Good enough to seed an agent or show a preview; not a full renderer.
func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = htmlAnyTag.ReplaceAllString(s, " ")
	s = htmlSpaceRe.ReplaceAllString(s, "\n")
	return strings.TrimSpace(s)
}
