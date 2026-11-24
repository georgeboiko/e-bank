BEGIN;

CREATE INDEX user_login_logined_at_idx ON user_login (logined_at);

CREATE INDEX outbox_event_pending_idx
    ON outbox_event (next_attempt_at, created_at)
    WHERE published = false;

CREATE INDEX audit_log_occurred_idx ON audit_log (occurred_at DESC, id DESC);

COMMIT;
