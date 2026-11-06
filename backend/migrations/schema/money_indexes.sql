BEGIN;

CREATE INDEX account_owner_idx ON account (owner_id);

CREATE INDEX transfer_from_created_idx
    ON transfer (from_account_id, created_at DESC);
CREATE INDEX transfer_to_created_idx
    ON transfer (to_account_id, created_at DESC);

CREATE INDEX transfer_created_idx ON transfer (created_at);

CREATE INDEX transfer_processing_idx ON transfer (id)
    WHERE state IN ('COMPLETING', 'RELEASING_FUNDS');

CREATE INDEX ledger_entry_account_created_idx
    ON ledger_entry (account_id, created_at DESC);

CREATE INDEX audit_log_occurred_idx ON audit_log (occurred_at DESC);

CREATE INDEX outbox_event_pending_idx
    ON outbox_event (next_attempt_at, created_at)
    WHERE published = false;

COMMIT;
