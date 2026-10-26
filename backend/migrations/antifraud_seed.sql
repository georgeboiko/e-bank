BEGIN;

INSERT INTO app_user (id) VALUES
    ('11111111-1111-4111-8111-111111111111'),
    ('22222222-2222-4222-8222-222222222222'),
    ('33333333-3333-4333-8333-333333333333'),
    ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa');

INSERT INTO bank_account (id, owner_id) VALUES
    (2, '11111111-1111-4111-8111-111111111111'),
    (3, '22222222-2222-4222-8222-222222222222'),
    (4, '33333333-3333-4333-8333-333333333333');

INSERT INTO user_device (id, user_id, device, first_seen_at, last_seen_at, use_count) VALUES
    (1, '11111111-1111-4111-8111-111111111111', 'alice-main', '2026-10-26 09:00:00+03:00', '2026-10-26 18:00:00+03:00', 3),
    (2, '11111111-1111-4111-8111-111111111111', 'alice-new', '2026-10-27 10:00:00+03:00', '2026-10-27 10:02:00+03:00', 4);

INSERT INTO login_event (id, user_id, ip_address, occurred_at, latitude, longitude, geo_status, geo_resolved_at) VALUES
    (1, '11111111-1111-4111-8111-111111111111', '203.0.113.10', '2026-10-27 09:50:00+03:00', '59.9386', '30.3141', 'FRESH', '2026-10-27 09:50:02+03:00'),
    (2, '22222222-2222-4222-8222-222222222222', '198.51.100.20', '2026-10-27 09:55:00+03:00', '55.7558', '37.6173', 'FRESH', '2026-10-27 09:55:02+03:00'),
    (3, '33333333-3333-4333-8333-333333333333', '2001:db8::30', '2026-10-27 09:58:00+03:00', NULL, NULL, 'MISSING', NULL);

INSERT INTO transfer_attempt (id, from_account_id, to_account_id, amount, device, ip_address, created_at, received_at, latitude, longitude, geo_status, geo_resolved_at) VALUES
    (2, 2, 3, 1000, 'alice-main', '203.0.113.10', '2026-10-26 09:00:00+03:00', '2026-10-26 09:00:01+03:00', '59.9386', '30.3141', 'FRESH', '2026-10-26 09:00:01+03:00'),
    (3, 2, 4, 1200, 'alice-main', '203.0.113.10', '2026-10-26 12:00:00+03:00', '2026-10-26 12:00:01+03:00', '59.9386', '30.3141', 'FRESH', '2026-10-26 12:00:01+03:00'),
    (4, 2, 3, 900, 'alice-main', '203.0.113.10', '2026-10-26 18:00:00+03:00', '2026-10-26 18:00:01+03:00', '59.9386', '30.3141', 'FRESH', '2026-10-26 18:00:01+03:00'),
    (5, 2, 3, 20000, 'alice-new', '203.0.113.10', '2026-10-27 10:00:00+03:00', '2026-10-27 10:00:01+03:00', '59.9386', '30.3141', 'FRESH', '2026-10-27 10:00:01+03:00'),
    (6, 2, 4, 1000, 'alice-new', '203.0.113.10', '2026-10-27 10:00:30+03:00', '2026-10-27 10:00:31+03:00', '59.9386', '30.3141', 'FRESH', '2026-10-27 10:00:31+03:00'),
    (7, 2, 3, 50000, 'alice-new', '198.51.100.10', '2026-10-27 10:01:00+03:00', '2026-10-27 10:01:01+03:00', '55.7558', '37.6173', 'FRESH', '2026-10-27 10:01:01+03:00'),
    (8, 2, 4, 40000, 'alice-new', '203.0.113.10', '2026-10-27 10:02:00+03:00', '2026-10-27 10:02:01+03:00', '59.9386', '30.3141', 'FRESH', '2026-10-27 10:02:01+03:00');

INSERT INTO antifraud_check (id, transfer_id, risk_score, automatic_decision, checked_at) VALUES
    (2, 2, 0, 'APPROVED', '2026-10-26 09:00:02+03:00'),
    (3, 3, 0, 'APPROVED', '2026-10-26 12:00:02+03:00'),
    (4, 4, 0, 'APPROVED', '2026-10-26 18:00:02+03:00'),
    (5, 5, 45, 'REVIEW_REQUIRED', '2026-10-27 10:00:02+03:00'),
    (6, 6, 15, 'APPROVED', '2026-10-27 10:00:32+03:00'),
    (7, 7, 75, 'REJECTED', '2026-10-27 10:01:02+03:00'),
    (8, 8, 60, 'REVIEW_REQUIRED', '2026-10-27 10:02:02+03:00');

INSERT INTO rule_evaluation (check_id, rule_id, status, score, reason) VALUES
    (2, 1, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (2, 2, 'SKIPPED', 0, 'Недостаточно истории'),
    (2, 3, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (2, 4, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (2, 5, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (2, 6, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (2, 7, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (2, 8, 'SKIPPED', 0, 'Недостаточно истории'),
    (3, 1, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (3, 2, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (3, 3, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (3, 4, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (3, 5, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (3, 6, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (3, 7, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (3, 8, 'SKIPPED', 0, 'Недостаточно истории'),
    (4, 1, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (4, 2, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (4, 3, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (4, 4, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (4, 5, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (4, 6, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (4, 7, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (4, 8, 'SKIPPED', 0, 'Недостаточно истории'),
    (5, 1, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (5, 2, 'TRIGGERED', 30, 'Сумма выше сильного порога по истории клиента'),
    (5, 3, 'TRIGGERED', 15, 'Новое устройство в первые 24 часа'),
    (5, 4, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (5, 5, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (5, 6, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (5, 7, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (5, 8, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (6, 1, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (6, 2, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (6, 3, 'TRIGGERED', 15, 'Новое устройство в первые 24 часа'),
    (6, 4, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (6, 5, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (6, 6, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (6, 7, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (6, 8, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (7, 1, 'TRIGGERED', 15, 'Не менее трёх попыток за две минуты'),
    (7, 2, 'TRIGGERED', 30, 'Сумма выше сильного порога по истории клиента'),
    (7, 3, 'TRIGGERED', 15, 'Новое устройство в первые 24 часа'),
    (7, 4, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (7, 5, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (7, 6, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (7, 7, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (7, 8, 'TRIGGERED', 15, 'Москва после входа из Санкт-Петербурга менее двух часов назад'),
    (8, 1, 'TRIGGERED', 15, 'Не менее трёх попыток за две минуты'),
    (8, 2, 'TRIGGERED', 30, 'Сумма выше сильного порога по истории клиента'),
    (8, 3, 'TRIGGERED', 15, 'Новое устройство в первые 24 часа'),
    (8, 4, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (8, 5, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (8, 6, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (8, 7, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено'),
    (8, 8, 'NOT_TRIGGERED', 0, 'Условие правила не выполнено');

INSERT INTO review_case (id, check_id, created_at) VALUES
    (1, 5, '2026-10-27 10:00:02+03:00'),
    (2, 8, '2026-10-27 10:02:02+03:00');

UPDATE review_case SET status = 'IN_PROGRESS', analyst_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa' WHERE id = 1;
UPDATE review_case SET status = 'CLOSED', decision = 'APPROVED',
    comment = 'Клиент подтвердил перевод', decided_at = '2026-10-27 10:00:10+03' WHERE id = 1;

INSERT INTO inbox_event (event_id, transfer_id, occurred_at, received_at, payload, applied, processed_at) VALUES
    ('01a17c1f-79c0-7065-8000-000000000065', NULL, '2026-10-27 09:50:00+03:00', '2026-10-27 09:50:01+03:00', '{"eventId":"01a17c1f-79c0-7065-8000-000000000065","eventType":"LoginSucceeded","occurredAt":"2026-10-27T09:50:00+03:00","data":{"userId":"11111111-1111-4111-8111-111111111111","ipAddress":"203.0.113.10"}}'::jsonb, true, '2026-10-27 09:50:03+03:00'),
    ('01a17c24-0da0-7066-8000-000000000066', NULL, '2026-10-27 09:55:00+03:00', '2026-10-27 09:55:01+03:00', '{"eventId":"01a17c24-0da0-7066-8000-000000000066","eventType":"LoginSucceeded","occurredAt":"2026-10-27T09:55:00+03:00","data":{"userId":"22222222-2222-4222-8222-222222222222","ipAddress":"198.51.100.20"}}'::jsonb, true, '2026-10-27 09:55:03+03:00'),
    ('01a17c26-ccc0-7067-8000-000000000067', NULL, '2026-10-27 09:58:00+03:00', '2026-10-27 09:58:01+03:00', '{"eventId":"01a17c26-ccc0-7067-8000-000000000067","eventType":"LoginSucceeded","occurredAt":"2026-10-27T09:58:00+03:00","data":{"userId":"33333333-3333-4333-8333-333333333333","ipAddress":"2001:db8::30"}}'::jsonb, true, '2026-10-27 09:58:03+03:00'),
    ('01a176cb-5700-70ca-8000-0000000000ca', 2, '2026-10-26 09:00:00+03:00', '2026-10-26 09:00:01+03:00', '{"eventId":"01a176cb-5700-70ca-8000-0000000000ca","eventType":"TransferRequested","occurredAt":"2026-10-26T09:00:00+03:00","data":{"transferId":2,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":1000,"deviceId":"alice-main","ipAddress":"203.0.113.10","createdAt":"2026-10-26T09:00:00+03:00"}}'::jsonb, true, '2026-10-26 09:00:02+03:00'),
    ('01a17770-2280-70cb-8000-0000000000cb', 3, '2026-10-26 12:00:00+03:00', '2026-10-26 12:00:01+03:00', '{"eventId":"01a17770-2280-70cb-8000-0000000000cb","eventType":"TransferRequested","occurredAt":"2026-10-26T12:00:00+03:00","data":{"transferId":3,"fromAccountId":2,"toAccountId":4,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"33333333-3333-4333-8333-333333333333","amount":1200,"deviceId":"alice-main","ipAddress":"203.0.113.10","createdAt":"2026-10-26T12:00:00+03:00"}}'::jsonb, true, '2026-10-26 12:00:02+03:00'),
    ('01a178b9-b980-70cc-8000-0000000000cc', 4, '2026-10-26 18:00:00+03:00', '2026-10-26 18:00:01+03:00', '{"eventId":"01a178b9-b980-70cc-8000-0000000000cc","eventType":"TransferRequested","occurredAt":"2026-10-26T18:00:00+03:00","data":{"transferId":4,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":900,"deviceId":"alice-main","ipAddress":"203.0.113.10","createdAt":"2026-10-26T18:00:00+03:00"}}'::jsonb, true, '2026-10-26 18:00:02+03:00'),
    ('01a17c28-a180-70cd-8000-0000000000cd', 5, '2026-10-27 10:00:00+03:00', '2026-10-27 10:00:01+03:00', '{"eventId":"01a17c28-a180-70cd-8000-0000000000cd","eventType":"TransferRequested","occurredAt":"2026-10-27T10:00:00+03:00","data":{"transferId":5,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":20000,"deviceId":"alice-new","ipAddress":"203.0.113.10","createdAt":"2026-10-27T10:00:00+03:00"}}'::jsonb, true, '2026-10-27 10:00:02+03:00'),
    ('01a17c29-16b0-70ce-8000-0000000000ce', 6, '2026-10-27 10:00:30+03:00', '2026-10-27 10:00:31+03:00', '{"eventId":"01a17c29-16b0-70ce-8000-0000000000ce","eventType":"TransferRequested","occurredAt":"2026-10-27T10:00:30+03:00","data":{"transferId":6,"fromAccountId":2,"toAccountId":4,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"33333333-3333-4333-8333-333333333333","amount":1000,"deviceId":"alice-new","ipAddress":"203.0.113.10","createdAt":"2026-10-27T10:00:30+03:00"}}'::jsonb, true, '2026-10-27 10:00:32+03:00'),
    ('01a17c29-8be0-70cf-8000-0000000000cf', 7, '2026-10-27 10:01:00+03:00', '2026-10-27 10:01:01+03:00', '{"eventId":"01a17c29-8be0-70cf-8000-0000000000cf","eventType":"TransferRequested","occurredAt":"2026-10-27T10:01:00+03:00","data":{"transferId":7,"fromAccountId":2,"toAccountId":3,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"22222222-2222-4222-8222-222222222222","amount":50000,"deviceId":"alice-new","ipAddress":"198.51.100.10","createdAt":"2026-10-27T10:01:00+03:00"}}'::jsonb, true, '2026-10-27 10:01:02+03:00'),
    ('01a17c2a-7640-70d0-8000-0000000000d0', 8, '2026-10-27 10:02:00+03:00', '2026-10-27 10:02:01+03:00', '{"eventId":"01a17c2a-7640-70d0-8000-0000000000d0","eventType":"TransferRequested","occurredAt":"2026-10-27T10:02:00+03:00","data":{"transferId":8,"fromAccountId":2,"toAccountId":4,"senderId":"11111111-1111-4111-8111-111111111111","recipientId":"33333333-3333-4333-8333-333333333333","amount":40000,"deviceId":"alice-new","ipAddress":"203.0.113.10","createdAt":"2026-10-27T10:02:00+03:00"}}'::jsonb, true, '2026-10-27 10:02:02+03:00');

INSERT INTO outbox_event (event_id, check_id, payload, created_at, published, attempt_count, next_attempt_at, published_at) VALUES
    ('01a176cb-5ed0-712e-8000-00000000012e', 2, '{"eventId":"01a176cb-5ed0-712e-8000-00000000012e","eventType":"AntifraudDecision","occurredAt":"2026-10-26T09:00:02+03:00","data":{"transferId":2,"decision":"APPROVED","riskScore":0}}'::jsonb, '2026-10-26 09:00:02+03:00', true, 1, '2026-10-26 09:00:02+03:00', '2026-10-26 09:00:03+03:00'),
    ('01a17770-2a50-712f-8000-00000000012f', 3, '{"eventId":"01a17770-2a50-712f-8000-00000000012f","eventType":"AntifraudDecision","occurredAt":"2026-10-26T12:00:02+03:00","data":{"transferId":3,"decision":"APPROVED","riskScore":0}}'::jsonb, '2026-10-26 12:00:02+03:00', true, 1, '2026-10-26 12:00:02+03:00', '2026-10-26 12:00:03+03:00'),
    ('01a178b9-c150-7130-8000-000000000130', 4, '{"eventId":"01a178b9-c150-7130-8000-000000000130","eventType":"AntifraudDecision","occurredAt":"2026-10-26T18:00:02+03:00","data":{"transferId":4,"decision":"APPROVED","riskScore":0}}'::jsonb, '2026-10-26 18:00:02+03:00', true, 1, '2026-10-26 18:00:02+03:00', '2026-10-26 18:00:03+03:00'),
    ('01a17c28-c890-7131-8000-000000000131', 5, '{"eventId":"01a17c28-c890-7131-8000-000000000131","eventType":"AntifraudDecision","occurredAt":"2026-10-27T10:00:10+03:00","data":{"transferId":5,"decision":"APPROVED","riskScore":45}}'::jsonb, '2026-10-27 10:00:10+03:00', true, 1, '2026-10-27 10:00:10+03:00', '2026-10-27 10:00:11+03:00'),
    ('01a17c29-1e80-7132-8000-000000000132', 6, '{"eventId":"01a17c29-1e80-7132-8000-000000000132","eventType":"AntifraudDecision","occurredAt":"2026-10-27T10:00:32+03:00","data":{"transferId":6,"decision":"APPROVED","riskScore":15}}'::jsonb, '2026-10-27 10:00:32+03:00', true, 1, '2026-10-27 10:00:32+03:00', '2026-10-27 10:00:33+03:00'),
    ('01a17c29-93b0-7133-8000-000000000133', 7, '{"eventId":"01a17c29-93b0-7133-8000-000000000133","eventType":"AntifraudDecision","occurredAt":"2026-10-27T10:01:02+03:00","data":{"transferId":7,"decision":"REJECTED","riskScore":75}}'::jsonb, '2026-10-27 10:01:02+03:00', true, 1, '2026-10-27 10:01:02+03:00', '2026-10-27 10:01:03+03:00');

INSERT INTO audit_log (id, occurred_at, actor_type, actor_id, action, check_id, details) VALUES
    (1, '2026-10-26 09:00:02+03:00', 'SYSTEM', NULL, 'CHECK_COMPLETED', 2, '{"riskScore":0}'::jsonb),
    (2, '2026-10-26 12:00:02+03:00', 'SYSTEM', NULL, 'CHECK_COMPLETED', 3, '{"riskScore":0}'::jsonb),
    (3, '2026-10-26 18:00:02+03:00', 'SYSTEM', NULL, 'CHECK_COMPLETED', 4, '{"riskScore":0}'::jsonb),
    (4, '2026-10-27 10:00:02+03:00', 'SYSTEM', NULL, 'CHECK_COMPLETED', 5, '{"riskScore":45}'::jsonb),
    (5, '2026-10-27 10:00:32+03:00', 'SYSTEM', NULL, 'CHECK_COMPLETED', 6, '{"riskScore":15}'::jsonb),
    (6, '2026-10-27 10:01:02+03:00', 'SYSTEM', NULL, 'CHECK_COMPLETED', 7, '{"riskScore":75}'::jsonb),
    (7, '2026-10-27 10:02:02+03:00', 'SYSTEM', NULL, 'CHECK_COMPLETED', 8, '{"riskScore":60}'::jsonb),
    (8, '2026-10-27 10:00:10+03:00', 'ANALYST', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', 'REVIEW_CLOSED', 5, '{"decision":"APPROVED"}'::jsonb);

SELECT setval('user_device_id_seq', (SELECT max(id) FROM user_device));
SELECT setval('login_event_id_seq', (SELECT max(id) FROM login_event));
SELECT setval('antifraud_check_id_seq', (SELECT max(id) FROM antifraud_check));
SELECT setval('review_case_id_seq', (SELECT max(id) FROM review_case));
SELECT setval('audit_log_id_seq', (SELECT max(id) FROM audit_log));

COMMIT;
