CREATE INDEX idx_raw_received_at ON raw_payloads (received_at);
CREATE INDEX idx_raw_gateway     ON raw_payloads (gateway);
CREATE INDEX idx_raw_corr        ON raw_payloads (correlation_id);

CREATE INDEX idx_orders_lead_id  ON orders (lead_id);

CREATE INDEX idx_le_correlation  ON lead_events (correlation_id);
CREATE INDEX idx_le_created_at   ON lead_events (created_at);

CREATE INDEX idx_ds_channel_status ON distribution_status (channel, status);
CREATE INDEX idx_ds_created_at     ON distribution_status (created_at);
CREATE INDEX idx_ds_delivered_at   ON distribution_status (delivered_at);
CREATE INDEX idx_ds_order_channel  ON distribution_status (order_id, channel);

CREATE INDEX idx_ddl_origin     ON lead_dead_letter (origin);
CREATE INDEX idx_ddl_created_at ON lead_dead_letter (created_at);
