# TRADEOFFS.md

Three key architectural decisions, the alternatives considered, and why the chosen
option won for this specific problem.

---

## 1. RabbitMQ (durable buffer) vs. Kafka vs. direct-to-database write

**Chosen: RabbitMQ with publisher confirms and a durable queue.**

- **Direct-to-DB write** (API writes straight to ClickHouse/MySQL synchronously) was
  rejected because it couples user-facing latency to the storage engine's write speed.
  Under a large burst, ClickHouse's `INSERT` throughput is much better with batched
  writes than with thousands of tiny single-row inserts — a synchronous per-request
  write either becomes a bottleneck or requires client-side batching we don't control
  (submissions arrive from anonymous end-users one at a time).
- **Kafka** was considered because it also gives durable, ordered, replayable buffering
  and scales further than RabbitMQ for extreme throughput. It was rejected *for this
  task's scope* because: (a) MailerCloud's own stated infra uses RabbitMQ, so aligning
  reduces onboarding friction and answers "what would you actually deploy here"; (b)
  Kafka's operational overhead (ZooKeeper/KRaft, partition management) is
  disproportionate to the problem's actual throughput ceiling — form submissions, even
  at "large sudden bursts," are orders of magnitude lower volume than the log/event
  streams Kafka is built for; (c) RabbitMQ's per-message ack/confirm model maps more
  directly onto "did this specific submission durably land" than Kafka's offset-based
  model, which is a better fit semantically for a queue of discrete user actions rather
  than a continuous event stream.
- Trade-off accepted: RabbitMQ tops out lower than Kafka on raw sustained throughput and
  has weaker native replay/reprocessing tooling. If MailerCloud's real traffic ever
  exceeds what a well-tuned RabbitMQ cluster (mirrored/quorum queues) can absorb, this
  is the first component to swap for Kafka — the consumer interface is written against
  a small internal `Queue` interface for exactly that reason.

---

## 2. Split store (MySQL for definitions, ClickHouse for submissions) vs. single Postgres with JSONB for everything

**Chosen: MySQL (control plane) + ClickHouse (data plane), split by access pattern.**

- **Single Postgres with JSONB** was seriously considered — it's simpler to operate (one
  database, one connection pool, ACID transactions across forms and submissions,
  strong JSONB indexing with GIN). It was rejected because the two workloads have
  fundamentally different shapes: `forms`/`form_versions` are low-volume,
  read-after-write consistent, transactional (publish must be atomic). Submissions are
  extremely high-volume, append-only, almost never updated, and read in large
  aggregate/paginated/exported queries. A row-store like Postgres degrades on very
  large append-only tables (index bloat, vacuum overhead, slower full-table
  aggregate scans) well before a columnar engine like ClickHouse does at the same
  volume — and "large number of submissions, performant export" is an explicit
  requirement.
- Trade-off accepted: operating two databases instead of one is real operational cost
  (two backup strategies, two schemas to migrate, cross-store joins are impossible so
  we denormalize `tenant_id`/`form_version_id` into ClickHouse instead of joining back
  to MySQL). This is justified because MailerCloud's own JD explicitly names ClickHouse
  for analytics at billion-row scale — this is a workload they already operate at
  scale, so the operational cost is one they've already absorbed, and it's the right
  call technically regardless of that alignment.

---

## 3. Server-side validation "re-derived from stored schema" vs. a shared validation library compiled once at publish time

**Chosen: validate every submission by re-parsing `form_versions.schema_json` at
request time (with an in-process cache keyed by `version_id`, not per-request DB hits).**

- **Compile-once-at-publish** (generate a versioned validator function/bytecode once
  when the form is published, store or cache that compiled validator, and just invoke
  it per submission) is faster per-request in theory. It was not fully rejected — it's
  effectively what the in-memory cache achieves — but a *literal* compiled-artifact
  approach (e.g., generating and storing actual Go code per form version) was rejected
  as unnecessary complexity: it adds a build/codegen step to the publish flow, and
  introduces a new failure mode (codegen fails, or drifts from a manually-maintained
  validation library) for a marginal performance gain over "parse JSON once, cache the
  parsed validator struct in memory, keyed by immutable version_id."
- Because `form_versions.schema_json` is immutable once published, caching the parsed
  validator by `version_id` is always safe to cache indefinitely (no invalidation
  problem) — giving nearly all the performance benefit of a compiled approach with far
  less operational complexity.
- Trade-off accepted: the very first submission against a brand-new version pays a
  one-time JSON-parse cost before the cache is warm; at MailerCloud's scale this is
  negligible compared to the correctness benefit of always validating against the
  literal, single source of truth for that version rather than a possibly-stale
  generated artifact.
