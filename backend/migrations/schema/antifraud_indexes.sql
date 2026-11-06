BEGIN;

CREATE INDEX bank_account_owner_idx ON bank_account (owner_id);

CREATE INDEX transfer_attempt_from_created_idx
    ON transfer_attempt (from_account_id, created_at DESC);

CREATE INDEX transfer_attempt_to_created_idx
    ON transfer_attempt (to_account_id, created_at DESC);

CREATE INDEX transfer_attempt_device_created_idx
    ON transfer_attempt (device, created_at DESC)
    WHERE device IS NOT NULL;

CREATE INDEX login_event_user_occurred_idx
    ON login_event (user_id, occurred_at DESC);

CREATE INDEX antifraud_check_checked_idx ON antifraud_check (checked_at);

CREATE INDEX review_case_pending_idx ON review_case (created_at, id)
    WHERE status IN ('OPEN', 'IN_PROGRESS');

CREATE INDEX audit_log_check_occurred_idx
    ON audit_log (check_id, occurred_at DESC);

CREATE INDEX audit_log_occurred_idx ON audit_log (occurred_at DESC);

CREATE INDEX outbox_event_pending_idx
    ON outbox_event (next_attempt_at, created_at)
    WHERE published = false;

COMMIT;
