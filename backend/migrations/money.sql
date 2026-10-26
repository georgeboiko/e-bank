BEGIN;

CREATE TABLE account (
    id serial PRIMARY KEY,
    owner_id text,
    type text NOT NULL CHECK (type IN ('CLIENT', 'TECHNICAL')),
    state text NOT NULL DEFAULT 'ACTIVE'
        CHECK (state IN ('ACTIVE', 'BLOCKED', 'CLOSED')),
    balance numeric(19,2) NOT NULL DEFAULT 0,
    reserved_sum numeric(19,2) NOT NULL DEFAULT 0,
    opened_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (type = 'CLIENT' AND owner_id IS NOT NULL AND btrim(owner_id) <> '')
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
    device_id text CHECK (btrim(device_id) <> ''),
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
    actor_id text,
    action text NOT NULL CHECK (action IN (
        'ACCOUNT_OPENED', 'ACCOUNT_BLOCKED', 'ACCOUNT_UNBLOCKED',
        'ACCOUNT_CLOSED', 'LIMIT_CHANGED'
    )),
    account_id integer REFERENCES account(id),
    details jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object'),
    CHECK (
        (actor_type = 'SYSTEM' AND actor_id IS NULL)
        OR (actor_type <> 'SYSTEM' AND actor_id IS NOT NULL AND btrim(actor_id) <> '')
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

CREATE FUNCTION protect_account_details()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF ROW(NEW.id, NEW.owner_id, NEW.type, NEW.opened_at)
       IS DISTINCT FROM ROW(OLD.id, OLD.owner_id, OLD.type, OLD.opened_at) THEN
        RAISE EXCEPTION 'Нельзя менять идентификатор, владельца, тип и дату открытия счёта';
    END IF;

    IF OLD.state = 'CLOSED' AND NEW.state <> 'CLOSED' THEN
        RAISE EXCEPTION 'Закрытый счёт нельзя открыть повторно';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER protect_account_details
BEFORE UPDATE ON account
FOR EACH ROW EXECUTE FUNCTION protect_account_details();

CREATE FUNCTION validate_transfer_change()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.state <> 'PENDING' THEN
            RAISE EXCEPTION 'Новый перевод должен иметь состояние PENDING';
        END IF;
        RETURN NEW;
    END IF;

    IF ROW(NEW.id, NEW.from_account_id, NEW.to_account_id, NEW.amount,
           NEW.idempotency_key, NEW.device_id, NEW.ip_address, NEW.created_at)
       IS DISTINCT FROM
       ROW(OLD.id, OLD.from_account_id, OLD.to_account_id, OLD.amount,
           OLD.idempotency_key, OLD.device_id, OLD.ip_address, OLD.created_at) THEN
        RAISE EXCEPTION 'Реквизиты созданного перевода нельзя изменять';
    END IF;

    IF OLD.state IN ('COMPLETED', 'REJECTED', 'CANCELLED')
       AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION 'Завершённый перевод нельзя изменять';
    END IF;

    IF NEW.state <> OLD.state AND NOT (
        (OLD.state = 'PENDING'
            AND NEW.state IN ('FRAUD_CHECK_PENDING', 'COMPLETING', 'CANCELLED'))
        OR (OLD.state = 'FRAUD_CHECK_PENDING'
            AND NEW.state IN ('COMPLETING', 'RELEASING_FUNDS', 'CANCELLED'))
        OR (OLD.state = 'COMPLETING' AND NEW.state = 'COMPLETED')
        OR (OLD.state = 'RELEASING_FUNDS' AND NEW.state = 'REJECTED')
    ) THEN
        RAISE EXCEPTION 'Недопустимый переход перевода: % -> %', OLD.state, NEW.state;
    END IF;

    IF OLD.state = 'PENDING' AND NEW.state = 'COMPLETING'
       AND NOT EXISTS (
           SELECT 1 FROM account
           WHERE id = NEW.from_account_id AND type = 'TECHNICAL'
       ) THEN
        RAISE EXCEPTION 'Клиентский перевод должен пройти этап FRAUD_CHECK_PENDING';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER validate_transfer_change
BEFORE INSERT OR UPDATE ON transfer
FOR EACH ROW EXECUTE FUNCTION validate_transfer_change();

CREATE FUNCTION protect_money_history()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'Записи таблицы % нельзя изменять или удалять', TG_TABLE_NAME;
END;
$$;

CREATE TRIGGER protect_ledger_history
BEFORE UPDATE OR DELETE ON ledger_entry
FOR EACH ROW EXECUTE FUNCTION protect_money_history();

CREATE TRIGGER protect_external_payment_history
BEFORE UPDATE OR DELETE ON external_payment
FOR EACH ROW EXECUTE FUNCTION protect_money_history();

CREATE TRIGGER protect_audit_history
BEFORE UPDATE OR DELETE ON audit_log
FOR EACH ROW EXECUTE FUNCTION protect_money_history();

-- Завершение перевода и две проводки сохраняются одной транзакцией.
-- Подтверждение пополнения к этому моменту уже должно существовать.
-- Проверяем итоговое состояние перед COMMIT, когда все записи уже добавлены.
-- Обновление balance и reserved_sum выполняет сервис в той же транзакции:
-- этот триггер сам деньги не списывает и не зачисляет.
CREATE FUNCTION validate_transfer_accounting()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    transfer_id_to_check integer;
    current_transfer transfer%ROWTYPE;
    sender_type text;
    recipient_type text;
    entry_count bigint;
    debit_count bigint;
    credit_count bigint;
    payment_amount numeric(19,2);
BEGIN
    IF TG_TABLE_NAME = 'transfer' THEN
        transfer_id_to_check := NEW.id;
    ELSE
        transfer_id_to_check := NEW.transfer_id;
    END IF;

    SELECT * INTO current_transfer FROM transfer WHERE id = transfer_id_to_check;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    SELECT count(*),
           count(*) FILTER (WHERE account_id = current_transfer.from_account_id
                             AND amount = -current_transfer.amount),
           count(*) FILTER (WHERE account_id = current_transfer.to_account_id
                             AND amount = current_transfer.amount)
    INTO entry_count, debit_count, credit_count
    FROM ledger_entry WHERE transfer_id = transfer_id_to_check;

    IF current_transfer.state = 'COMPLETED' THEN
        IF entry_count <> 2 OR debit_count <> 1 OR credit_count <> 1 THEN
            RAISE EXCEPTION 'Перевод % должен иметь две проводки: -amount и +amount',
                transfer_id_to_check;
        END IF;
    ELSIF entry_count <> 0 THEN
        RAISE EXCEPTION 'Проводки допустимы только для выполненного перевода %',
            transfer_id_to_check;
    END IF;

    SELECT type INTO sender_type FROM account WHERE id = current_transfer.from_account_id;
    SELECT type INTO recipient_type FROM account WHERE id = current_transfer.to_account_id;
    SELECT amount INTO payment_amount FROM external_payment
    WHERE transfer_id = transfer_id_to_check;

    IF payment_amount IS NOT NULL AND (
        sender_type <> 'TECHNICAL' OR recipient_type <> 'CLIENT'
        OR payment_amount <> current_transfer.amount
    ) THEN
        RAISE EXCEPTION 'Подтверждение пополнения не соответствует переводу %',
            transfer_id_to_check;
    END IF;

    IF current_transfer.state = 'COMPLETED' AND sender_type = 'TECHNICAL'
       AND payment_amount IS NULL THEN
        RAISE EXCEPTION 'Для пополнения % требуется подтверждение внешнего платежа',
            transfer_id_to_check;
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER validate_transfer_accounting
AFTER INSERT OR UPDATE ON transfer
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_transfer_accounting();

CREATE CONSTRAINT TRIGGER validate_ledger_accounting
AFTER INSERT ON ledger_entry
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_transfer_accounting();

CREATE CONSTRAINT TRIGGER validate_external_payment_accounting
AFTER INSERT ON external_payment
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_transfer_accounting();

COMMIT;
