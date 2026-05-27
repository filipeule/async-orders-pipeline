DROP PROCEDURE IF EXISTS sp_insert_lead;

DELIMITER //

CREATE PROCEDURE sp_insert_lead(
    IN  p_correlation_id    VARCHAR(36),
    IN  p_gateway           VARCHAR(50),
    IN  p_email             VARCHAR(255),
    IN  p_first_name        VARCHAR(100),
    IN  p_last_name         VARCHAR(100),
    IN  p_phone             VARCHAR(50),
    IN  p_country           CHAR(2),
    IN  p_transaction_id    VARCHAR(255),
    IN  p_transaction_time  DATETIME,
    IN  p_product_id        VARCHAR(100),
    IN  p_product_name      VARCHAR(255),
    IN  p_product_niche     VARCHAR(100),
    IN  p_quantity          INT,
    IN  p_amount_usd        DECIMAL(10,2),
    IN  p_payment_method    VARCHAR(50),
    IN  p_payment_status    VARCHAR(50),
    IN  p_event             VARCHAR(100),
    OUT p_order_id          BIGINT
)
BEGIN
    DECLARE v_lead_id  BIGINT;
    DECLARE v_lag      INT;
    DECLARE v_inserted INT DEFAULT 0;

    START TRANSACTION;

    INSERT INTO leads (email, first_name, last_name, phone, country)
    VALUES (p_email, p_first_name, p_last_name, p_phone, p_country)
    ON DUPLICATE KEY UPDATE
        first_name = VALUES(first_name),
        last_name  = VALUES(last_name),
        phone      = VALUES(phone),
        country    = VALUES(country);

    SELECT id INTO v_lead_id FROM leads WHERE email = p_email;

    INSERT IGNORE INTO orders
        (lead_id, gateway, transaction_id, transaction_time,
         product_id, product_name, product_niche, quantity,
         amount_usd, payment_method, payment_status)
    VALUES
        (v_lead_id, p_gateway, p_transaction_id, p_transaction_time,
         p_product_id, p_product_name, p_product_niche, p_quantity,
         p_amount_usd, p_payment_method, p_payment_status);

    SELECT id INTO p_order_id
    FROM orders
    WHERE gateway = p_gateway AND transaction_id = p_transaction_id;

    SET v_lag = TIMESTAMPDIFF(SECOND, p_transaction_time, UTC_TIMESTAMP());

    INSERT IGNORE INTO lead_events
        (order_id, correlation_id, event, transaction_time, persisted_at, lag_seconds)
    VALUES
        (p_order_id, p_correlation_id, p_event, p_transaction_time, UTC_TIMESTAMP(), v_lag);

    SET v_inserted = ROW_COUNT();

    IF v_inserted > 0 THEN
        INSERT INTO distribution_status (order_id, channel, status) VALUES
            (p_order_id, 'SMS',         'pending'),
            (p_order_id, 'EMAIL',       'pending'),
            (p_order_id, 'CALL_CENTER', 'pending'),
            (p_order_id, 'WHATSAPP',    'pending');
    END IF;

    COMMIT;
END //

DELIMITER ;
