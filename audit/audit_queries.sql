-- 1) Lag médio (em segundos) entre transaction_time do gateway e delivered_at em SMS, agrupado por gateway, nas últimas 24h

SELECT
    o.gateway,
    COUNT(*) AS total_delivered,
    AVG(
        TIMESTAMPDIFF(SECOND, le.transaction_time, ds.delivered_at)
    ) AS avg_lag_seconds,
    MIN(
        TIMESTAMPDIFF(SECOND, le.transaction_time, ds.delivered_at)
    ) AS min_lag_seconds,
    MAX(
        TIMESTAMPDIFF(SECOND, le.transaction_time, ds.delivered_at)
    ) AS max_lag_seconds
FROM
    distribution_status ds
    JOIN orders o ON o.id = ds.order_id
    JOIN lead_events le ON le.order_id = o.id
    AND le.event = 'order.approved'
WHERE
    ds.channel = 'SMS'
    AND ds.status = 'delivered'
    AND ds.delivered_at >= UTC_TIMESTAMP() - INTERVAL 24 HOUR
GROUP BY
    o.gateway;

-- 2) Quantos leads em pending há mais de 5 minutos? Lista order_id, canal e idade do pendente em segundos

SELECT
    ds.order_id,
    ds.channel,
    TIMESTAMPDIFF(SECOND, ds.created_at, UTC_TIMESTAMP()) AS pending_age_seconds
FROM
    distribution_status ds
WHERE
    ds.status = 'pending'
    AND ds.created_at <= UTC_TIMESTAMP() - INTERVAL 5 MINUTE
ORDER BY
    pending_age_seconds DESC;

-- 3) Taxa de sucesso de SMS por produto, por hora, nas últimas 6h

SELECT
    o.product_id,
    o.product_name,
    DATE_FORMAT(ds.created_at, '%Y-%m-%d %H:00') AS hour_bucket,
    SUM(ds.status = 'delivered') AS delivered,
    SUM(ds.status = 'failed') AS failed,
    COUNT(*) AS total,
    ROUND(
        100.0 * SUM(ds.status = 'delivered') / NULLIF(COUNT(*), 0),
        2
    ) AS success_rate_pct
FROM
    distribution_status ds
    JOIN orders o ON o.id = ds.order_id
WHERE
    ds.channel = 'SMS'
    AND ds.status IN ('delivered', 'failed')
    AND ds.created_at >= UTC_TIMESTAMP() - INTERVAL 6 HOUR
GROUP BY
    o.product_id,
    o.product_name,
    hour_bucket
ORDER BY
    hour_bucket,
    o.product_id;

-- 4) Quantos leads em DLQ por motivo, nas últimas 24h

SELECT
    origin,
    error_reason,
    COUNT(*) AS total
FROM
    lead_dead_letter
WHERE
    created_at >= UTC_TIMESTAMP() - INTERVAL 24 HOUR
GROUP BY
    origin,
    error_reason
ORDER BY
    total DESC;

-- 5) Reconciliação: total de leads aprovados em lead_events vs total delivered em SMS, por dia, últimos 7 dias. Gap absoluto e percentual.

SELECT
    DATE(le.persisted_at) AS day,
    COUNT(DISTINCT le.id) AS approved_events,
    SUM(
        CASE
            WHEN ds.status = 'delivered' THEN 1
            ELSE 0
        END
    ) AS sms_delivered,
    COUNT(DISTINCT le.id) - SUM(
        CASE
            WHEN ds.status = 'delivered' THEN 1
            ELSE 0
        END
    ) AS gap_abs,
    ROUND(
        100.0 * SUM(
            CASE
                WHEN ds.status = 'delivered' THEN 1
                ELSE 0
            END
        ) / NULLIF(COUNT(DISTINCT le.id), 0),
        2
    ) AS coverage_pct
FROM
    lead_events le
    LEFT JOIN distribution_status ds ON ds.order_id = le.order_id
    AND ds.channel = 'SMS'
WHERE
    le.event = 'order.approved'
    AND le.persisted_at >= UTC_TIMESTAMP() - INTERVAL 7 DAY
GROUP BY
    DATE(le.persisted_at)
ORDER BY
    day DESC;