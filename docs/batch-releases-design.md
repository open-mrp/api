# Batch releases — electronic sign-off for regulated production

Lets an account replace the paper batch traveler with electronic records that satisfy a medical-device quality system (ISO 13485 §4.2.5 control of records, §7.5.8 identification, §7.5.9 traceability, §8.2.6 release of product; 21 CFR Part 11 for electronic signatures where FDA rules apply). Status: **phases 1–3 built** (API and dashboard). Phase 4 (lots on picks and shipments) not started.

## What the paper traveler does today

The printed `batch_traveler` document lists each department the batch passes through with its firsts, seconds and waste (filled from scans), then leaves blank **Shift** and **Released By** columns, two checkboxes ("Labeling and packaging correct?", "Bag attached?"), a release signature, a release date and a space for the bag's date code. The blanks are what this design makes electronic.

## Requirements

1. Every person who records data or signs is individually identified. Access is granted by role and documented.
2. Operators record data at their station. Data is stored as entered; a correction is approved by a supervisor or quality inspector.
3. Each production step decides, on a documented risk basis, whether a batch needs a supervisor or quality sign-off before it moves on.
4. Supervisors sign off for their own area only. Quality can sign off anywhere.
5. Quality gives a final release before the product is packed and shipped; the release records who released it, when, and the answers to a release checklist.
6. The final release assigns a lot number that traces back to the electronic production record and to the material lots consumed.
7. Records cannot be edited or deleted after the fact. A mistake is fixed by a new record that supersedes it.

## Concepts

### Release requirement (per department)

New fields on `department` (the work center after forge.1). A batch is in the department of the scanning station it was last scanned at, so one policy covers every production step done there: production steps are per SKU, so an account has far more of them than departments, which usually map one to one onto stations. Product line isn't used: only finished products carry one, so in-process parts at every department but packing have none.

| Field | Values | Meaning |
|---|---|---|
| `release_requirement` | `none` (default), `supervisor`, `quality`, `first_article`, `final` | Who must sign before a batch leaves this step. `first_article` verifies the set-up: the run's first batch at this step needs a supervisor **and** a quality sign-off, and once one batch of a production run has both, every batch of that run at this step counts as released. `final` is a `quality` sign-off that also releases the product and assigns its lot. |
| `release_enforcement` | `block` (default), `warn` | `block`: the next scan is refused until the batch is signed. `warn`: the scan proceeds, the missing sign-off shows on the batch record, and the final release is refused until every earlier sign-off exists. |
| `release_operator_signing` | `allowed` (default), `forbidden` | Whether the operator who scanned a batch may sign it off in this department. `forbidden` needs someone else's signature even when the operator's role lets them sign. |
| `release_rationale` | text | Why this department needs (or doesn't need) a sign-off: its risk assessment, kept with the configuration. |
| `release_checklist` | list of prompts | Yes/no questions the signer answers, such as "Labeling and packaging correct?". |

Changes are audited (`release_policy`, keyed by department), so who set each department's requirement, when, and why is on record.

### Batch signature

An electronic signature on one batch at the step it is at. Append-only: there is no update or delete.

| Field | Meaning |
|---|---|
| `type` | `step_release`, `final_release` or `correction`. Set from the department's requirement; `correction` is requested explicitly. |
| `decision` | `approved` or `rejected`. |
| `capacity` | `supervisor` or `quality`: the authority the signer signed with. Derived from the department's requirement; in a `first_article` department the request says which of the two signatures this is. |
| `signer` | The account user who signed, identified by signing PIN (below). |
| `signer_name`, `signer_role` | Copied at signing, so the record still reads correctly after the user is renamed or changes role. |
| `production_step` | The step the batch was at; copied so the record survives step edits. |
| `checklist` | Each prompt as it read at signing, with the answer (`yes`, `no`, `not_applicable`). |
| `note` | Free text. Required on `rejected` and on `correction`. |
| `actor` | Who submitted it (the device or session), separate from the signer. |
| `signed_at` | Server time. |

A batch's release state at its step is its **latest** signature of type `step_release` / `final_release`:

- none → unreleased
- `approved` → released (on a `first_article` step: an `approved` signature in each capacity, on this batch or on another batch of the same production run at this step)
- `rejected` → on hold. The next scan is refused regardless of `release_enforcement` until a later `approved` signature (re-inspection) supersedes it. Withdrawing a release is a `rejected` signature recorded after the approval.

### Signing PIN

Each person sets their own PIN while signed in as themselves; nobody can read it. Operators who never sign in (no email) get a temporary PIN from an admin, which must be changed the first time it is used, at the station. An admin can also clear a PIN, which forces a new one to be set.

- 4–8 digits. Trivial PINs (one repeated digit, ascending or descending runs) are refused.
- Stored hashed (bcrypt) in auth-service, in its own table, never on a record or in a log.
- To sign, the signer is chosen (from a list of people eligible to sign that step) and enters their PIN, so a supervisor can sign on the operator's tablet without signing the tablet out. The pair of signer and PIN is what makes the signature unique to that person (Part 11 §11.200(a)).
- Five wrong PINs lock signing for that person for 15 minutes. Failed attempts and lockouts are audit events.
- This is the same PIN the forge.1 auth redesign plans for operators at devices (A3), so operator identification at scans reuses it (see open questions).

### Who may sign

- `supervisor` steps: the signer needs the `batch_releases:create` permission. A signer assigned to a department may sign only steps in that department; a signer with no department may sign any step.
- `quality` and `final` steps, and the quality signature on `first_article` steps: the signer needs `quality_releases:create`. The supervisor signature on `first_article` steps follows the `supervisor` rule.
- A quality role is normally given both permissions, so quality can also sign supervisor steps.
- When the department's `operator_signing` is `forbidden`, the operator recorded on the batch can't sign it. The dashboard doesn't offer this setting while operator check-in is hidden, because without a recorded operator there is nobody to exclude.
- Account users without an email always hold the scanner role, so supervisors and inspectors who sign need email accounts with a role granting the permission. Operators without email only check in, which needs no permission of their own.
- The caller submitting the signature (a station device or a signed-in user) needs `batches:create`, the same as scanning. The signer's eligibility is checked separately, against the signer, not the caller.

### Operators at stations

A scanning station gets `operator_check_in: optional | required` (separate from `operator_requirement`, which already means a material check). At such a station an operator checks in at the start of their shift: they pick their name and enter their signing PIN (the same PIN; anyone who works at a station sets one). The check-in lasts until they check out, another operator checks in, or 12 hours pass. Every scan at the station records the checked-in operator on the batch (`operator`), next to the station login that submitted it (`actor`). A scan at a `required` station with nobody checked in is refused with 409 `operator_check_in_required`. Check-ins and check-outs are audit events. The dashboard hides check-in for now: the station setting isn't offered, and the check-in bar shows only at a station already set to `required`.

## Behavior

### Scanning

When a move, merge or split consumes a batch sitting at a step with a requirement other than `none`:

| Batch state | `block` | `warn` |
|---|---|---|
| released | scan proceeds | scan proceeds |
| unreleased | 409 `batch_release_required` | scan proceeds |
| rejected | 409 `batch_on_hold` | 409 `batch_on_hold` |

A merge checks every input batch. Initialize scans are not gated: the batch has only just reached its first step.

### Final release

A `final_release` signature with `approved`:

1. Refused with 409 `batch_release_incomplete` if any batch in the batch's upstream lineage (`_batch_flow`) is unreleased or on hold at a step that requires release. This is what makes `warn` safe.
2. Refused with 422 if any checklist prompt is unanswered.
3. Assigns the finished-goods lot (phase 2): a `lot` row for the batch's item with lot number `<sequence>-<YYDDD>`. The sequence is per account and increasing. `YYDDD` is the two-digit year and the day of the year (the Julian date) of the release, in the account's time zone. The batch links to the lot. The lot's material genealogy is the existing issue → allocation → receipt → lot chain.

### Corrections

Data is stored as entered; a correction is approved by a supervisor or inspector. Undoing a scan is today's correction path.

- Undoing a scan whose batch sits at a step with a requirement other than `none` needs a `correction` signature on that batch, recorded after the batch was scanned. The signer must be eligible to sign that step, and the note gives the reason. Without one the undo is refused with 409 `correction_signature_required`.
- Undoing a scan whose batch has an `approved` release is refused with 409 `batch_released`. Withdraw the release first with a `rejected` signature, then record the correction.
- Signatures are never deleted with the batch. They keep the batch ID after an undo deletes the batch row, alongside the existing `deleted_record` snapshot.

### Batch record (electronic traveler)

The dashboard composes the device history record from standard resources:
- the batch flow, with each scan's station, step, time, quantities, seconds, waste and actor
- the batch's signatures
- the material lots consumed
- after phase 2, the released lot

A PDF export replaces the printed traveler for audits.

## API (internal at first; follows forge.1 conventions)

- `POST /v1/operations/batches/{id}/signatures`
  - Request: `{ type?: correction, decision, signer: { account_user_id, pin }, checklist?: [{ prompt, answer }], note? }`.
  - `type` defaults from the department's requirement.
  - Returns 201 `batch_signature`.
- `GET /v1/operations/batches/{id}/signatures`: the batch's signatures, newest first.
- `POST /v1/operations/scanning-stations/{id}/operator-check-ins { account_user_id, pin }` → 201 `operator_check_in`; `POST …/operator-check-ins/{id}/actions/check-out`; `GET /v1/operations/scanning-stations/{id}/operator-check-in` (the active one, or 404). History is in audit events.
- Batches gain `operator` (the checked-in account user when scanned).
- `GET /v1/operations/batches/{id}/eligible-signers`: who may sign the batch off in its department and in which capacities, leaving out the operator when the department forbids them signing.
- `GET` / `PUT /v1/operations/departments/{id}/release-policy` `{ requirement, enforcement?, operator_signing?, rationale?, checklist? }`: replaced as a whole.
- `PUT /v1/identity/users/me/signing-pin { pin }`. Admin-only: `POST /v1/identity/account-users/{id}/actions/set-temporary-signing-pin { pin }` and `…/actions/clear-signing-pin`. Using a temporary PIN returns 409 `signing_pin_change_required` until `POST /v1/identity/account-users/{id}/actions/change-signing-pin { current_pin, new_pin }` replaces it (callable from a station).
- New errors (409): `batch_release_required` (also for an incomplete lineage), `batch_on_hold`, `correction_signature_required`, `operator_check_in_required`, `signing_pin_not_set`, `signing_pin_change_required`; undoing a released batch is 409 `resource_conflict`. 403 `signing_pin_invalid`, 429 `signing_pin_locked`; an ineligible signer is 403 `insufficient_permissions`.

All are new endpoints or new optional fields, so nothing here needs an API version.

## Data

- `scanning_station.operator_check_in_code varchar(32) NOT NULL DEFAULT 'optional'`.
- `department`:
  - `release_requirement varchar(32) NOT NULL DEFAULT 'none'`
  - `release_enforcement varchar(32) NOT NULL DEFAULT 'block'`
  - `release_rationale text`
  - `release_checklist json`
- `batch_signature`:
  - columns: `id`, `account_id`, `batch_id` (no FK; outlives an undone batch), `production_step_id`, `type`, `decision`, `signer_account_user_id`, `signer_name`, `signer_role`, `checklist json`, `note`, `actor_id`, `actor_type`, `signed_at`, `created_at`
  - keys: `(account_id, batch_id, signed_at DESC, id DESC)`
- `batch.operator_account_user_id` (nullable): the checked-in operator when the batch was scanned.
- `operator_check_in`: `id`, `account_id`, `scanning_station_id`, `account_user_id`, `checked_in_at`, `checked_out_at`, `expires_at`; key `(account_id, scanning_station_id, checked_in_at DESC, id DESC)`.
- `signing_pin` (auth-service):
  - columns: `user_id` PK, `pin_hash`, `is_temporary`, `failed_attempts`, `locked_until`, `set_at`
- Permissions: `batch_releases` and `quality_releases`, added with a data migration, the same way `billing` was.
- The repository exposes insert and select for `batch_signature`, and nothing else.

## Phases

1. **Step releases and operators** (built). Step release config (including `first_article`), signatures, signing PIN, scan enforcement, the correction rule, operator check-in at `pin` stations, permissions, an audit event for every signature, check-in and failed PIN, and e2e tests.
2. **Final release and lot** (built). The signature carries `lot`; the batch lists it among its `lots` as type `release`; the date is read in the account's shift time zone, else UTC; the sequence is the `release_lot_number` counter. The final-release lineage check, lot assignment with the Julian date code, and the lot on the batch.
3. **Dashboard** (built). Also a printable batch record from the batch flow (production history, sign-offs, material lots, final release, document control footer), and a signer picker limited to eligible signers. Release policy card on the step page, operator check-in bar and sign-off dialog on the station screen (it opens when a scan or undo is refused for want of a signature), sign-off from batch cards and the batch flow, operator and sign-offs on each batch in the flow, the station's check-in setting, your own signing PIN under Account → Security, and temporary/clear PIN on a team member.
4. **Forward traceability.** Record the lot on pick and shipment lines, so a material lot can be traced to every customer shipment.

## Outside the software

ISO 13485 §4.1.6 requires the manufacturer to validate software used in its quality system for its intended use. A short validation package (intended use, requirements mapped to the tests that cover them) is worth providing with phase 3.

## Decided

- Operators check in once per shift with their PIN; per-scan PIN entry is not required.
- Set-up approval is `first_article`: per production run and step.
- Finished stock without a final release is not blocked from picking; the process (quality releases before packing) covers it.

## Open questions

None. Separation of duties is set per step (`operator_signing`).
