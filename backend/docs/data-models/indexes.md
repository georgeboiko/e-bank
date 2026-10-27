# Индексы и их обоснование

Индексы выбраны по прецедентам этапа 1 и правилам антифрода.

Исходные требования: [прецеденты и архитектура](../srs/srs.pdf), [правила антифрода](../design-docs/antifraud.pdf).

## Принцип выбора

Используется обычный B-tree. В составных индексах сначала указано поле, по которому выбирается объект, затем время: например, `(from_account_id, created_at)`. Это соответствует запросу «переводы этого счёта за период».

Для очередей используются частичные индексы: в них попадают только неопубликованные события, переводы на исполнение или незакрытые ручные проверки.

## User

| Индекс | Поля и условие | Прецедент и польза |
| --- | --- | --- |
| `outbox_event_pending_idx` | `(next_attempt_at, created_at) WHERE published = false` | UC-1: поиск готовых к отправке событий входа. Publisher выбирает записи, время очередной попытки которых уже наступило. Опубликованные события исключены. |

## Money

| Индекс | Поля и условие | Прецедент и польза |
| --- | --- | --- |
| `account_owner_idx` | `(owner_id)` | UC-2, UC-4: получение всех счетов клиента и подсчёт их количества перед открытием нового счёта. |
| `transfer_from_created_idx` | `(from_account_id, created_at DESC)` | UC-4: исходящие переводы выбранного счёта за период, последние операции в начале списка. |
| `transfer_to_created_idx` | `(to_account_id, created_at DESC)` | UC-4: входящие переводы выбранного счёта, включая пополнения. |
| `transfer_created_idx` | `(created_at)` | UC-7: выбор переводов за период без ограничения по одному счёту для подсчёта количества, сумм и распределения по статусам. |
| `transfer_processing_idx` | `(id) WHERE state IN ('COMPLETING', 'RELEASING_FUNDS')` | UC-3, UC-8: воркер находит операции, для которых нужно вызвать `commit_transfer` или `release_funds`, в том числе после перезапуска сервиса. |
| `ledger_entry_account_created_idx` | `(account_id, created_at DESC)` | UC-4: история проводок счёта за период. Имеющийся `UNIQUE (transfer_id, account_id)` обслуживает поиск по переводу, поэтому для истории по счёту нужен другой порядок полей. |
| `audit_log_occurred_idx` | `(occurred_at DESC)` | UC-9: просмотр административных действий за период и последних записей аудита. |
| `outbox_event_pending_idx` | `(next_attempt_at, created_at) WHERE published = false` | UC-3: выбор запросов на антифрод-проверку, готовых к первой или повторной отправке в Kafka. |

## Antifraud

| Индекс | Поля и условие | Прецедент и польза |
| --- | --- | --- |
| `bank_account_owner_idx` | `(owner_id)` | UC-5, UC-6: получение всех счетов клиента. Правила, определённые на уровне клиента, должны учитывать его попытки со всех счетов. |
| `transfer_attempt_from_created_idx` | `(from_account_id, created_at DESC)` | R1, R2, R4, R7: исходящие попытки за период; история сумм; получатели; поиск следующего ребра при обходе графа. |
| `transfer_attempt_to_created_idx` | `(to_account_id, created_at DESC)` | R5, R7: входящие попытки конкретного счёта и обход входящих рёбер графа. Владельцы счетов определяются через `bank_account`. |
| `transfer_attempt_device_created_idx` | `(device, created_at DESC) WHERE device IS NOT NULL` | R6: попытки с одного устройства за сутки; после выборки подсчитываются разные владельцы счетов отправителей. Попытки без устройства в индекс не включены. |
| `login_event_user_occurred_idx` | `(user_id, occurred_at DESC)` | R8: последний успешный вход клиента перед переводом в пределах двух часов. Координаты берутся из найденной записи. |
| `antifraud_check_checked_idx` | `(checked_at)` | UC-7: статистика решений за период. Для статистики правил к выбранным проверкам присоединяются их `rule_evaluation`. |
| `review_case_pending_idx` | `(created_at, id) WHERE status IN ('OPEN', 'IN_PROGRESS')` | UC-6: очередь незакрытых разборов от старых к новым. Закрытые карточки исключены. |
| `audit_log_check_occurred_idx` | `(check_id, occurred_at DESC)` | UC-6: история действий по выбранной проверке, включая автоматический результат и решение аналитика. |
| `audit_log_occurred_idx` | `(occurred_at DESC)` | UC-9: общий журнал действий за период, без ограничения одной проверкой. |
| `outbox_event_pending_idx` | `(next_attempt_at, created_at) WHERE published = false` | UC-5, UC-6: отправка окончательных автоматических и ручных решений в Money. |

## Индексы, уже созданные ограничениями

PostgreSQL автоматически создаёт уникальные B-tree-индексы для PRIMARY KEY и UNIQUE. Они уже входят в модель и повторно не создаются:

- первичные ключи: поиск счетов, переводов, проверок, карточек и событий по ID;
- Money: `transfer(idempotency_key)`, `inbox_event(transfer_id)`, `outbox_event(transfer_id)`, `external_payment(external_payment_id)` и `external_payment(transfer_id)` для повторных запросов и платежей;
- Money: `ledger_entry(transfer_id, account_id)` для получения проводок перевода и проверки их целостности;
- Antifraud: `user_device(user_id, device)`, `antifraud_check(transfer_id)`, `review_case(check_id)`, `outbox_event(check_id)` для связей между попыткой, проверкой, разбором и решением;
- Antifraud: `rule_evaluation(check_id, rule_id)` для результатов восьми правил конкретной проверки и проверки итогового score;
- User: `outbox_event(user_login_id)` для связи события с записью входа.
