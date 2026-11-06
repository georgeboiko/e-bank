BEGIN;

CREATE OR REPLACE FUNCTION record_login(
    p_event_id uuid,
    p_sub text,
    p_ip inet,
    p_at timestamptz DEFAULT now()
)
RETURNS integer LANGUAGE plpgsql AS $$
DECLARE
    login_id integer;
BEGIN
    INSERT INTO user_login (keycloak_sub, logined_at, ip)
    VALUES (p_sub, p_at, p_ip)
    RETURNING id INTO login_id;

    INSERT INTO outbox_event (event_id, user_login_id, payload)
    VALUES (p_event_id, login_id, jsonb_build_object(
        'eventId', p_event_id,
        'eventType', 'LoginSucceeded',
        'occurredAt', p_at,
        'data', jsonb_build_object('userId', p_sub, 'ipAddress', host(p_ip))
    ));

    RETURN login_id;
END;
$$;

COMMIT;
