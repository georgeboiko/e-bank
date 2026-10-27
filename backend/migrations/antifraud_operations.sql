BEGIN;

CREATE OR REPLACE FUNCTION save_antifraud_check(
    p_transfer_id integer,
    p_scores integer[],
    p_statuses text[],
    p_reasons text[],
    p_event_id uuid
)
RETURNS integer LANGUAGE plpgsql AS $$
DECLARE
    check_id_created integer;
    total_score integer;
    decision text;
    saved_scores integer[];
    saved_statuses text[];
    saved_reasons text[];
    rule_number integer;
    operation_at timestamptz;
BEGIN
    IF array_dims(p_scores) IS DISTINCT FROM '[1:8]'
       OR array_dims(p_statuses) IS DISTINCT FROM '[1:8]'
       OR array_dims(p_reasons) IS DISTINCT FROM '[1:8]' THEN
        RAISE EXCEPTION 'Нужны три массива с элементами 1..8: баллы, статусы и причины';
    END IF;

    PERFORM id FROM transfer_attempt WHERE id = p_transfer_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Попытка перевода % не найдена', p_transfer_id;
    END IF;
    SELECT id INTO check_id_created FROM antifraud_check WHERE transfer_id = p_transfer_id;
    IF FOUND THEN
        SELECT array_agg(score ORDER BY rule_id), array_agg(status ORDER BY rule_id),
               array_agg(reason ORDER BY rule_id)
        INTO saved_scores, saved_statuses, saved_reasons
        FROM rule_evaluation WHERE check_id = check_id_created;
        IF ROW(saved_scores, saved_statuses, saved_reasons)
           IS DISTINCT FROM ROW(p_scores, p_statuses, p_reasons) THEN
            RAISE EXCEPTION 'Для перевода % уже сохранены другие результаты проверки', p_transfer_id;
        END IF;
        RETURN check_id_created;
    END IF;

    total_score := least(100, greatest(p_scores[1], p_scores[4])
        + p_scores[2] + p_scores[3] + p_scores[5]
        + p_scores[6] + p_scores[7] + p_scores[8]);
    decision := CASE WHEN total_score < 30 THEN 'APPROVED'
                     WHEN total_score < 70 THEN 'REVIEW_REQUIRED'
                     ELSE 'REJECTED' END;
    operation_at := clock_timestamp();

    INSERT INTO antifraud_check (transfer_id, risk_score, automatic_decision, checked_at)
    VALUES (p_transfer_id, total_score, decision, operation_at)
    RETURNING id INTO check_id_created;
    FOR rule_number IN 1..8 LOOP
        INSERT INTO rule_evaluation (check_id, rule_id, status, score, reason)
        VALUES (check_id_created, rule_number, p_statuses[rule_number],
                p_scores[rule_number], p_reasons[rule_number]);
    END LOOP;

    IF decision = 'REVIEW_REQUIRED' THEN
        INSERT INTO review_case (check_id, created_at) VALUES (check_id_created, operation_at);
    ELSE
        INSERT INTO outbox_event (event_id, check_id, payload, created_at)
        VALUES (p_event_id, check_id_created, jsonb_build_object(
            'eventId', p_event_id, 'eventType', 'FraudDecision', 'occurredAt', operation_at,
            'data', jsonb_build_object(
                'transferId', p_transfer_id, 'decision', decision, 'riskScore', total_score,
                'source', 'AUTOMATIC'
            )
        ), operation_at);
    END IF;
    INSERT INTO audit_log (occurred_at, actor_type, action, check_id, details)
    VALUES (operation_at, 'SYSTEM', 'CHECK_COMPLETED', check_id_created,
            jsonb_build_object('riskScore', total_score, 'automaticDecision', decision));

    RETURN check_id_created;
END;
$$;

CREATE OR REPLACE PROCEDURE resolve_review_case(
    p_case_id integer,
    p_analyst_id text,
    p_decision text,
    p_comment text,
    p_event_id uuid
)
LANGUAGE plpgsql AS $$
DECLARE
    current_case review_case%ROWTYPE;
    current_check antifraud_check%ROWTYPE;
    operation_at timestamptz;
BEGIN
    IF p_decision IS NULL OR p_decision NOT IN ('APPROVED', 'REJECTED')
       OR p_analyst_id IS NULL OR btrim(p_analyst_id) = '' THEN
        RAISE EXCEPTION 'Нужны идентификатор аналитика и решение APPROVED либо REJECTED';
    END IF;
    SELECT * INTO current_case FROM review_case WHERE id = p_case_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Ручной разбор % не найден', p_case_id;
    END IF;
    IF current_case.status = 'CLOSED' THEN
        IF ROW(current_case.analyst_id, current_case.decision, current_case.comment)
           IS DISTINCT FROM ROW(p_analyst_id, p_decision, p_comment) THEN
            RAISE EXCEPTION 'Разбор % уже закрыт с другими данными', p_case_id;
        END IF;
        RETURN;
    END IF;
    IF current_case.analyst_id IS NOT NULL AND current_case.analyst_id <> p_analyst_id THEN
        RAISE EXCEPTION 'Разбор % назначен другому аналитику', p_case_id;
    END IF;

    INSERT INTO app_user (id) VALUES (p_analyst_id) ON CONFLICT DO NOTHING;
    SELECT * INTO current_check FROM antifraud_check WHERE id = current_case.check_id;
    operation_at := clock_timestamp();
    UPDATE review_case SET status = 'CLOSED', analyst_id = p_analyst_id,
        decision = p_decision, comment = p_comment, decided_at = operation_at
    WHERE id = p_case_id;

    INSERT INTO outbox_event (event_id, check_id, payload, created_at)
    VALUES (p_event_id, current_check.id, jsonb_build_object(
        'eventId', p_event_id, 'eventType', 'FraudDecision', 'occurredAt', operation_at,
        'data', jsonb_build_object(
            'transferId', current_check.transfer_id, 'decision', p_decision,
            'riskScore', current_check.risk_score, 'source', 'ANALYST'
        )
    ), operation_at);
    INSERT INTO audit_log (occurred_at, actor_type, actor_id, action, check_id, details)
    VALUES (operation_at, 'ANALYST', p_analyst_id, 'REVIEW_CLOSED', current_check.id,
            jsonb_build_object('reviewCaseId', p_case_id, 'decision', p_decision, 'comment', p_comment));
END;
$$;

COMMIT;
