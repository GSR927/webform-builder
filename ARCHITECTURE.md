# ARCHITECTURE.md — Webform Builder & Submission Platform

## 1. Overview

This system lets tenants (customer accounts) design forms, publish immutable versions of
them, serve those forms publicly, and collect submissions durably at scale. The design
optimizes for one truth: **the write path (public submission) must never lose data and
must stay fast under bursty load**, even if it means the read/analytics path is
eventually consistent.

## 2. High-Level Components

```
                                  ┌────────────────────┐
                                  │   Vue 3 Builder UI  │  (authenticated, tenant admin)
                                  └─────────┬───────────┘
                                            │ REST/GraphQL (draft CRUD, publish)
                                            ▼
   ┌───────────────┐   HAProxy    ┌──────────────────────┐        ┌───────────────┐
   │  Public Form   │◄────────────┤   Go API (stateless)  │◄──────►│    MySQL      │
   │  (Vue render)  │   (LB, TLS) │  forms / publish /    │        │ forms,        │
   │  + client val. │             │  public-serve /       │        │ form_versions,│
   └──────┬─────────┘             │  submit endpoints     │        │ tenants       │
          │ GET published schema  └──────────┬─────────────┘        └───────────────┘
          │ (CDN/Redis cached,               │
          │  immutable per version)          │ POST submission (validated)
          ▼                                  ▼
   ┌────────────────┐               ┌──────────────────────┐
   │  Redis          │               │  RabbitMQ (durable,   │
   │  - rate limits  │               │  publisher-confirmed) │
   │  - form cache   │               └──────────┬─────────────┘
   └────────────────┘                            │ batched consume
                                                  ▼
                                       ┌──────────────────────┐
                                       │  Go Consumer Pool     │
                                       │  (idempotent insert,  │
                                       │   retry + DLQ)        │
                                       └──────────┬─────────────┘
                                                  ▼
                                       ┌──────────────────────┐
                                       │     ClickHouse        │
                                       │  submissions (append- │
                                       │  only, partitioned by │
                                       │  tenant_id)           │
                                       └──────────────────────┘
```

## 3. End-to-End Request Flow

### 3a. Build & publish (low volume, admin-authenticated)
1. Customer edits a form in the Vue builder. Every keystroke/save updates
   `forms.draft_schema_json` in MySQL. This never touches the live form.
2. On "Publish," the API snapshots `draft_schema_json` into a new, immutable row in
   `form_versions` (`version_number` incremented, `schema_json` frozen, `published_at`
   set). `forms.current_version_id` is updated atomically in one transaction.
3. Because published versions are immutable and referenced by ID, editing the draft
   afterward can never affect a version already serving traffic or already referenced
   by past submissions.

### 3b. Public serve (high volume, anonymous, read-heavy)
1. `GET /f/:formId` resolves `forms.current_version_id` → fetches
   `form_versions.schema_json`.
2. Response is cached aggressively (CDN + Redis) keyed by `version_id`, not `form_id`,
   with a long TTL — since a version is immutable, cache invalidation is a non-issue:
   a new publish simply produces a new cache key. Only the *pointer* (`current_version_id`)
   changes and that lookup is cheap and cacheable for seconds, not the payload itself.
3. Vue renders fields dynamically from `schema_json` (type, validators, conditional
   `showIf` rules) and runs client-side validation for UX — but this is never trusted.

### 3c. Public submission (the critical path)
This is the path the whole architecture is built around, because it is public,
anonymous, bursty, and must never silently drop data.

1. `POST /f/:formId/submit` hits Go API behind HAProxy (horizontally scaled, stateless
   pods/processes — HAProxy round-robins/least-conn across them).
2. Rate limiting: Redis token bucket keyed by `(tenant_id, form_id, ip_hash)` — rejects
   abusive volume with `429` *before* doing any expensive work. Buckets are sized per
   tenant plan so one tenant's burst can't consume another's budget.
3. Server-side validation is **re-derived from `form_versions.schema_json`** — the same
   version the client rendered (we pass `version_id` in the submit payload so we know
   exactly which schema to validate against). We never trust client-declared types or
   required flags.
4. Basic spam defenses: honeypot field check, submission-rate heuristics, optional
   CAPTCHA token verification hook.
5. **Durability commit point**: once validated, the API publishes the submission to
   RabbitMQ with **publisher confirms** (`channel.Confirm` mode) on a durable queue with
   a durable exchange. Only after RabbitMQ acks the message as persisted does the API
   return `202 Accepted` to the end-user. This is the guarantee that answers "a
   submission must never be lost between click and durable storage" — durability is
   defined as "on RabbitMQ's disk," not "in ClickHouse," decoupling user-facing latency
   from downstream storage speed.
6. If RabbitMQ itself is unreachable, the API falls back to writing an emergency record
   directly to a small MySQL `submissions_inbox` table (same transaction guarantees as
   any other MySQL write) so the request path has a second durable option before ever
   returning an error to the end-user. This is the "even when a downstream component is
   slow or unavailable" case.
7. A pool of Go consumers reads from RabbitMQ (prefetch-limited, batched) and performs
   idempotent inserts into ClickHouse, keyed by a submission UUID generated by the API
   (not by the consumer), so redelivery after a crash never double-counts. Failures are
   retried with backoff; poison messages go to a dead-letter queue for manual replay.

## 4. Data Model

### MySQL (control plane — low volume, high consistency needs)
```sql
tenants(id, name, plan, created_at)

forms(
  id, tenant_id, name, status ENUM('draft','published','archived'),
  draft_schema_json JSON,
  current_version_id BIGINT NULL,
  created_at, updated_at
)

form_versions(
  id, form_id, tenant_id, version_number,
  schema_json JSON,          -- frozen field defs: type, validators, required, help_text, showIf
  published_at, published_by
)  -- INSERT-only, never UPDATE/DELETE

submissions_inbox(              -- emergency fallback path only
  id, tenant_id, form_id, form_version_id, payload_json, received_at, status
)
```

`schema_json` shape (per field):
```json
{
  "key": "email",
  "type": "email",
  "label": "Work email",
  "required": true,
  "help_text": "We'll never spam you",
  "validators": {"maxLength": 120},
  "showIf": {"field": "role", "equals": "marketer"}
}
```

Storing the schema as JSON inside MySQL (rather than a fully normalized field table) is
deliberate: field sets are arbitrary and nested (conditional rules reference other
fields), so a rigid relational schema would need constant migrations. MySQL's JSON type
still lets us index/query top-level fields like `tenant_id` and `status` relationally,
which is where our actual query patterns are (list forms per tenant), while the
schema's internal shape stays flexible.

### ClickHouse (data plane — write-heavy, append-only, analytical reads)
```sql
CREATE TABLE submissions (
    submission_id UUID,
    tenant_id UInt64,
    form_id UInt64,
    form_version_id UInt64,
    submitted_at DateTime64(3),
    payload String,        -- JSON blob of field_key -> value
    ip_hash String,
    status Enum8('accepted' = 1, 'flagged_spam' = 2),
    inserted_at DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree
PARTITION BY toYYYYMM(submitted_at)
ORDER BY (tenant_id, form_id, submitted_at)
```
Why ClickHouse for submissions specifically:
- Submissions are write-once, append-only, and read mostly in aggregate (paginated
  lists, filters, exports) — ClickHouse's columnar, MergeTree storage is built exactly
  for this, and scales to billions of rows with predictable performance (this mirrors
  MailerCloud's own stated use of ClickHouse for analytics at that scale).
- Ordering by `(tenant_id, form_id, submitted_at)` makes the two most common queries —
  "submissions for form X, paginated/filtered" and "export all submissions for form X"
  — sequential scans instead of random I/O, and doubles as tenant isolation on the
  storage layer (one tenant's data is physically clustered, not scattered).
- `payload` stays a JSON string rather than one column per field, because the field set
  is dynamic per form/version — a fixed columnar schema per-form would require DDL on
  every publish, which doesn't scale to "many tenants, many forms." Frequently-filtered
  fields could later be promoted to materialized columns via a materialized view if a
  specific customer's query pattern demands it — see TRADEOFFS.md.

## 5. Meeting Each Requirement

**Bursty public load** — stateless Go API instances behind HAProxy scale horizontally;
the expensive/stateful part (durable persistence) is handed to RabbitMQ, which buffers
bursts far beyond what a synchronous DB write could absorb. The API's own work per
request is O(1): rate-limit check, validate against an already-fetched schema, publish
to queue.

**Multi-tenancy & isolation** — every row everywhere carries `tenant_id`; rate limits
and queue consumer fairness are enforced per tenant; ClickHouse's ordering key clusters
tenant data physically, so one tenant's huge export doesn't force a full-table scan
that touches others' data; consumer pool can apply per-tenant concurrency caps so no
single tenant's backlog starves others.

**Correctness under failure** — durability is committed at the RabbitMQ ack (with a
MySQL inbox fallback), *before* the user sees success — not at the ClickHouse insert.
Consumers are idempotent by submission UUID, so redelivery after a crash is safe.

**Data integrity across edits** — publishing never mutates a `form_versions` row; every
submission is permanently tied to `form_version_id`, so a schema change can never
reinterpret old data.

**Security (XSS/injection)** — field definitions are structured JSON, never raw
HTML/JS; all customer-supplied strings (labels, help text) are output-encoded at render
time; server-side validation is regenerated from the schema on every submit, never
trusting client-declared rules; CSP headers on the public embed; submitted values are
stored as data (JSON), never interpolated into HTML/SQL directly (parameterized
inserts).

**Dynamic-schema storage** — JSON-in-MySQL for the low-volume, consistency-sensitive
control plane; JSON-payload-in-ClickHouse for the high-volume, append-only data plane.
Each store is used for what it's actually good at.

## 6. Built vs. Designed

**Built (working slice):**
- Go API: form create/update draft, publish (creates immutable version), public GET
  form-by-version, public POST submission with schema-derived server-side validation.
- RabbitMQ publish with confirms on submit; a consumer that inserts into ClickHouse
  (falls back to a local SQLite/MySQL table if ClickHouse isn't available in the dev
  environment, clearly flagged in code).
- MySQL schema + migrations for `forms`/`form_versions`.
- Minimal Vue 3 renderer for 3 field types + one conditional-visibility rule.
- Unit tests: schema-derived server-side validation; submission-integrity across a
  version change.
- k6 load generator hitting the submit endpoint with a burst profile.

**Designed but not built:**
- HAProxy config and horizontal autoscaling of API/consumer pods (documented, not
  wired up — no multi-node infra in the time box).
- CDN layer in front of public form serving.
- Per-tenant plan-based rate-limit tiers (only a single flat limit is implemented).
- CAPTCHA/third-party spam-scoring integration (only a honeypot field is implemented).
- Dead-letter queue replay tooling and consumer autoscaling based on queue depth.
- Materialized-view promotion of hot fields in ClickHouse for filtered exports.
- Full form builder UI (drag/drop) — only a renderer + minimal JSON-driven builder stub.
