BEGIN;

CREATE TABLE app_user (
    id text PRIMARY KEY CHECK (btrim(id) <> '')
);

CREATE TABLE bank_account (
    id integer PRIMARY KEY CHECK (id > 0),
    owner_id text NOT NULL REFERENCES app_user(id)
);

CREATE TABLE user_device (
    id serial PRIMARY KEY,
    user_id text NOT NULL REFERENCES app_user(id),
    device text NOT NULL CHECK (btrim(device) <> ''),
    first_seen_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    use_count integer NOT NULL DEFAULT 1 CHECK (use_count > 0),
    UNIQUE (user_id, device),
    CHECK (last_seen_at >= first_seen_at)
);

CREATE TABLE login_event (
    id serial PRIMARY KEY,
    user_id text NOT NULL REFERENCES app_user(id),
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
    device text CHECK (btrim(device) <> ''),
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
    analyst_id text REFERENCES app_user(id),
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
    actor_id text REFERENCES app_user(id),
    action text NOT NULL CHECK (btrim(action) <> ''),
    check_id integer REFERENCES antifraud_check(id),
    details jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object'),
    CHECK (
        (actor_type = 'SYSTEM' AND actor_id IS NULL)
        OR (actor_type <> 'SYSTEM' AND actor_id IS NOT NULL)
    )
);

CREATE FUNCTION protect_antifraud_history()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND NEW IS NOT DISTINCT FROM OLD THEN
        RETURN NEW;
    END IF;

    RAISE EXCEPTION 'Записи таблицы % нельзя изменять или удалять', TG_TABLE_NAME;
END;
$$;

CREATE TRIGGER protect_bank_account_details
BEFORE UPDATE OR DELETE ON bank_account
FOR EACH ROW EXECUTE FUNCTION protect_antifraud_history();

CREATE TRIGGER protect_check_history
BEFORE UPDATE OR DELETE ON antifraud_check
FOR EACH ROW EXECUTE FUNCTION protect_antifraud_history();

CREATE TRIGGER protect_rule_history
BEFORE UPDATE OR DELETE ON rule_evaluation
FOR EACH ROW EXECUTE FUNCTION protect_antifraud_history();

CREATE TRIGGER protect_audit_history
BEFORE UPDATE OR DELETE ON audit_log
FOR EACH ROW EXECUTE FUNCTION protect_antifraud_history();

-- Проверка и результаты всех восьми правил сохраняются одной транзакцией.
-- Для отсутствующих данных также нужна строка: SKIPPED с нулевым score.
CREATE FUNCTION validate_check_results()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    check_id_to_validate integer;
    stored_score integer;
    rule_count bigint;
    calculated_score integer;
BEGIN
    IF TG_TABLE_NAME = 'antifraud_check' THEN
        check_id_to_validate := NEW.id;
    ELSE
        check_id_to_validate := NEW.check_id;
    END IF;

    SELECT risk_score INTO stored_score FROM antifraud_check
    WHERE id = check_id_to_validate;

    SELECT count(*), least(100,
        coalesce(max(CASE WHEN rule_id IN (1, 4) THEN score ELSE 0 END), 0)
        + coalesce(sum(CASE WHEN rule_id NOT IN (1, 4) THEN score ELSE 0 END), 0)
    ) INTO rule_count, calculated_score
    FROM rule_evaluation WHERE check_id = check_id_to_validate;

    IF rule_count <> 8 THEN
        RAISE EXCEPTION 'Для проверки % нужны результаты всех восьми правил',
            check_id_to_validate;
    END IF;

    IF stored_score <> calculated_score THEN
        RAISE EXCEPTION 'Баллы проверки % не соответствуют результатам правил: % вместо %',
            check_id_to_validate, stored_score, calculated_score;
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER validate_check_results
AFTER INSERT ON antifraud_check
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_check_results();

CREATE CONSTRAINT TRIGGER validate_rule_results
AFTER INSERT ON rule_evaluation
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_check_results();

CREATE FUNCTION validate_review_case()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM antifraud_check
        WHERE id = NEW.check_id AND automatic_decision = 'REVIEW_REQUIRED'
    ) THEN
        RAISE EXCEPTION 'Ручной разбор допустим только для REVIEW_REQUIRED';
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'OPEN' THEN
            RAISE EXCEPTION 'Новый разбор должен иметь статус OPEN';
        END IF;
        RETURN NEW;
    END IF;

    IF ROW(NEW.id, NEW.check_id, NEW.created_at)
       IS DISTINCT FROM ROW(OLD.id, OLD.check_id, OLD.created_at) THEN
        RAISE EXCEPTION 'Нельзя менять идентификатор, проверку и дату создания разбора';
    END IF;

    IF OLD.status = 'CLOSED' AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION 'Закрытый разбор нельзя изменять';
    END IF;

    IF NEW.status <> OLD.status AND NOT (
        (OLD.status = 'OPEN' AND NEW.status IN ('IN_PROGRESS', 'CLOSED'))
        OR (OLD.status = 'IN_PROGRESS' AND NEW.status = 'CLOSED')
    ) THEN
        RAISE EXCEPTION 'Недопустимый переход разбора: % -> %', OLD.status, NEW.status;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER validate_review_case
BEFORE INSERT OR UPDATE ON review_case
FOR EACH ROW EXECUTE FUNCTION validate_review_case();

CREATE TRIGGER protect_review_history
BEFORE DELETE ON review_case
FOR EACH ROW EXECUTE FUNCTION protect_antifraud_history();

CREATE FUNCTION validate_final_decision_event()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF ROW(NEW.event_id, NEW.check_id, NEW.payload, NEW.created_at)
           IS DISTINCT FROM ROW(OLD.event_id, OLD.check_id, OLD.payload, OLD.created_at) THEN
            RAISE EXCEPTION 'Содержимое события решения нельзя изменять';
        END IF;
        RETURN NEW;
    END IF;

    IF EXISTS (
        SELECT 1 FROM antifraud_check
        WHERE id = NEW.check_id AND automatic_decision = 'REVIEW_REQUIRED'
    ) AND NOT EXISTS (
        SELECT 1 FROM review_case WHERE check_id = NEW.check_id AND status = 'CLOSED'
    ) THEN
        RAISE EXCEPTION 'Событие окончательного решения требует закрытого ручного разбора';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER validate_final_decision_event
BEFORE INSERT OR UPDATE ON outbox_event
FOR EACH ROW EXECUTE FUNCTION validate_final_decision_event();

COMMIT;
