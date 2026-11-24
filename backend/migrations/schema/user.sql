BEGIN;

CREATE TABLE user_login (
    id serial PRIMARY KEY,
    keycloak_sub text NOT NULL CHECK (btrim(keycloak_sub) <> ''),
    logined_at timestamptz NOT NULL DEFAULT now(),
    ip inet NOT NULL
);

CREATE TABLE outbox_event (
    event_id uuid PRIMARY KEY,
    user_login_id integer NOT NULL UNIQUE REFERENCES user_login(id),
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
    id bigserial PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_id text NOT NULL CHECK (btrim(actor_id) <> ''),
    user_id text NOT NULL CHECK (btrim(user_id) <> ''),
    action text NOT NULL CHECK (action IN (
        'USER_BLOCKED', 'USER_UNBLOCKED', 'USER_ROLES_CHANGED', 'USER_SESSIONS_TERMINATED'
    )),
    details jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object')
);

CREATE FUNCTION protect_login_history()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'Историю входов нельзя изменять или удалять';
END;
$$;

CREATE TRIGGER protect_login_history
BEFORE UPDATE OR DELETE ON user_login
FOR EACH ROW EXECUTE FUNCTION protect_login_history();

CREATE FUNCTION protect_user_audit()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'Журнал аудита User нельзя изменять или удалять';
END;
$$;

CREATE TRIGGER protect_user_audit
BEFORE UPDATE OR DELETE ON audit_log
FOR EACH ROW EXECUTE FUNCTION protect_user_audit();

COMMIT;
