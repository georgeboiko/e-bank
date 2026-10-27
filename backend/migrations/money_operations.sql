BEGIN;

CREATE OR REPLACE FUNCTION create_transfer(
    p_actor_id text,
    p_from_account_id integer,
    p_to_account_id integer,
    p_amount numeric,
    p_idempotency_key uuid,
    p_event_id uuid,
    p_device_id text DEFAULT NULL,
    p_ip_address inet DEFAULT NULL
)
RETURNS integer LANGUAGE plpgsql AS $$
DECLARE
    sender account%ROWTYPE;
    recipient account%ROWTYPE;
    existing_transfer transfer%ROWTYPE;
    transfer_id_created integer;
    operation_at timestamptz;
BEGIN
    IF p_amount IS NULL OR p_amount <= 0 OR p_amount = 'NaN'::numeric
       OR p_amount <> round(p_amount, 2) THEN
        RAISE EXCEPTION 'Сумма должна быть положительной и содержать не более двух дробных знаков';
    END IF;
    IF p_from_account_id = p_to_account_id THEN
        RAISE EXCEPTION 'Счета отправителя и получателя должны различаться';
    END IF;

    PERFORM id FROM account
    WHERE id IN (p_from_account_id, p_to_account_id)
    ORDER BY id FOR UPDATE;

    SELECT * INTO sender FROM account WHERE id = p_from_account_id;
    IF NOT FOUND OR sender.type <> 'CLIENT'
       OR sender.owner_id IS DISTINCT FROM p_actor_id THEN
        RAISE EXCEPTION 'Клиентский счёт отправителя не найден или принадлежит другому пользователю';
    END IF;
    SELECT * INTO recipient FROM account WHERE id = p_to_account_id;
    IF NOT FOUND OR recipient.type <> 'CLIENT' THEN
        RAISE EXCEPTION 'Клиентский счёт получателя не найден';
    END IF;

    SELECT * INTO existing_transfer FROM transfer WHERE idempotency_key = p_idempotency_key;
    IF NOT FOUND THEN
        operation_at := clock_timestamp();
        INSERT INTO transfer (
            from_account_id, to_account_id, amount, idempotency_key,
            device_id, ip_address, created_at, updated_at
        ) VALUES (
            p_from_account_id, p_to_account_id, p_amount, p_idempotency_key,
            p_device_id, p_ip_address, operation_at, operation_at
        )
        ON CONFLICT (idempotency_key) DO NOTHING
        RETURNING id INTO transfer_id_created;
        IF transfer_id_created IS NULL THEN
            SELECT * INTO existing_transfer FROM transfer
            WHERE idempotency_key = p_idempotency_key;
        END IF;
    END IF;

    IF transfer_id_created IS NULL THEN
        IF ROW(existing_transfer.from_account_id, existing_transfer.to_account_id,
               existing_transfer.amount, existing_transfer.device_id, existing_transfer.ip_address)
           IS DISTINCT FROM
           ROW(p_from_account_id, p_to_account_id, p_amount, p_device_id, p_ip_address) THEN
            RAISE EXCEPTION 'Ключ идемпотентности уже использован для другого запроса';
        END IF;
        RETURN existing_transfer.id;
    END IF;

    IF sender.state <> 'ACTIVE' OR recipient.state <> 'ACTIVE' THEN
        RAISE EXCEPTION 'Оба счёта должны быть активны';
    END IF;
    IF sender.balance - sender.reserved_sum < p_amount THEN
        RAISE EXCEPTION 'Недостаточно доступных средств';
    END IF;

    UPDATE account SET reserved_sum = reserved_sum + p_amount WHERE id = sender.id;
    UPDATE transfer SET state = 'FRAUD_CHECK_PENDING', updated_at = operation_at
    WHERE id = transfer_id_created;

    INSERT INTO outbox_event (event_id, transfer_id, payload, created_at)
    VALUES (p_event_id, transfer_id_created, jsonb_build_object(
        'eventId', p_event_id, 'eventType', 'TransferRequested', 'occurredAt', operation_at,
        'data', jsonb_build_object(
            'transferId', transfer_id_created,
            'fromAccountId', sender.id, 'toAccountId', recipient.id,
            'senderId', sender.owner_id, 'recipientId', recipient.owner_id,
            'amount', p_amount, 'deviceId', p_device_id,
            'ipAddress', host(p_ip_address), 'createdAt', operation_at
        )
    ), operation_at);

    RETURN transfer_id_created;
END;
$$;

CREATE OR REPLACE PROCEDURE apply_antifraud_decision(
    p_event_id uuid,
    p_transfer_id integer,
    p_decision text,
    p_risk_score integer,
    p_occurred_at timestamptz
)
LANGUAGE plpgsql AS $$
DECLARE
    current_transfer transfer%ROWTYPE;
    received_event inbox_event%ROWTYPE;
    inserted_event_id uuid;
    event_payload jsonb;
    operation_at timestamptz;
BEGIN
    IF p_decision IS NULL OR p_decision NOT IN ('APPROVED', 'REJECTED')
       OR p_risk_score IS NULL OR p_risk_score NOT BETWEEN 0 AND 100
       OR p_occurred_at IS NULL THEN
        RAISE EXCEPTION 'Требуется окончательное решение с баллом 0–100 и временем события';
    END IF;

    SELECT * INTO current_transfer FROM transfer WHERE id = p_transfer_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Перевод % не найден', p_transfer_id;
    END IF;

    event_payload := jsonb_build_object(
        'eventId', p_event_id, 'eventType', 'AntifraudDecision', 'occurredAt', p_occurred_at,
        'data', jsonb_build_object(
            'transferId', p_transfer_id, 'decision', p_decision, 'riskScore', p_risk_score
        )
    );
    INSERT INTO inbox_event (event_id, transfer_id, payload)
    VALUES (p_event_id, p_transfer_id, event_payload)
    ON CONFLICT DO NOTHING RETURNING event_id INTO inserted_event_id;

    IF inserted_event_id IS NULL THEN
        SELECT * INTO received_event FROM inbox_event WHERE event_id = p_event_id;
        IF NOT FOUND OR received_event.transfer_id IS DISTINCT FROM p_transfer_id
           OR received_event.payload->>'eventType' IS DISTINCT FROM 'AntifraudDecision'
           OR received_event.payload->'data'->>'decision' IS DISTINCT FROM p_decision
           OR (received_event.payload->'data'->>'riskScore')::integer IS DISTINCT FROM p_risk_score
           OR (received_event.payload->>'occurredAt')::timestamptz IS DISTINCT FROM p_occurred_at THEN
            RAISE EXCEPTION 'Событие или перевод уже зарегистрированы с другими данными';
        END IF;
        IF received_event.applied THEN
            RETURN;
        END IF;
    END IF;

    IF current_transfer.state <> 'FRAUD_CHECK_PENDING' THEN
        RAISE EXCEPTION 'Перевод % не ожидает решения антифрода', p_transfer_id;
    END IF;

    operation_at := clock_timestamp();

    IF p_decision = 'APPROVED' THEN
        UPDATE transfer SET state = 'COMPLETING', updated_at = operation_at
        WHERE id = p_transfer_id;
    ELSE
        UPDATE transfer SET state = 'RELEASING_FUNDS', reject_cause = 'ANTIFRAUD_REJECTED',
            updated_at = operation_at WHERE id = p_transfer_id;
    END IF;

    UPDATE inbox_event SET applied = true, processed_at = operation_at
    WHERE event_id = p_event_id;
END;
$$;

CREATE OR REPLACE FUNCTION commit_transfer(p_transfer_id integer)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    current_transfer transfer%ROWTYPE;
    sender account%ROWTYPE;
    recipient account%ROWTYPE;
    operation_at timestamptz;
BEGIN
    SELECT * INTO current_transfer FROM transfer WHERE id = p_transfer_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Перевод % не найден', p_transfer_id;
    END IF;
    IF current_transfer.state = 'COMPLETED' THEN
        RETURN;
    END IF;
    IF current_transfer.state <> 'COMPLETING' THEN
        RAISE EXCEPTION 'Перевод % не находится в COMPLETING', p_transfer_id;
    END IF;

    PERFORM id FROM account
    WHERE id IN (current_transfer.from_account_id, current_transfer.to_account_id)
    ORDER BY id FOR UPDATE;
    SELECT * INTO sender FROM account WHERE id = current_transfer.from_account_id;
    SELECT * INTO recipient FROM account WHERE id = current_transfer.to_account_id;
    IF sender.state <> 'ACTIVE' OR recipient.state <> 'ACTIVE' THEN
        RAISE EXCEPTION 'Для исполнения перевода оба счёта должны быть активны';
    END IF;
    IF sender.reserved_sum < current_transfer.amount THEN
        RAISE EXCEPTION 'Для перевода % недостаточно зарезервированных средств', p_transfer_id;
    END IF;
    operation_at := clock_timestamp();

    UPDATE account SET balance = balance - current_transfer.amount,
        reserved_sum = reserved_sum - current_transfer.amount WHERE id = sender.id;
    UPDATE account SET balance = balance + current_transfer.amount WHERE id = recipient.id;
    INSERT INTO ledger_entry (transfer_id, account_id, amount, created_at) VALUES
        (p_transfer_id, sender.id, -current_transfer.amount, operation_at),
        (p_transfer_id, recipient.id, current_transfer.amount, operation_at);
    UPDATE transfer SET state = 'COMPLETED', updated_at = operation_at,
        finished_at = operation_at WHERE id = p_transfer_id;
END;
$$;

CREATE OR REPLACE FUNCTION release_funds(p_transfer_id integer)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    current_transfer transfer%ROWTYPE;
    sender account%ROWTYPE;
    operation_at timestamptz;
BEGIN
    SELECT * INTO current_transfer FROM transfer WHERE id = p_transfer_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Перевод % не найден', p_transfer_id;
    END IF;
    IF current_transfer.state = 'REJECTED' THEN
        RETURN;
    END IF;
    IF current_transfer.state <> 'RELEASING_FUNDS' THEN
        RAISE EXCEPTION 'Перевод % не находится в RELEASING_FUNDS', p_transfer_id;
    END IF;

    SELECT * INTO sender FROM account
    WHERE id = current_transfer.from_account_id FOR UPDATE;
    IF sender.reserved_sum < current_transfer.amount THEN
        RAISE EXCEPTION 'Для перевода % недостаточно зарезервированных средств', p_transfer_id;
    END IF;
    operation_at := clock_timestamp();

    UPDATE account SET reserved_sum = reserved_sum - current_transfer.amount
    WHERE id = sender.id;
    UPDATE transfer SET state = 'REJECTED', updated_at = operation_at,
        finished_at = operation_at WHERE id = p_transfer_id;
END;
$$;

CREATE OR REPLACE FUNCTION confirm_topup(
    p_external_payment_id text,
    p_technical_account_id integer,
    p_to_account_id integer,
    p_amount numeric,
    p_idempotency_key uuid
)
RETURNS integer LANGUAGE plpgsql AS $$
DECLARE
    sender account%ROWTYPE;
    recipient account%ROWTYPE;
    existing_transfer transfer%ROWTYPE;
    transfer_id_created integer;
    operation_at timestamptz;
BEGIN
    IF p_amount IS NULL OR p_amount <= 0 OR p_amount = 'NaN'::numeric
       OR p_amount <> round(p_amount, 2) THEN
        RAISE EXCEPTION 'Сумма должна быть положительной и содержать не более двух дробных знаков';
    END IF;
    IF p_external_payment_id IS NULL OR btrim(p_external_payment_id) = '' THEN
        RAISE EXCEPTION 'Нужен идентификатор внешнего платежа';
    END IF;

    PERFORM id FROM account
    WHERE id IN (p_technical_account_id, p_to_account_id)
    ORDER BY id FOR UPDATE;
    SELECT * INTO sender FROM account WHERE id = p_technical_account_id;
    IF NOT FOUND OR sender.type <> 'TECHNICAL' THEN
        RAISE EXCEPTION 'Технический счёт не найден';
    END IF;
    SELECT * INTO recipient FROM account WHERE id = p_to_account_id;
    IF NOT FOUND OR recipient.type <> 'CLIENT' THEN
        RAISE EXCEPTION 'Клиентский счёт получателя не найден';
    END IF;

    SELECT t.* INTO existing_transfer FROM external_payment e
    JOIN transfer t ON t.id = e.transfer_id
    WHERE e.external_payment_id = p_external_payment_id;
    IF FOUND THEN
        IF ROW(existing_transfer.from_account_id, existing_transfer.to_account_id, existing_transfer.amount)
           IS DISTINCT FROM ROW(p_technical_account_id, p_to_account_id, p_amount) THEN
            RAISE EXCEPTION 'Внешний платёж уже связан с другим пополнением';
        END IF;
        IF existing_transfer.state NOT IN ('COMPLETING', 'COMPLETED') THEN
            RAISE EXCEPTION 'Внешний платёж связан с пополнением в неожиданном состоянии';
        END IF;
        RETURN existing_transfer.id;
    END IF;

    IF sender.state <> 'ACTIVE' OR recipient.state <> 'ACTIVE' THEN
        RAISE EXCEPTION 'Оба счёта должны быть активны';
    END IF;
    IF sender.balance - sender.reserved_sum < p_amount THEN
        RAISE EXCEPTION 'Недостаточно средств на техническом счёте';
    END IF;
    operation_at := clock_timestamp();

    INSERT INTO transfer (
        from_account_id, to_account_id, amount, idempotency_key, created_at, updated_at
    ) VALUES (
        sender.id, recipient.id, p_amount, p_idempotency_key, operation_at, operation_at
    ) RETURNING id INTO transfer_id_created;
    INSERT INTO external_payment (transfer_id, amount, external_payment_id, created_at)
    VALUES (transfer_id_created, p_amount, p_external_payment_id, operation_at);

    UPDATE account SET reserved_sum = reserved_sum + p_amount WHERE id = sender.id;
    UPDATE transfer SET state = 'COMPLETING', updated_at = operation_at
    WHERE id = transfer_id_created;

    RETURN transfer_id_created;
END;
$$;

COMMIT;
