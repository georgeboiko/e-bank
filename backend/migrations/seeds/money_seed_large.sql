BEGIN;

INSERT INTO transfer_limits (daily_limit, monthly_limit)
VALUES (100000.00, 1000000.00)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE account DISABLE TRIGGER USER;
ALTER TABLE transfer DISABLE TRIGGER USER;
ALTER TABLE ledger_entry DISABLE TRIGGER USER;
ALTER TABLE external_payment DISABLE TRIGGER USER;

INSERT INTO account (id, owner_id, type, balance, opened_at)
SELECT 100000 + n, 'bulk-user-' || ((n - 1) / 2 + 1), 'CLIENT', 1000000,
       timestamptz '2026-01-01 00:00+03'
FROM generate_series(1, 20000) AS n;

INSERT INTO transfer (id, from_account_id, to_account_id, amount, state,
    idempotency_key, reject_cause, device_id, ip_address, created_at, updated_at, finished_at)
SELECT 1000000 + n, 100001 + (n - 1) % 20000, 100001 + n % 20000, 100 + n % 9900,
       CASE n % 1000 WHEN 0 THEN 'COMPLETING' WHEN 1 THEN 'RELEASING_FUNDS' ELSE 'COMPLETED' END,
       uuidv7(), CASE WHEN n % 1000 = 1 THEN 'ANTIFRAUD_REJECTED' END,
       'bulk-device-' || n % 2000, inet '10.0.0.1' + n % 65535,
       timestamptz '2026-11-01 00:00+03' - n * interval '1 minute',
       timestamptz '2026-11-01 00:00+03' - n * interval '1 minute' + interval '1 second',
       CASE WHEN n % 1000 NOT IN (0, 1) THEN
           timestamptz '2026-11-01 00:00+03' - n * interval '1 minute' + interval '1 second' END
FROM generate_series(1, 100000) AS n;

INSERT INTO ledger_entry (transfer_id, account_id, amount, created_at)
SELECT id, from_account_id, -amount, created_at + interval '1 second'
FROM transfer WHERE id BETWEEN 1000001 AND 1100000;

INSERT INTO ledger_entry (transfer_id, account_id, amount, created_at)
SELECT id, to_account_id, amount, created_at + interval '1 second'
FROM transfer WHERE id BETWEEN 1000001 AND 1100000;

WITH events AS MATERIALIZED (
    SELECT uuidv7() AS event_id, t.*, sender.owner_id AS sender_id, recipient.owner_id AS recipient_id
    FROM transfer t
    JOIN account sender ON sender.id = t.from_account_id
    JOIN account recipient ON recipient.id = t.to_account_id
    WHERE t.id BETWEEN 1000001 AND 1100000
)
INSERT INTO outbox_event (event_id, transfer_id, payload, created_at, published,
    next_attempt_at, published_at)
SELECT event_id, id, jsonb_build_object(
           'eventId', event_id, 'eventType', 'FraudCheckRequested',
           'occurredAt', created_at,
           'data', jsonb_build_object(
               'transferId', id, 'fromAccountId', from_account_id, 'toAccountId', to_account_id,
               'senderId', sender_id, 'recipientId', recipient_id, 'amount', amount,
               'deviceId', device_id, 'ipAddress', host(ip_address),
               'createdAt', created_at
           )
       ), created_at, id % 100 <> 0,
       now() + CASE WHEN id % 200 = 0 THEN interval '5 minutes' ELSE interval '-1 minute' END,
       CASE WHEN id % 100 <> 0 THEN created_at + interval '1 second' END
FROM events;

WITH events AS MATERIALIZED (
    SELECT uuidv7() AS event_id, id, created_at + interval '1 second' AS decided_at,
           state IN ('RELEASING_FUNDS', 'REJECTED') AS rejected
    FROM transfer
    WHERE id BETWEEN 1000001 AND 1100000
)
INSERT INTO inbox_event (event_id, transfer_id, payload, applied, processed_at)
SELECT event_id, id, jsonb_build_object(
           'eventId', event_id, 'eventType', 'FraudDecision',
           'occurredAt', decided_at,
           'data', jsonb_build_object(
               'transferId', id, 'source', 'AUTOMATIC',
               'decision', CASE WHEN rejected THEN 'REJECTED' ELSE 'APPROVED' END,
               'riskScore', CASE WHEN rejected THEN 75 ELSE 0 END,
               'triggeredRules', CASE WHEN rejected THEN jsonb_build_array(
                   jsonb_build_object('ruleId', 1, 'score', 30, 'reason', 'Не менее шести попыток за десять минут'),
                   jsonb_build_object('ruleId', 2, 'score', 30, 'reason', 'Сумма выше сильного порога по истории клиента'),
                   jsonb_build_object('ruleId', 3, 'score', 15, 'reason', 'Новое устройство в первые 24 часа')
               ) ELSE '[]'::jsonb END
           )
       ), true, decided_at
FROM events;

INSERT INTO audit_log (occurred_at, actor_type, action, account_id, details)
SELECT created_at, 'SYSTEM', 'LIMIT_CHANGED', from_account_id, '{}'::jsonb
FROM transfer WHERE id BETWEEN 1000001 AND 1100000;

INSERT INTO external_payment (transfer_id, amount, created_at, external_payment_id)
SELECT id, amount, created_at, 'bulk-payment-' || id
FROM transfer WHERE id BETWEEN 1000001 AND 1100000;

ALTER TABLE external_payment ENABLE TRIGGER USER;
ALTER TABLE ledger_entry ENABLE TRIGGER USER;
ALTER TABLE transfer ENABLE TRIGGER USER;
ALTER TABLE account ENABLE TRIGGER USER;
COMMIT;

SELECT setval('account_id_seq', (SELECT max(id) FROM account));
SELECT setval('transfer_id_seq', (SELECT max(id) FROM transfer));
SELECT setval('ledger_entry_id_seq', (SELECT max(id) FROM ledger_entry));
SELECT setval('audit_log_id_seq', (SELECT max(id) FROM audit_log));
SELECT setval('external_payment_id_seq', (SELECT max(id) FROM external_payment));
ANALYZE account;
ANALYZE transfer;
ANALYZE ledger_entry;
ANALYZE outbox_event;
ANALYZE inbox_event;
ANALYZE audit_log;
ANALYZE external_payment;
