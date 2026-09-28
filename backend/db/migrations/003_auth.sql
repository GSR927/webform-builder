CREATE TABLE IF NOT EXISTS users (
 id BIGINT AUTO_INCREMENT PRIMARY KEY,
 tenant_id BIGINT NOT NULL,
 name VARCHAR(120) NOT NULL,
 email VARCHAR(254) NOT NULL UNIQUE,
 password_hash VARCHAR(255) NOT NULL,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 INDEX idx_users_tenant(tenant_id),
 FOREIGN KEY(tenant_id) REFERENCES tenants(id)
);
CREATE TABLE IF NOT EXISTS auth_sessions (
 token_hash BINARY(32) PRIMARY KEY,
 user_id BIGINT NOT NULL,
 tenant_id BIGINT NOT NULL,
 csrf_hash BINARY(32) NOT NULL,
 expires_at DATETIME NOT NULL,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 INDEX idx_session_expiry(expires_at),
 FOREIGN KEY(user_id) REFERENCES users(id),
 FOREIGN KEY(tenant_id) REFERENCES tenants(id)
);
