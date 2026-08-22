package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/shared/appnav"
)

// FindAppPageSlug is the meta-tool a chat agent uses to find the app pages it can link to. Like
// search_api_tools it is a runtime capability the runner injects rather than a tool a merchant grants:
// linking a page is part of how a chat reply is written, and every existing agent would otherwise be
// stuck describing menu paths.
const FindAppPageSlug = "find_app_page"

// FindAppPageDescription is what the model reads when deciding to call the tool. It states the one
// thing the agent cannot work out on its own — that link keys come from here rather than from a
// guessed URL — because the failure this tool exists to prevent is a confidently invented path.
const FindAppPageDescription = "Find the pages of this app so you can link a user to one. Describe the page in plain language (e.g. \"customer prices\", \"where discounts are set up\") and this returns the matching pages with the exact link to write for each. Use it whenever you want to point someone at a screen — a list, a settings page, or the detail page for a kind of record — instead of guessing a URL or describing where to click. Call it with no query to see every page."

// FindAppPageInputSchema is the tool's parameter schema as shown to the model.
const FindAppPageInputSchema = `{"type":"object","properties":{"query":{"type":"string","description":"Plain-language description of the page you want to link, e.g. \"customer prices\" or \"sales order list\". Omit to list every page."}}}`

// findAppPageLimit caps how many pages one lookup returns, keeping the result readable when a broad
// query matches a whole section.
const findAppPageLimit = 12

// HandleFindAppPage answers a page lookup with the exact markdown links to write.
//
// The result gives the agent finished link text rather than parts to assemble, because assembling is
// where it went wrong before: it would reason its way to a plausible `/dashboard/...` URL that the
// chat renderer has no way to resolve. For a page whose detail view shows a kind of record, the
// record-link form is included too, since an agent that has just looked up (say) a contracted price
// usually wants to link that record and not the list it lives on.
func HandleFindAppPage(_ context.Context, input json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			return "", fmt.Errorf("invalid find_app_page input: %w", err)
		}
	}

	matches := appnav.Search(params.Query, findAppPageLimit)
	if len(matches) == 0 {
		return fmt.Sprintf("No app page matched %q. Try a broader description, or call this tool with no query to list every page.", params.Query), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d matching page(s). Write the link exactly as shown:\n", len(matches))
	for _, p := range matches {
		fmt.Fprintf(&b, "- %s", p.Title)
		if loc := breadcrumb(p); loc != "" {
			fmt.Fprintf(&b, " (%s)", loc)
		}
		fmt.Fprintf(&b, "\n  page link: [%s](openmrp:page/%s)\n", p.Title, p.Key)
		if p.RecordType != "" {
			fmt.Fprintf(&b, "  link one %s record: [<its number or name>](openmrp:%s/<id>)\n", strings.ReplaceAll(p.RecordType, "_", " "), p.RecordType)
		}
	}
	return b.String(), nil
}

// breadcrumb renders where the page sits in the navigation, which is how users refer to screens ("under Sales › Pricing").
func breadcrumb(p appnav.Page) string {
	if p.Subsection == "" {
		return p.Section
	}
	return p.Section + " › " + p.Subsection
}
