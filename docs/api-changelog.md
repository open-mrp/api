# API changelog

Changes to the public API, by the `OpenMRP-Version` that introduces them. A client pinned to an older supported version keeps the shape described for that version; see [API versioning](patterns/api-versioning-patterns.md).

## 1.0.forge-preview.6

### Breaking changes

Clients pinned to 1.0.forge-preview.5 or earlier keep the previous behavior.

- **Registration flow option lists.** `PATCH /v1/sales/registration-flows/{id}` replaces an option list (`customer_group_ids`, `payment_term_ids`, `shipping_term_ids`) whenever it is sent, and `[]` removes every option. An omitted list is left unchanged. The `has_customer_group_ids`, `has_payment_term_ids` and `has_shipping_term_ids` flags are gone.
  - Older versions: a list is replaced only when its `has_<list>` flag is `true`, and is ignored otherwise.
  - Migration: drop the flags, and send a list only when you mean to replace it.
