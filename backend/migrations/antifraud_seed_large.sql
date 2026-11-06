BEGIN;

ALTER TABLE antifraud_check DISABLE TRIGGER USER;
ALTER TABLE rule_evaluation DISABLE TRIGGER USER;
ALTER TABLE review_case DISABLE TRIGGER USER;
ALTER TABLE outbox_event DISABLE TRIGGER USER;

INSERT INTO app_user (id)
SELECT 'bulk-user-' || n FROM generate_series(1, 10000) AS n;
INSERT INTO app_user (id) VALUES ('bulk-analyst');

INSERT INTO bank_account (id, owner_id)
SELECT 100000 + n, 'bulk-user-' || ((n - 1) / 2 + 1)
FROM generate_series(1, 20000) AS n;

INSERT INTO transfer_attempt (id, from_account_id, to_account_id, amount,
    device, ip_address, created_at, received_at)
SELECT 1000000 + n, 100001 + (n - 1) % 20000, 100001 + n % 20000,
       (100 + n % 9900)::numeric(19,2), 'bulk-device-' || (n % 2000),
       inet '10.0.0.1' + (n % 65535),
       timestamptz '2026-11-01 00:00+03' - n * interval '1 minute',
       timestamptz '2026-11-01 00:00+03' - n * interval '1 minute' + interval '1 second'
FROM generate_series(1, 100000) AS n;

INSERT INTO antifraud_check (id, transfer_id, risk_score, automatic_decision, checked_at)
SELECT 1000000 + n, 1000000 + n, 0, 'APPROVED',
       timestamptz '2026-11-01 00:00+03' - n * interval '1 minute' + interval '2 seconds'
FROM generate_series(1, 100000) AS n;

INSERT INTO rule_evaluation (check_id, rule_id, status, score, reason)
SELECT 1000000 + n, rule_id, 'NOT_TRIGGERED', 0, 'Массовые данные'
FROM generate_series(1, 100000) AS n CROSS JOIN generate_series(1, 8) AS r(rule_id);

INSERT INTO login_event (user_id, ip_address, occurred_at)
SELECT 'bulk-user-' || (1 + (n - 1) % 10000), inet '10.0.0.1' + (n % 65535),
       timestamptz '2026-11-01 00:00+03' - n * interval '1 minute'
FROM generate_series(1, 100000) AS n;

INSERT INTO user_device (user_id, device, first_seen_at, last_seen_at, use_count)
SELECT 'bulk-user-' || ((n - 1) / 2 + 1), 'bulk-device-' || ((n - 1) % 2000),
       timestamptz '2026-01-01 00:00+03', timestamptz '2026-11-01 00:00+03', 1
FROM generate_series(1, 20000) AS n;

INSERT INTO review_case (check_id, analyst_id, status, decision, created_at, decided_at)
SELECT 1000000 + n * 100, 'bulk-analyst',
       CASE WHEN n % 100 = 0 THEN 'OPEN' ELSE 'CLOSED' END,
       CASE WHEN n % 100 <> 0 THEN 'APPROVED' END,
       timestamptz '2026-11-01 00:00+03' - n * interval '100 minutes' + interval '2 seconds',
       CASE WHEN n % 100 <> 0 THEN
           timestamptz '2026-11-01 00:00+03' - n * interval '100 minutes' + interval '1 minute' END
FROM generate_series(1, 1000) AS n;

INSERT INTO outbox_event (event_id, check_id, payload, created_at, published,
    attempt_count, next_attempt_at, published_at)
SELECT uuidv7(), 1000000 + n, '{}'::jsonb,
       timestamptz '2026-11-01 00:00+03' - n * interval '1 minute', n % 100 <> 0, 1,
       now() + CASE WHEN n % 200 = 0 THEN interval '5 minutes' ELSE interval '-1 minute' END,
       CASE WHEN n % 100 <> 0 THEN
           timestamptz '2026-11-01 00:00+03' - n * interval '1 minute' + interval '1 second' END
FROM generate_series(1, 100000) AS n;

INSERT INTO audit_log (occurred_at, actor_type, action, check_id, details)
SELECT timestamptz '2026-11-01 00:00+03' - n * interval '1 minute',
       'SYSTEM', 'CHECK_COMPLETED', 1000000 + n, '{}'::jsonb
FROM generate_series(1, 100000) AS n;

INSERT INTO inbox_event (event_id, transfer_id, occurred_at, received_at,
    payload, applied, processed_at)
SELECT uuidv7(), id, created_at, created_at + interval '1 second',
       '{}'::jsonb, true, created_at + interval '2 seconds'
FROM transfer_attempt WHERE id BETWEEN 1000001 AND 1100000;

SELECT setval('user_device_id_seq', (SELECT max(id) FROM user_device));
SELECT setval('login_event_id_seq', (SELECT max(id) FROM login_event));
SELECT setval('antifraud_check_id_seq', (SELECT max(id) FROM antifraud_check));
SELECT setval('review_case_id_seq', (SELECT max(id) FROM review_case));
SELECT setval('audit_log_id_seq', (SELECT max(id) FROM audit_log));

ALTER TABLE antifraud_check ENABLE TRIGGER USER;
ALTER TABLE rule_evaluation ENABLE TRIGGER USER;
ALTER TABLE review_case ENABLE TRIGGER USER;
ALTER TABLE outbox_event ENABLE TRIGGER USER;
COMMIT;

ANALYZE app_user;
ANALYZE bank_account;
ANALYZE user_device;
ANALYZE login_event;
ANALYZE transfer_attempt;
ANALYZE antifraud_check;
ANALYZE rule_evaluation;
ANALYZE review_case;
ANALYZE outbox_event;
ANALYZE audit_log;
ANALYZE inbox_event;
