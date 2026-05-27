CREATE TABLE IF NOT EXISTS raw_payloads (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    correlation_id VARCHAR(36) NOT NULL,
    gateway VARCHAR(50) NOT NULL,
    received_at DATETIME NOT NULL,
    headers TEXT,
    body_raw MEDIUMTEXT,
    body_decrypted MEDIUMTEXT,
    created_at DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP())
);

CREATE TABLE IF NOT EXISTS leads (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(50),
    country CHAR(2),
    created_at DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP()),
    UNIQUE KEY uq_leads_email (email)
);

CREATE TABLE IF NOT EXISTS orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    lead_id BIGINT NOT NULL,
    gateway VARCHAR(50) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL,
    transaction_time DATETIME NOT NULL,
    product_id VARCHAR(100),
    product_name VARCHAR(255),
    product_niche VARCHAR(100),
    quantity INT,
    amount_usd DECIMAL(10, 2),
    payment_method VARCHAR(50),
    payment_status VARCHAR(50),
    created_at DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP()),
    UNIQUE KEY uq_orders_gateway_txid (gateway, transaction_id),
    CONSTRAINT fk_orders_lead FOREIGN KEY (lead_id) REFERENCES leads (id)
);

CREATE TABLE IF NOT EXISTS lead_events (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    correlation_id VARCHAR(36) NOT NULL,
    event VARCHAR(100) NOT NULL,
    transaction_time DATETIME NOT NULL,
    persisted_at DATETIME NOT NULL,
    lag_seconds INT,
    created_at DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP()),
    UNIQUE KEY uq_lead_events_order_event (order_id, event),
    CONSTRAINT fk_lead_events_order FOREIGN KEY (order_id) REFERENCES orders (id)
);

CREATE TABLE IF NOT EXISTS distribution_status (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    channel ENUM ('SMS', 'EMAIL', 'CALL_CENTER', 'WHATSAPP') NOT NULL,
    status ENUM ('pending', 'delivered', 'failed') NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP()),
    delivered_at DATETIME,
    lag_db_channel_seconds INT,
    UNIQUE KEY uq_dist_order_channel (order_id, channel),
    CONSTRAINT fk_dist_order FOREIGN KEY (order_id) REFERENCES orders (id)
);

CREATE TABLE IF NOT EXISTS lead_dead_letter (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    correlation_id VARCHAR(36),
    origin VARCHAR(100) NOT NULL,
    payload MEDIUMTEXT,
    error_reason TEXT,
    created_at DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP())
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    transaction_id VARCHAR(255) NOT NULL,
    event VARCHAR(100) NOT NULL,
    gateway VARCHAR(50) NOT NULL,
    correlation_id VARCHAR(36) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (UTC_TIMESTAMP()),
    PRIMARY KEY (transaction_id, event)
);