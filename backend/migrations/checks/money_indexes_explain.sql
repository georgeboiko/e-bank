ANALYZE account, transfer, ledger_entry, audit_log, outbox_event;

EXPLAIN ANALYZE
SELECT id, state, balance, reserved_sum
FROM account
WHERE owner_id = 'bulk-user-42';

BEGIN;
DROP INDEX account_owner_idx;

EXPLAIN ANALYZE
SELECT id, state, balance, reserved_sum
FROM account
WHERE owner_id = 'bulk-user-42';

ROLLBACK;

EXPLAIN ANALYZE
SELECT id, to_account_id, amount, state, created_at
FROM transfer
WHERE from_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 50;

BEGIN;
DROP INDEX transfer_from_created_idx;

EXPLAIN ANALYZE
SELECT id, to_account_id, amount, state, created_at
FROM transfer
WHERE from_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 50;

ROLLBACK;

EXPLAIN ANALYZE
SELECT id, from_account_id, amount, state, created_at
FROM transfer
WHERE to_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 50;

BEGIN;
DROP INDEX transfer_to_created_idx;

EXPLAIN ANALYZE
SELECT id, from_account_id, amount, state, created_at
FROM transfer
WHERE to_account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 50;

ROLLBACK;

EXPLAIN ANALYZE
SELECT state, count(*), sum(amount)
FROM transfer
WHERE created_at >= timestamptz '2026-10-31 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
GROUP BY state;

BEGIN;
DROP INDEX transfer_created_idx;

EXPLAIN ANALYZE
SELECT state, count(*), sum(amount)
FROM transfer
WHERE created_at >= timestamptz '2026-10-31 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
GROUP BY state;

ROLLBACK;

EXPLAIN ANALYZE
SELECT id, state
FROM transfer
WHERE state IN ('COMPLETING', 'RELEASING_FUNDS')
ORDER BY id
LIMIT 100;

BEGIN;
DROP INDEX transfer_processing_idx;

EXPLAIN ANALYZE
SELECT id, state
FROM transfer
WHERE state IN ('COMPLETING', 'RELEASING_FUNDS')
ORDER BY id
LIMIT 100;

ROLLBACK;

EXPLAIN ANALYZE
SELECT id, transfer_id, amount, created_at
FROM ledger_entry
WHERE account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 50;

BEGIN;
DROP INDEX ledger_entry_account_created_idx;

EXPLAIN ANALYZE
SELECT id, transfer_id, amount, created_at
FROM ledger_entry
WHERE account_id = 100042
  AND created_at >= timestamptz '2026-09-01 00:00+03'
  AND created_at < timestamptz '2026-11-01 00:00+03'
ORDER BY created_at DESC
LIMIT 50;

ROLLBACK;

EXPLAIN ANALYZE
SELECT id, occurred_at, actor_type, action, account_id
FROM audit_log
WHERE occurred_at >= timestamptz '2026-10-31 00:00+03'
  AND occurred_at < timestamptz '2026-11-01 00:00+03'
ORDER BY occurred_at DESC
LIMIT 100;

BEGIN;
DROP INDEX audit_log_occurred_idx;

EXPLAIN ANALYZE
SELECT id, occurred_at, actor_type, action, account_id
FROM audit_log
WHERE occurred_at >= timestamptz '2026-10-31 00:00+03'
  AND occurred_at < timestamptz '2026-11-01 00:00+03'
ORDER BY occurred_at DESC
LIMIT 100;

ROLLBACK;

EXPLAIN ANALYZE
SELECT event_id, transfer_id, payload
FROM outbox_event
WHERE published = false AND next_attempt_at <= now()
ORDER BY next_attempt_at, created_at
LIMIT 100;

BEGIN;
DROP INDEX outbox_event_pending_idx;

EXPLAIN ANALYZE
SELECT event_id, transfer_id, payload
FROM outbox_event
WHERE published = false AND next_attempt_at <= now()
ORDER BY next_attempt_at, created_at
LIMIT 100;

ROLLBACK;
