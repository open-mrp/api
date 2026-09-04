# README screenshots

Shots of the product for the root `README.md`. It references five today: `landing` as the hero, then `production-flow-full`, `production-schedule`, `agent-run`, and `request-logs`, one per feature callout. The rest are captured and available; nothing here renders broken.

## Capture rules

- **Viewport 1600×1000, 2× DPR**, then downscale to 1600px wide. GitHub renders README images at roughly 850px, so 2× keeps text crisp.
- **Dark theme.** The nine existing shots are all dark; anything new has to match or the gallery looks accidental.
- **Real seeded data, never lorem.** `make local-db` seeds accounts, items, orders, and users — run the flow far enough that charts have shape and lists have depth. Empty states and single-row tables read as "unfinished," not "clean."
- **Crop the browser chrome.** No URL bar, no bookmarks, no OS menu bar, no cursor.
- **Scrub identity.** No real customer names, no personal email addresses, no live account IDs. The seed data is already fictional — keep it that way when you re-shoot against a demo tenant.
- **Trim the empty right edge.** Wide tables with three used columns and a sea of white waste the reader's screen; either widen the data or crop.

## The shots

| File | Route | Status | What it has to show |
| --- | --- | --- | --- |
| `dashboard.png` | `/dashboard` | shot, unused | **Hero.** Full app chrome — nav rail, header, widgets. The current shot has an empty alerts card and a single message; re-shoot against a busier tenant when one exists. |
| `production-flow-full.png` | `/dashboard/production-flow` | shot | **Hero #2.** The whole BOM / production-step graph, zoomed to fit. The most distinctive screen in the product and the one no spreadsheet can imitate. |
| `production-flow.png` | `/dashboard/production-flow` | shot, unused | The same graph at close range, so per-edge quantities and node labels stay legible at README width. |
| `production-schedule.png` | `/dashboard/production-schedules/[id]` | shot | Scheduled work by SKU across weeks, with run hours and utilization. Show enough columns that the scheduling is obviously non-trivial. |
| `sales-order.png` | `/dashboard/sales-orders` | shot, unused | The order list with filters engaged — status, ship-by, payment terms, row actions. The screen most readers map to their own business. |
| `scanning-stations.png` | `/dashboard/scanning-stations` | shot, unused | Stations with their batch operations and label formats, one per shop-floor step. |
| `agent-chat.png` | `/dashboard/inbox` | shot, unused | An agent @mentioned in a thread, asking a clarifying question instead of guessing. |
| `agent-run.png` | `/dashboard/inbox` | shot | The approval gate: a write-capable tool call held, then approved by name. This is the differentiator — keep the tool calls legible when scaled down. |
| `request-logs.png` | `/dashboard/request-logs` | shot | Method, path, status, timing, caller. The trust shot for developers — proof the UI runs on the same public API they'd call. |
| `inventory.png` | `/dashboard/inventory-logs` or a storage-location detail | **not shot** | Quantities by location and lot. If the reconciliation view has variance rows, prefer it — variance is more interesting than agreement. |
| `customer-portal.png` | `/[accountSlug]/dashboard` (customer side) | **not shot** | The branded portal a customer sees. Use seeded branding so it clearly isn't the internal dashboard — that contrast is the entire point of the shot. |
| `manufacturing-analytics.png` | `/dashboard/manufacturing-analytics` | **not shot** | Charts with real curves. A flat line or a single bar undersells it; seed enough history that trends exist. |
| `landing.png` | `openmrp.ai`, signed out | shot | **Hero.** The marketing landing page. `(marketing)/layout.tsx` wraps it in `GuestGuard redirectAuthenticated`, so a signed-in browser bounces to `/dashboard` — shoot it signed out, or locally at `/dev/landing`. The hero animates in on scroll, so scroll down and back up before capturing or the copy shoots blank. |

## Optional extras

Worth having on hand, though the README doesn't reference them today: the shared inbox (`/dashboard/inbox`), picking (`/dashboard/picking/[id]`), demand forecast (`/dashboard/demand-forecast`), and the API key / sandbox screens (`/dashboard/sandboxes`).

## Animation

If any one of these becomes a GIF, make it `production-flow-full.png` → a short loop of building a flow, or `agent-run.png` → an agent proposing an action and a human approving it. Keep it under ~8 seconds and under 5 MB; a heavy GIF at the top of a README costs more than it earns.
