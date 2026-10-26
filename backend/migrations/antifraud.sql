BEGIN;

CREATE TABLE app_user (
    id integer PRIMARY KEY CHECK (id > 0),
    keycloak_sub text UNIQUE CHECK (btrim(keycloak_sub) <> '')
);

CREATE TABLE bank_account (
    id integer PRIMARY KEY CHECK (id > 0),
    owner_id integer NOT NULL REFERENCES app_user(id)
);

CREATE TABLE user_device (
    id serial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES app_user(id),
    device text NOT NULL CHECK (btrim(device) <> ''),
    first_seen_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    use_count integer NOT NULL DEFAULT 1 CHECK (use_count > 0),
    UNIQUE (user_id, device),
    CHECK (last_seen_at >= first_seen_at)
);

CREATE TABLE login_event (
    id serial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES app_user(id),
    ip_address inet NOT NULL,
    occurred_at timestamptz NOT NULL,
    latitude numeric(9,6) CHECK (latitude BETWEEN -90 AND 90),
    longitude numeric(9,6) CHECK (longitude BETWEEN -180 AND 180),
    geo_status text NOT NULL DEFAULT 'MISSING'
        CHECK (geo_status IN ('FRESH', 'CACHED', 'STALE', 'MISSING')),
    geo_resolved_at timestamptz,
    CHECK (
        (geo_status = 'MISSING' AND latitude IS NULL
            AND longitude IS NULL AND geo_resolved_at IS NULL)
        OR (geo_status <> 'MISSING' AND latitude IS NOT NULL
            AND longitude IS NOT NULL AND geo_resolved_at IS NOT NULL)
    )
);

CREATE TABLE transfer_attempt (
    id integer PRIMARY KEY CHECK (id > 0),
    from_account_id integer NOT NULL REFERENCES bank_account(id),
    to_account_id integer NOT NULL REFERENCES bank_account(id),
    amount numeric(19,2) NOT NULL CHECK (amount > 0 AND amount <> 'NaN'::numeric),
    device text,
    ip_address inet,
    created_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    latitude numeric(9,6) CHECK (latitude BETWEEN -90 AND 90),
    longitude numeric(9,6) CHECK (longitude BETWEEN -180 AND 180),
    geo_status text NOT NULL DEFAULT 'MISSING'
        CHECK (geo_status IN ('FRESH', 'CACHED', 'STALE', 'MISSING')),
    geo_resolved_at timestamptz,
    CHECK (from_account_id <> to_account_id),
    CHECK (
        (geo_status = 'MISSING' AND latitude IS NULL
            AND longitude IS NULL AND geo_resolved_at IS NULL)
        OR (geo_status <> 'MISSING' AND latitude IS NOT NULL
            AND longitude IS NOT NULL AND geo_resolved_at IS NOT NULL)
    ),
    CHECK (ip_address IS NOT NULL OR geo_status = 'MISSING')
);

CREATE TABLE antifraud_check (
    id serial PRIMARY KEY,
    transfer_id integer NOT NULL UNIQUE REFERENCES transfer_attempt(id),
    risk_score integer NOT NULL CHECK (risk_score BETWEEN 0 AND 100),
    automatic_decision text NOT NULL
        CHECK (automatic_decision IN ('APPROVED', 'REVIEW_REQUIRED', 'REJECTED')),
    checked_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        automatic_decision = CASE
            WHEN risk_score < 30 THEN 'APPROVED'
            WHEN risk_score < 70 THEN 'REVIEW_REQUIRED'
            ELSE 'REJECTED'
        END
    )
);

CREATE TABLE rule_evaluation (
    check_id integer NOT NULL REFERENCES antifraud_check(id),
    rule_id integer NOT NULL CHECK (rule_id BETWEEN 1 AND 8),
    status text NOT NULL CHECK (status IN ('TRIGGERED', 'NOT_TRIGGERED', 'SKIPPED')),
    score integer NOT NULL CHECK (score BETWEEN 0 AND 100),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    PRIMARY KEY (check_id, rule_id),
    CHECK (
        (status = 'TRIGGERED' AND score > 0)
        OR (status IN ('NOT_TRIGGERED', 'SKIPPED') AND score = 0)
    )
);

CREATE TABLE review_case (
    id serial PRIMARY KEY,
    check_id integer NOT NULL UNIQUE REFERENCES antifraud_check(id),
    analyst_id integer REFERENCES app_user(id),
    status text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'IN_PROGRESS', 'CLOSED')),
    decision text CHECK (decision IN ('APPROVED', 'REJECTED')),
    comment text,
    created_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz,
    CHECK (status = 'OPEN' OR analyst_id IS NOT NULL),
    CHECK ((status = 'CLOSED') = (decision IS NOT NULL)),
    CHECK ((status = 'CLOSED') = (decided_at IS NOT NULL)),
    CHECK (decided_at IS NULL OR decided_at >= created_at)
);

CREATE TABLE inbox_event (
    event_id uuid PRIMARY KEY,
    transfer_id integer REFERENCES transfer_attempt(id),
    occurred_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    applied boolean NOT NULL DEFAULT false,
    processed_at timestamptz,
    CHECK (applied = (processed_at IS NOT NULL))
);

CREATE TABLE outbox_event (
    event_id uuid PRIMARY KEY,
    check_id integer NOT NULL UNIQUE REFERENCES antifraud_check(id),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    published boolean NOT NULL DEFAULT false,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    last_error text,
    CHECK (published = (published_at IS NOT NULL))
);

CREATE TABLE audit_log (
    id serial PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_type text NOT NULL CHECK (actor_type IN ('USER', 'ANALYST', 'ADMIN', 'SYSTEM')),
    actor_id integer REFERENCES app_user(id),
    action text NOT NULL CHECK (btrim(action) <> ''),
    check_id integer REFERENCES antifraud_check(id),
    details jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object'),
    CHECK (
        (actor_type = 'SYSTEM' AND actor_id IS NULL)
        OR (actor_type <> 'SYSTEM' AND actor_id IS NOT NULL)
    )
);

COMMIT;
