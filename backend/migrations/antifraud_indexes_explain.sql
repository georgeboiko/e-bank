ANALYZE bank_account, transfer_attempt, login_event, antifraud_check,
        review_case, audit_log, outbox_event;

EXPLAIN ANALYZE
SELECT id, owner_id
FROM bank_account
WHERE owner_id = 'bulk-user-42';

BEGIN;
DROP INDEX bank_account_owner_idx;

EXPLAIN ANALYZE
SELECT id, owner_id
FROM bank_account
WHERE owner_id = 'bulk-user-42';
ROLLBACK;

EXPLAIN ANALYZE
SELECT id, to_account_id, amount, created_at
FROM transfer_attempt
WHERE from_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 100;

BEGIN;
DROP INDEX transfer_attempt_from_created_idx;

EXPLAIN ANALYZE
SELECT id, to_account_id, amount, created_at
FROM transfer_attempt
WHERE from_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 100;
ROLLBACK;

EXPLAIN ANALYZE
SELECT id, from_account_id, amount, created_at
FROM transfer_attempt
WHERE to_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 100;

BEGIN;
DROP INDEX transfer_attempt_to_created_idx;

EXPLAIN ANALYZE
SELECT id, from_account_id, amount, created_at
FROM transfer_attempt
WHERE to_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 100;
ROLLBACK;

EXPLAIN ANALYZE
SELECT count(DISTINCT a.owner_id)
FROM transfer_attempt t
JOIN bank_account a ON a.id = t.from_account_id
WHERE t.device IS NOT NULL
  AND t.device = 'bulk-device-42'
  AND t.created_at >= timestamptz '2026-10-31 00:00+03'
  AND t.created_at < timestamptz '2026-11-01 00:00+03';

BEGIN;
DROP INDEX transfer_attempt_device_created_idx;

EXPLAIN ANALYZE
SELECT count(DISTINCT a.owner_id)
FROM transfer_attempt t
JOIN bank_account a ON a.id = t.from_account_id
WHERE t.device IS NOT NULL
  AND t.device = 'bulk-device-42'
  AND t.created_at >= timestamptz '2026-10-31 00:00+03'
  AND t.created_at < timestamptz '2026-11-01 00:00+03';
ROLLBACK;

EXPLAIN ANALYZE
SELECT id, ip_address, latitude, longitude, geo_status, occurred_at
FROM login_event
WHERE user_id = 'bulk-user-42'
  AND occurred_at >= timestamptz '2026-10-31 22:00+03'
  AND occurred_at < timestamptz '2026-11-01 00:00+03'
ORDER BY occurred_at DESC
LIMIT 1;

BEGIN;
DROP INDEX login_event_user_occurred_idx;

EXPLAIN ANALYZE
SELECT id, ip_address, latitude, longitude, geo_status, occurred_at
FROM login_event
WHERE user_id = 'bulk-user-42'
  AND occurred_at >= timestamptz '2026-10-31 22:00+03'
  AND occurred_at < timestamptz '2026-11-01 00:00+03'
ORDER BY occurred_at DESC
LIMIT 1;
ROLLBACK;

EXPLAIN ANALYZE
SELECT automatic_decision, count(*)
FROM antifraud_check
WHERE checked_at >= timestamptz '2026-10-31 00:00+03'
  AND checked_at < timestamptz '2026-11-01 00:00+03'
GROUP BY automatic_decision;

BEGIN;
DROP INDEX antifraud_check_checked_idx;

EXPLAIN ANALYZE
SELECT automatic_decision, count(*)
FROM antifraud_check
WHERE checked_at >= timestamptz '2026-10-31 00:00+03'
  AND checked_at < timestamptz '2026-11-01 00:00+03'
GROUP BY automatic_decision;
ROLLBACK;

EXPLAIN ANALYZE
SELECT id, check_id, status, created_at
FROM review_case
WHERE status IN ('OPEN', 'IN_PROGRESS')
ORDER BY created_at, id
LIMIT 100;

BEGIN;
DROP INDEX review_case_pending_idx;

EXPLAIN ANALYZE
SELECT id, check_id, status, created_at
FROM review_case
WHERE status IN ('OPEN', 'IN_PROGRESS')
ORDER BY created_at, id
LIMIT 100;
ROLLBACK;

EXPLAIN ANALYZE
SELECT id, occurred_at, actor_type, action
FROM audit_log
WHERE check_id = 1000042
ORDER BY occurred_at DESC
LIMIT 100;

BEGIN;
DROP INDEX audit_log_check_occurred_idx;

EXPLAIN ANALYZE
SELECT id, occurred_at, actor_type, action
FROM audit_log
WHERE check_id = 1000042
ORDER BY occurred_at DESC
LIMIT 100;
ROLLBACK;

EXPLAIN ANALYZE
SELECT id, occurred_at, actor_type, action
FROM audit_log
WHERE occurred_at >= timestamptz '2026-10-31 00:00+03'
  AND occurred_at < timestamptz '2026-11-01 00:00+03'
ORDER BY occurred_at DESC
LIMIT 100;

BEGIN;
DROP INDEX audit_log_occurred_idx;

EXPLAIN ANALYZE
SELECT id, occurred_at, actor_type, action
FROM audit_log
WHERE occurred_at >= timestamptz '2026-10-31 00:00+03'
  AND occurred_at < timestamptz '2026-11-01 00:00+03'
ORDER BY occurred_at DESC
LIMIT 100;
ROLLBACK;

EXPLAIN ANALYZE
SELECT event_id, payload, next_attempt_at, created_at
FROM outbox_event
WHERE published = false
  AND next_attempt_at <= now()
ORDER BY next_attempt_at, created_at
LIMIT 100;

BEGIN;
DROP INDEX outbox_event_pending_idx;

EXPLAIN ANALYZE
SELECT event_id, payload, next_attempt_at, created_at
FROM outbox_event
WHERE published = false
  AND next_attempt_at <= now()
ORDER BY next_attempt_at, created_at
LIMIT 100;
ROLLBACK;
