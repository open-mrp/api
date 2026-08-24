# README screenshots

The ten images the root `README.md` expects, in the order a reader meets them. Until a file lands here its slot renders as a broken image on GitHub — capture them, or delete that block from the README.

## Capture rules

- **Viewport 1600×1000, 2× DPR**, then downscale to 1600px wide. GitHub renders README images at roughly 850px, so 2× keeps text crisp.
- **One theme for all ten.** Pick light or dark and stay there; a mixed gallery looks accidental.
- **Real seeded data, never lorem.** `make local-db` seeds accounts, items, orders, and users — run the flow far enough that charts have shape and lists have depth. Empty states and single-row tables read as "unfinished," not "clean."
- **Crop the browser chrome.** No URL bar, no bookmarks, no OS menu bar, no cursor.
- **Scrub identity.** No real customer names, no personal email addresses, no live account IDs. The seed data is already fictional — keep it that way when you re-shoot against a demo tenant.
- **Trim the empty right edge.** Wide tables with three used columns and a sea of white waste the reader's screen; either widen the data or crop.

## The shots

| File | Route | What it has to show |
| --- | --- | --- |
| `dashboard.png` | `/dashboard` | **Hero.** The first thing anyone sees. Full app chrome — nav rail, header, populated widgets. This one image has to say "this is a real, dense operational product," so make sure every card has data. |
| `production-flow.png` | `/dashboard/production-flow` | **Hero #2.** The BOM / production step graph. It's the most visually distinctive screen in the product and the one no spreadsheet can imitate — a graph with 8–15 nodes and visible branching beats a tidy 4-node line. |
| `production-schedule.png` | `/dashboard/production-schedules/[id]` | Scheduled work across machines and days. Show enough rows that the scheduling is obviously non-trivial. |
| `sales-order.png` | `/dashboard/sales-orders/[id]` | One order detail: line items, pricing, status, fulfillment. The screen most readers will mentally map to their own business. |
| `inventory.png` | `/dashboard/inventory-logs` or a storage-location detail | Quantities by location and lot. If the reconciliation view has variance rows, prefer it — variance is more interesting than agreement. |
| `agent-run.png` | `/dashboard/agents/runs/[id]` | An agent run expanded: steps, tool calls, the approval gate. This is the differentiator — make the tool calls legible even when scaled down. |
| `scanning-station.png` | `/scanning-station/[id]` | The shop-floor view, ideally at a narrow/tablet viewport to signal it's meant for a device on the floor. Crop to the device frame, not the desktop window. |
| `customer-portal.png` | `/[accountSlug]/dashboard` (customer side) | The branded portal a customer sees. Use seeded branding so it clearly isn't the internal dashboard — that contrast is the entire point of the shot. |
| `manufacturing-analytics.png` | `/dashboard/manufacturing-analytics` | Charts with real curves. A flat line or a single bar undersells it; seed enough history that trends exist. |
| `request-logs.png` | `/dashboard/request-logs/[id]` | One request log: method, path, status, timing, payload. The trust shot for developers — proof the UI runs on the same public API they'd call. |

## Optional extras

Worth having on hand, though the README doesn't reference them today: the shared inbox (`/dashboard/inbox`), picking (`/dashboard/picking/[id]`), demand forecast (`/dashboard/demand-forecast`), and the API key / sandbox screens (`/dashboard/sandboxes`).

## Animation

If any one of these becomes a GIF, make it `production-flow.png` → a short loop of building a flow, or `agent-run.png` → an agent proposing an action and a human approving it. Keep it under ~8 seconds and under 5 MB; a heavy GIF at the top of a README costs more than it earns.
