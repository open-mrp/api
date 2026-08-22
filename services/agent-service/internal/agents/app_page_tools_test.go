package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestHandleFindAppPage_ReturnsWritableLinks: the tool's whole job is to hand the agent link text it can
// copy, since the failure it replaces is a confidently invented `/dashboard/...` URL.
func TestHandleFindAppPage_ReturnsWritableLinks(t *testing.T) {
	out, err := HandleFindAppPage(context.Background(), json.RawMessage(`{"query":"customer prices"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "(openmrp:page/customer-prices)") {
		t.Errorf("no page link in result:\n%s", out)
	}
	// The record form matters more than the list page: an agent that just looked up a price wants to link it.
	if !strings.Contains(out, "openmrp:account_price/<id>") {
		t.Errorf("no record-link form for the page's record type:\n%s", out)
	}
	if !strings.Contains(out, "Sales › Pricing") {
		t.Errorf("no navigation breadcrumb:\n%s", out)
	}
}

func TestHandleFindAppPage_NoQueryListsPages(t *testing.T) {
	out, err := HandleFindAppPage(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, "page link:") < 5 {
		t.Errorf("expected a browsable list of pages:\n%s", out)
	}
}

func TestHandleFindAppPage_NoMatchSaysSo(t *testing.T) {
	out, err := HandleFindAppPage(context.Background(), json.RawMessage(`{"query":"zzzzqqqx"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No app page matched") {
		t.Errorf("want an explicit miss, got:\n%s", out)
	}
}

func TestHandleFindAppPage_RejectsMalformedInput(t *testing.T) {
	if _, err := HandleFindAppPage(context.Background(), json.RawMessage(`nope`), nil); err == nil {
		t.Error("want an error for malformed input")
	}
}
