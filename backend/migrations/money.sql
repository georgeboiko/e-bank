BEGIN;

CREATE TABLE account (
    id serial PRIMARY KEY,
    owner_id integer,
    type text NOT NULL CHECK (type IN ('CLIENT', 'TECHNICAL')),
    state text NOT NULL DEFAULT 'ACTIVE'
        CHECK (state IN ('ACTIVE', 'BLOCKED', 'CLOSED')),
    balance numeric(19,2) NOT NULL DEFAULT 0,
    reserved_sum numeric(19,2) NOT NULL DEFAULT 0,
    opened_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (type = 'CLIENT' AND owner_id IS NOT NULL AND owner_id > 0)
        OR (type = 'TECHNICAL' AND owner_id IS NULL)
    ),
    CHECK (balance >= 0 AND balance <> 'NaN'::numeric),
    CHECK (reserved_sum >= 0 AND reserved_sum <= balance),
    CHECK (state <> 'CLOSED' OR (balance = 0 AND reserved_sum = 0))
);

CREATE TABLE transfer (
    id serial PRIMARY KEY,
    from_account_id integer NOT NULL REFERENCES account(id),
    to_account_id integer NOT NULL REFERENCES account(id),
    amount numeric(19,2) NOT NULL CHECK (amount > 0 AND amount <> 'NaN'::numeric),
    state text NOT NULL DEFAULT 'PENDING' CHECK (state IN (
        'PENDING', 'FRAUD_CHECK_PENDING', 'COMPLETING', 'RELEASING_FUNDS',
        'COMPLETED', 'REJECTED', 'CANCELLED'
    )),
    idempotency_key uuid NOT NULL UNIQUE,
    reject_cause text CHECK (reject_cause = 'ANTIFRAUD_REJECTED'),
    device_id text,
    ip_address inet,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    CHECK (from_account_id <> to_account_id),
    CHECK (updated_at >= created_at),
    CHECK (finished_at IS NULL OR finished_at >= created_at),
    CHECK (
        (state IN ('COMPLETED', 'REJECTED', 'CANCELLED'))
        = (finished_at IS NOT NULL)
    ),
    CHECK (
        (state IN ('RELEASING_FUNDS', 'REJECTED'))
        = (reject_cause IS NOT NULL)
    )
);

CREATE TABLE ledger_entry (
    id serial PRIMARY KEY,
    transfer_id integer NOT NULL REFERENCES transfer(id),
    account_id integer NOT NULL REFERENCES account(id),
    amount numeric(19,2) NOT NULL CHECK (amount <> 0 AND amount <> 'NaN'::numeric),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (transfer_id, account_id)
);

CREATE TABLE audit_log (
    id serial PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_type text NOT NULL CHECK (actor_type IN ('USER', 'ADMIN', 'SYSTEM')),
    actor_id integer,
    action text NOT NULL CHECK (action IN (
        'ACCOUNT_OPENED', 'ACCOUNT_BLOCKED', 'ACCOUNT_UNBLOCKED',
        'ACCOUNT_CLOSED', 'LIMIT_CHANGED'
    )),
    account_id integer REFERENCES account(id),
    details jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object'),
    CHECK (
        (actor_type = 'SYSTEM' AND actor_id IS NULL)
        OR (actor_type <> 'SYSTEM' AND actor_id IS NOT NULL AND actor_id > 0)
    )
);

CREATE TABLE outbox_event (
    event_id uuid PRIMARY KEY,
    transfer_id integer NOT NULL UNIQUE REFERENCES transfer(id),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    published boolean NOT NULL DEFAULT false,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    last_error text,
    CHECK (published = (published_at IS NOT NULL))
);

CREATE TABLE inbox_event (
    event_id uuid PRIMARY KEY,
    transfer_id integer NOT NULL UNIQUE REFERENCES transfer(id),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    applied boolean NOT NULL DEFAULT false,
    processed_at timestamptz,
    CHECK (applied = (processed_at IS NOT NULL))
);

CREATE TABLE external_payment (
    id serial PRIMARY KEY,
    transfer_id integer NOT NULL UNIQUE REFERENCES transfer(id),
    amount numeric(19,2) NOT NULL CHECK (amount > 0 AND amount <> 'NaN'::numeric),
    created_at timestamptz NOT NULL DEFAULT now(),
    external_payment_id text NOT NULL UNIQUE CHECK (btrim(external_payment_id) <> '')
);

COMMIT;
