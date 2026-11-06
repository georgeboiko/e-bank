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

CREATE FUNCTION protect_login_history()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'Историю входов нельзя изменять или удалять';
END;
$$;

CREATE TRIGGER protect_login_history
BEFORE UPDATE OR DELETE ON user_login
FOR EACH ROW EXECUTE FUNCTION protect_login_history();

COMMIT;
