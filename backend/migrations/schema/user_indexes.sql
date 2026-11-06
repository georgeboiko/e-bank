BEGIN;

CREATE INDEX outbox_event_pending_idx
    ON outbox_event (next_attempt_at, created_at)
    WHERE published = false;

COMMIT;
