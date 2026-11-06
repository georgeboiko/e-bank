BEGIN;

INSERT INTO user_login (id, keycloak_sub, logined_at, ip)
SELECT 1000000 + n,
       'bulk-user-' || (1 + (n - 1) % 10000),
       timestamptz '2026-11-01 00:00:00+03' - n * interval '1 minute',
       inet '10.0.0.1' + (n % 65535)
FROM generate_series(1, 200000) AS n;

INSERT INTO outbox_event (
    event_id, user_login_id, payload, created_at, published,
    attempt_count, next_attempt_at, published_at, last_error
)
SELECT uuidv7(), id, '{}'::jsonb, logined_at, id % 100 <> 0, 1,
       CASE WHEN id % 100 <> 0 THEN logined_at
            WHEN id % 200 = 0 THEN now() + interval '5 minutes'
            ELSE now() - interval '1 minute' END,
       CASE WHEN id % 100 <> 0 THEN logined_at + interval '1 second' ELSE NULL END,
       CASE WHEN id % 100 = 0 THEN 'Тестовая повторная публикация' ELSE NULL END
FROM user_login
WHERE id BETWEEN 1000001 AND 1200000;

SELECT setval('user_login_id_seq', (SELECT max(id) FROM user_login));

COMMIT;

ANALYZE user_login;
ANALYZE outbox_event;
