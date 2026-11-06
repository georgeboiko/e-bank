BEGIN;

INSERT INTO account (id, owner_id, type, balance, opened_at) VALUES
    (1, NULL, 'TECHNICAL', 0, '2026-10-25 11:00:00+03:00'),
    (2, '11111111-1111-4111-8111-111111111111', 'CLIENT', 0, '2026-10-25 11:00:00+03:00'),
    (3, '22222222-2222-4222-8222-222222222222', 'CLIENT', 0, '2026-10-25 11:00:00+03:00'),
    (4, '33333333-3333-4333-8333-333333333333', 'CLIENT', 0, '2026-10-25 11:00:00+03:00');

INSERT INTO transfer (id, from_account_id, to_account_id, amount, idempotency_key, device_id, ip_address, created_at, updated_at) VALUES
    (1, 1, 2, 100000, '10000000-0000-4000-8000-000000000001', NULL, NULL, '2026-10-25 12:00:00+03:00', '2026-10-25 12:00:00+03:00');

COMMIT;

BEGIN;

INSERT INTO external_payment (id, transfer_id, amount, created_at, external_payment_id) VALUES
    (1, 1, 100000, '2026-10-25 12:00:05+03:00', 'demo-payment-001');
UPDATE transfer SET state = 'COMPLETING', updated_at = '2026-10-25 12:00:12+03:00'
WHERE id = 1;

COMMIT;

BEGIN;

INSERT INTO ledger_entry (id, transfer_id, account_id, amount, created_at) VALUES
    (1, 1, 1, -100000, '2026-10-25 12:00:15+03:00'),
    (2, 1, 2, 100000, '2026-10-25 12:00:15+03:00');
UPDATE transfer SET state = 'COMPLETED', updated_at = '2026-10-25 12:00:15+03:00',
    finished_at = '2026-10-25 12:00:15+03:00' WHERE id = 1;

INSERT INTO transfer (id, from_account_id, to_account_id, amount, idempotency_key, device_id, ip_address, created_at, updated_at) VALUES
    (2, 2, 3, 1000, '10000000-0000-4000-8000-000000000002', 'alice-main', '203.0.113.10', '2026-10-26 09:00:00+03:00', '2026-10-26 09:00:00+03:00'),
    (3, 2, 4, 1200, '10000000-0000-4000-8000-000000000003', 'alice-main', '203.0.113.10', '2026-10-26 12:00:00+03:00', '2026-10-26 12:00:00+03:00'),
    (4, 2, 3, 900, '10000000-0000-4000-8000-000000000004', 'alice-main', '203.0.113.10', '2026-10-26 18:00:00+03:00', '2026-10-26 18:00:00+03:00'),
    (5, 2, 3, 20000, '10000000-0000-4000-8000-000000000005', 'alice-new', '203.0.113.10', '2026-10-27 10:00:00+03:00', '2026-10-27 10:00:00+03:00'),
    (6, 2, 4, 1000, '10000000-0000-4000-8000-000000000006', 'alice-new', '203.0.113.10', '2026-10-27 10:00:30+03:00', '2026-10-27 10:00:30+03:00'),
    (7, 2, 3, 50000, '10000000-0000-4000-8000-000000000007', 'alice-new', '198.51.100.10', '2026-10-27 10:01:00+03:00', '2026-10-27 10:01:00+03:00'),
    (8, 2, 4, 40000, '10000000-0000-4000-8000-000000000008', 'alice-new', '203.0.113.10', '2026-10-27 10:02:00+03:00', '2026-10-27 10:02:00+03:00');

UPDATE transfer SET state = 'FRAUD_CHECK_PENDING', updated_at = created_at + interval '2 seconds'
WHERE id BETWEEN 2 AND 8;

UPDATE transfer SET state = 'COMPLETING', updated_at = created_at + interval '12 seconds'
WHERE id BETWEEN 2 AND 6;

INSERT INTO ledger_entry (id, transfer_id, account_id, amount, created_at) VALUES
    (3, 2, 2, -1000, '2026-10-26 09:00:15+03:00'),
    (4, 2, 3, 1000, '2026-10-26 09:00:15+03:00'),
    (5, 3, 2, -1200, '2026-10-26 12:00:15+03:00'),
    (6, 3, 4, 1200, '2026-10-26 12:00:15+03:00'),
    (7, 4, 2, -900, '2026-10-26 18:00:15+03:00'),
    (8, 4, 3, 900, '2026-10-26 18:00:15+03:00'),
    (9, 5, 2, -20000, '2026-10-27 10:00:15+03:00'),
    (10, 5, 3, 20000, '2026-10-27 10:00:15+03:00');

UPDATE transfer SET state = 'COMPLETED', finished_at = created_at + interval '15 seconds',
    updated_at = created_at + interval '15 seconds' WHERE id BETWEEN 2 AND 5;

UPDATE transfer SET state = 'RELEASING_FUNDS', reject_cause = 'ANTIFRAUD_REJECTED',
    updated_at = created_at + interval '12 seconds' WHERE id = 7;

UPDATE transfer SET state = 'REJECTED', finished_at = created_at + interval '15 seconds',
    updated_at = created_at + interval '15 seconds' WHERE id = 7;

UPDATE account SET reserved_sum = 41000 WHERE id = 2;

INSERT INTO outbox_event (event_id, transfer_id, payload, created_at, published, attempt_count, next_attempt_at, published_at) VALUES
    ('01a176cb-5700-70ca-8000-0000000000ca', 2, '{"eventId":"01a176cb-5700-70ca-8000-0000000000ca","eventType":"FraudCheckRequested","occurredAt":"2026-10-26T09:00:00+03:00","data":{"transferId":2,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":1000,"deviceId":"alice-main","ipAddress":"203.0.113.10","createdAt":"2026-10-26T09:00:00+03:00"}}'::jsonb, '2026-10-26 09:00:00+03:00', true, 1, '2026-10-26 09:00:00+03:00', '2026-10-26 09:00:01+03:00'),
    ('01a17770-2280-70cb-8000-0000000000cb', 3, '{"eventId":"01a17770-2280-70cb-8000-0000000000cb","eventType":"FraudCheckRequested","occurredAt":"2026-10-26T12:00:00+03:00","data":{"transferId":3,"fromAccountId":2,"toAccountId":4,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"33333333-3333-4333-8333-333333333333","amount":1200,"deviceId":"alice-main","ipAddress":"203.0.113.10","createdAt":"2026-10-26T12:00:00+03:00"}}'::jsonb, '2026-10-26 12:00:00+03:00', true, 1, '2026-10-26 12:00:00+03:00', '2026-10-26 12:00:01+03:00'),
    ('01a178b9-b980-70cc-8000-0000000000cc', 4, '{"eventId":"01a178b9-b980-70cc-8000-0000000000cc","eventType":"FraudCheckRequested","occurredAt":"2026-10-26T18:00:00+03:00","data":{"transferId":4,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":900,"deviceId":"alice-main","ipAddress":"203.0.113.10","createdAt":"2026-10-26T18:00:00+03:00"}}'::jsonb, '2026-10-26 18:00:00+03:00', true, 1, '2026-10-26 18:00:00+03:00', '2026-10-26 18:00:01+03:00'),
    ('01a17c28-a180-70cd-8000-0000000000cd', 5, '{"eventId":"01a17c28-a180-70cd-8000-0000000000cd","eventType":"FraudCheckRequested","occurredAt":"2026-10-27T10:00:00+03:00","data":{"transferId":5,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":20000,"deviceId":"alice-new","ipAddress":"203.0.113.10","createdAt":"2026-10-27T10:00:00+03:00"}}'::jsonb, '2026-10-27 10:00:00+03:00', true, 1, '2026-10-27 10:00:00+03:00', '2026-10-27 10:00:01+03:00'),
    ('01a17c29-16b0-70ce-8000-0000000000ce', 6, '{"eventId":"01a17c29-16b0-70ce-8000-0000000000ce","eventType":"FraudCheckRequested","occurredAt":"2026-10-27T10:00:30+03:00","data":{"transferId":6,"fromAccountId":2,"toAccountId":4,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"33333333-3333-4333-8333-333333333333","amount":1000,"deviceId":"alice-new","ipAddress":"203.0.113.10","createdAt":"2026-10-27T10:00:30+03:00"}}'::jsonb, '2026-10-27 10:00:30+03:00', true, 1, '2026-10-27 10:00:30+03:00', '2026-10-27 10:00:31+03:00'),
    ('01a17c29-8be0-70cf-8000-0000000000cf', 7, '{"eventId":"01a17c29-8be0-70cf-8000-0000000000cf","eventType":"FraudCheckRequested","occurredAt":"2026-10-27T10:01:00+03:00","data":{"transferId":7,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":50000,"deviceId":"alice-new","ipAddress":"198.51.100.10","createdAt":"2026-10-27T10:01:00+03:00"}}'::jsonb, '2026-10-27 10:01:00+03:00', true, 1, '2026-10-27 10:01:00+03:00', '2026-10-27 10:01:01+03:00'),
    ('01a17c2a-7640-70d0-8000-0000000000d0', 8, '{"eventId":"01a17c2a-7640-70d0-8000-0000000000d0","eventType":"FraudCheckRequested","occurredAt":"2026-10-27T10:02:00+03:00","data":{"transferId":8,"fromAccountId":2,"toAccountId":4,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"33333333-3333-4333-8333-333333333333","amount":40000,"deviceId":"alice-new","ipAddress":"203.0.113.10","createdAt":"2026-10-27T10:02:00+03:00"}}'::jsonb, '2026-10-27 10:02:00+03:00', true, 1, '2026-10-27 10:02:00+03:00', '2026-10-27 10:02:01+03:00');

INSERT INTO inbox_event (event_id, transfer_id, payload, applied, processed_at) VALUES
    ('01a176cb-5ed0-712e-8000-00000000012e', 2, '{"eventId":"01a176cb-5ed0-712e-8000-00000000012e","eventType":"FraudDecision","occurredAt":"2026-10-26T09:00:02+03:00","data":{"transferId":2,"decision":"APPROVED","riskScore":0,"source":"AUTOMATIC"}}'::jsonb, true, '2026-10-26 09:00:04+03:00'),
    ('01a17770-2a50-712f-8000-00000000012f', 3, '{"eventId":"01a17770-2a50-712f-8000-00000000012f","eventType":"FraudDecision","occurredAt":"2026-10-26T12:00:02+03:00","data":{"transferId":3,"decision":"APPROVED","riskScore":0,"source":"AUTOMATIC"}}'::jsonb, true, '2026-10-26 12:00:04+03:00'),
    ('01a178b9-c150-7130-8000-000000000130', 4, '{"eventId":"01a178b9-c150-7130-8000-000000000130","eventType":"FraudDecision","occurredAt":"2026-10-26T18:00:02+03:00","data":{"transferId":4,"decision":"APPROVED","riskScore":0,"source":"AUTOMATIC"}}'::jsonb, true, '2026-10-26 18:00:04+03:00'),
    ('01a17c28-c890-7131-8000-000000000131', 5, '{"eventId":"01a17c28-c890-7131-8000-000000000131","eventType":"FraudDecision","occurredAt":"2026-10-27T10:00:10+03:00","data":{"transferId":5,"decision":"APPROVED","riskScore":45,"source":"ANALYST"}}'::jsonb, true, '2026-10-27 10:00:12+03:00'),
    ('01a17c29-1e80-7132-8000-000000000132', 6, '{"eventId":"01a17c29-1e80-7132-8000-000000000132","eventType":"FraudDecision","occurredAt":"2026-10-27T10:00:32+03:00","data":{"transferId":6,"decision":"APPROVED","riskScore":15,"source":"AUTOMATIC"}}'::jsonb, true, '2026-10-27 10:00:34+03:00'),
    ('01a17c29-93b0-7133-8000-000000000133', 7, '{"eventId":"01a17c29-93b0-7133-8000-000000000133","eventType":"FraudDecision","occurredAt":"2026-10-27T10:01:02+03:00","data":{"transferId":7,"decision":"REJECTED","riskScore":75,"source":"AUTOMATIC"}}'::jsonb, true, '2026-10-27 10:01:04+03:00');

INSERT INTO audit_log (id, occurred_at, actor_type, actor_id, action, account_id, details) VALUES
    (1, '2026-10-25 11:00:00+03:00', 'SYSTEM', NULL, 'ACCOUNT_OPENED', 1, '{"initialBalance":0}'::jsonb),
    (2, '2026-10-25 11:00:00+03:00', 'USER', '11111111-1111-4111-8111-111111111111', 'ACCOUNT_OPENED', 2, '{"initialBalance":0}'::jsonb),
    (3, '2026-10-25 11:00:00+03:00', 'USER', '22222222-2222-4222-8222-222222222222', 'ACCOUNT_OPENED', 3, '{"initialBalance":0}'::jsonb),
    (4, '2026-10-25 11:00:00+03:00', 'USER', '33333333-3333-4333-8333-333333333333', 'ACCOUNT_OPENED', 4, '{"initialBalance":0}'::jsonb);

SELECT setval('account_id_seq', (SELECT max(id) FROM account));
SELECT setval('transfer_id_seq', (SELECT max(id) FROM transfer));
SELECT setval('ledger_entry_id_seq', (SELECT max(id) FROM ledger_entry));
SELECT setval('external_payment_id_seq', (SELECT max(id) FROM external_payment));
SELECT setval('audit_log_id_seq', (SELECT max(id) FROM audit_log));

COMMIT;

BEGIN;

UPDATE account SET reserved_sum = reserved_sum - 1000 WHERE id = 2;
INSERT INTO ledger_entry (transfer_id, account_id, amount, created_at) VALUES
    (6, 2, -1000, '2026-10-27 10:00:45+03:00'),
    (6, 4, 1000, '2026-10-27 10:00:45+03:00');
UPDATE transfer SET state = 'COMPLETED', updated_at = '2026-10-27 10:00:45+03:00',
    finished_at = '2026-10-27 10:00:45+03:00' WHERE id = 6;

COMMIT;
