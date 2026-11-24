BEGIN;

ALTER TABLE audit_log
    ADD COLUMN IF NOT EXISTS transfer_id integer REFERENCES transfer(id),
    DROP CONSTRAINT IF EXISTS audit_log_action_check,
    DROP CONSTRAINT IF EXISTS audit_log_transfer_subject_check;

ALTER TABLE audit_log
    ADD CONSTRAINT audit_log_action_check CHECK (action IN (
        'ACCOUNT_OPENED', 'ACCOUNT_BLOCKED', 'ACCOUNT_UNBLOCKED', 'ACCOUNT_CLOSED',
        'LIMIT_CHANGED', 'TRANSFER_CREATED', 'TRANSFER_STATE_CHANGED'
    )),
    ADD CONSTRAINT audit_log_transfer_subject_check CHECK (
        (action IN ('TRANSFER_CREATED', 'TRANSFER_STATE_CHANGED')) = (transfer_id IS NOT NULL)
        AND (transfer_id IS NULL OR account_id IS NULL)
    );

CREATE OR REPLACE FUNCTION record_transfer_audit()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    owner_sub text;
BEGIN
    IF TG_OP = 'INSERT' THEN
        SELECT coalesce(sender.owner_id, recipient.owner_id) INTO owner_sub
        FROM account sender CROSS JOIN account recipient
        WHERE sender.id = NEW.from_account_id AND recipient.id = NEW.to_account_id;

        INSERT INTO audit_log (occurred_at, actor_type, actor_id, action, transfer_id, details)
        VALUES (NEW.created_at, CASE WHEN owner_sub IS NULL THEN 'SYSTEM' ELSE 'USER' END,
            owner_sub, 'TRANSFER_CREATED', NEW.id, jsonb_build_object('state', NEW.state));
    ELSIF NEW.state <> OLD.state THEN
        INSERT INTO audit_log (occurred_at, actor_type, action, transfer_id, details)
        VALUES (NEW.updated_at, 'SYSTEM', 'TRANSFER_STATE_CHANGED', NEW.id,
            jsonb_build_object('previousState', OLD.state, 'state', NEW.state));
    END IF;

    RETURN NULL;
END;
$$;

CREATE OR REPLACE TRIGGER record_transfer_audit
AFTER INSERT OR UPDATE OF state ON transfer
FOR EACH ROW EXECUTE FUNCTION record_transfer_audit();

CREATE INDEX IF NOT EXISTS transfer_completed_finished_idx
    ON transfer (finished_at) WHERE state = 'COMPLETED';

COMMIT;
