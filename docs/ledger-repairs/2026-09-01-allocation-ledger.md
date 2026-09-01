# 2026-09-01 — inventory allocation ledger repair

A record of rows changed directly in `augno_core` / `prod`, kept because the ledger has no foreign
keys and no capacity constraints: nothing in the schema explains why an issue's coverage does not
equal its demand, so the explanation has to live somewhere a person can find it. Six months from
now the question will be "why does `inrs_sufbtcm0xibo` read 104 of 240", and this is the answer.

Found by `cmd/inventory-invariant-check` on its first run against production.

## What was wrong

| Detector | Rows |
|---|---|
| `over_drawn_receipts` | 1 |
| `non_positive_allocations` | 16 |
| `orphaned_allocations` | 1 |

All three are now zero, as are `over_allocated_issues` and `duplicate_allocations`.

`closed_issue_not_covered` matched 656 issues, 543 of them with no allocations at all. That is a
pre-existing backlog, was not touched, and is why the detector ships report-only.

## Part 1 — the over-drawn receipt and the degenerate rows

Receipt `inrp_bdchkwydwk2t` held 180.5 pairs (361 base units) and carried 363. Its two large
allocations were written 36 seconds apart on 2026-08-25 — the double-allocation race commit
`7044443a` was written to close, with one survivor. Repaired the way
`cmd/repair-overallocated-receipts` does it: drop the newest allocation whole, reopen the demand it
uncovers. `inrs_8qh1tp70h8z2` had that allocation and no other, so it went back to `open`.

Deleted alongside it: twelve allocations of exactly zero, one of -3.55e-15 (2^-48, an IEEE754
residue from `Number()` on the dashboard's write path), and one allocation whose issue and receipt
still existed while all three satellite rows were gone — the signature of a half-applied delete,
which is what an untransacted reversal produces. Removing them changed no issue's coverage and no
receipt's status; that was verified before and after.

## Part 2 — the three compensating negatives

Not corruption. Two of them balanced an over-sized positive back to the issue's exact demand, the
positive being over-sized because it was written in a different unit with the inverted conversion
`b3e12c25` later fixed. The third was residue from an allocation moved between receipts on
2026-02-23, where the mover wrote a cancelling negative *and* deleted the original, counting the
release twice.

| Issue | Demand | Before | After |
|---|---|---|---|
| `inrs_01kbmw4be8e0yrkgq9thtgkwcg` | 80 | +137 and -57 | one allocation of 80 |
| `inrs_krdwfsy9i1en` | 48 | +70.000000000000008 and -22 | one allocation of 48 |
| `inrs_sufbtcm0xibo` | 240 | +104 and -104 | one allocation of 104 |

Each over-sized positive was shrunk to the demand it covered, with its total cost moved to match —
cost is priced per pair, so `total = (base_units / 2) * unit_cost`, a formula that reproduced both
recorded costs exactly, which is what identified the quantity rather than the unit cost as the
corrupted field. Then the negatives and their satellites were deleted.

The first two issues carry the same net coverage they did before. The third reads 104 of 240, which
is what receipt `inrp_s3jv1smzxxvj` genuinely supplied.

## The assumption, and how to reverse it

All three orders were `fulfilled`, so the goods had shipped and the issue-level totals were taken
as correct. The stock behind the capacity that no longer has an allocation against it is recorded
as **consumed**, not on the shelf. Physical verification was not available.

The direction was deliberate. Assuming consumed leaves stock hidden, which costs money and
self-corrects at the next physical count. Assuming on-shelf invents inventory that may not exist,
and that surfaces as overselling to a customer. So no receipt was freed and no fulfilled demand was
reopened.

Three receipts are consequently marked `allocated` while carrying less draw than their capacity:

| Receipt | Capacity (base) | Drawn | Unexplained |
|---|---|---|---|
| `inrp_a44g88hdy74e` | 108 | 0 | 108 (54 pairs) |
| `inrp_01kbkckycfeey979bphyebd8gw` | 180 | 157 | 23 |
| `inrp_01kbzvegvee6gvfebqnx47zc01` | 180 | 154 | 26 |

If a physical count ever shows that stock is real, setting those receipts back to `available` is
the whole correction — the allocator will offer them again on its next pass for the item. Do not do
it without a count.

## Why the writers stopped

Nothing has written a non-positive allocation since 2026-03-02. All sixteen were order-linked, none
batch-linked, and the writer was the dashboard's per-issue `allocateOpenIssueById`, which started
from the issue's full quantity rather than its remaining. `dc141f68` (2026-03-10) replaced it with a
batch planner that guards on `.lte(0)` in decimal.

Still open: that planner writes through `Number(clonedQty.measure)` at
`dashboard/apps/api/src/repositories/inventory-issue.repo.ts`, so a float64 conversion remains on the
write path. It is the source of the 2^-48 residue and of quantities small enough to round to zero in
a `DECIMAL(65,30)` column. Consolidating allocation onto Go retires it.
