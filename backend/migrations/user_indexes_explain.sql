ANALYZE outbox_event;

EXPLAIN ANALYZE
SELECT event_id, payload
FROM outbox_event
WHERE published = false AND next_attempt_at <= now()
ORDER BY next_attempt_at, created_at
LIMIT 100;

BEGIN;
DROP INDEX outbox_event_pending_idx;

EXPLAIN ANALYZE
SELECT event_id, payload
FROM outbox_event
WHERE published = false AND next_attempt_at <= now()
ORDER BY next_attempt_at, created_at
LIMIT 100;

ROLLBACK;
