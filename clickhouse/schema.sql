-- Production submissions store (billion-row scale target).
-- Consumer writes here when USE_CLICKHOUSE=1 and SUBMISSIONS_DSN points at
-- a ClickHouse-compatible driver instead of MySQL. Kept as a separate SQL
-- file since the working slice's Go driver targets MySQL by default (see
-- TRADEOFFS.md #2 for why ClickHouse wasn't wired into the timeboxed build).

CREATE TABLE IF NOT EXISTS submissions (
    submission_id UUID,
    tenant_id UInt64,
    form_id UInt64,
    form_version_id UInt64,
    submitted_at DateTime64(3),
    payload String,       -- JSON blob, field_key -> value
    ip_hash String,
    status Enum8('accepted' = 1, 'flagged_spam' = 2),
    inserted_at DateTime64(3) DEFAULT now64()
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(submitted_at)
ORDER BY (tenant_id, form_id, submitted_at)
TTL submitted_at + INTERVAL 5 YEAR DELETE; -- example retention policy, tune per plan

-- Example materialized view to promote a hot filtered field (e.g. "email")
-- to its own column for a specific high-traffic customer, without touching
-- the base table's dynamic-schema design. Designed, not built.
-- CREATE MATERIALIZED VIEW submissions_email_mv ...
