CREATE TABLE form_versions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    form_id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL,
    version_number INT NOT NULL,
    schema_json JSON NOT NULL,
    published_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_form_version (form_id, version_number),
    INDEX idx_form (form_id),
    CONSTRAINT fk_versions_form FOREIGN KEY (form_id) REFERENCES forms(id)
    -- Deliberately no UPDATE/DELETE path exposed at the application layer:
    -- rows in this table are treated as append-only/immutable.
);

ALTER TABLE forms
    ADD CONSTRAINT fk_forms_current_version
    FOREIGN KEY (current_version_id) REFERENCES form_versions(id);

-- Fallback durability path used only if RabbitMQ publish/confirm fails or
-- times out on the hot submission path. A background sweeper (designed,
-- not built in the working slice) would drain this into the queue/CH.
CREATE TABLE submissions_inbox (
    id CHAR(36) PRIMARY KEY,           -- same UUID the API generated
    tenant_id BIGINT NOT NULL,
    form_id BIGINT NOT NULL,
    form_version_id BIGINT NOT NULL,
    payload_json JSON NOT NULL,
    received_at DATETIME NOT NULL,
    status ENUM('pending_consume', 'consumed') NOT NULL DEFAULT 'pending_consume',
    INDEX idx_status (status)
);

-- Local dev / no-ClickHouse fallback target used by the consumer when
-- USE_CLICKHOUSE is not set to "1" (see backend/consumer/store.go).
CREATE TABLE submissions (
    id CHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    form_id BIGINT NOT NULL,
    form_version_id BIGINT NOT NULL,
    payload_json JSON NOT NULL,
    ip_hash CHAR(64) NOT NULL,
    submitted_at DATETIME NOT NULL,
    status ENUM('accepted', 'flagged_spam') NOT NULL DEFAULT 'accepted',
    INDEX idx_form_time (form_id, submitted_at)
);
