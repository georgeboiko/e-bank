# Kafka-контракты E-Bank

Структуру сообщений задают JSON Schema. В примерах показаны полные сообщения, которые получит сервис.

| Событие | Направление | Топик | Ключ Kafka | Схема и примеры |
| --- | --- | --- | --- | --- |
| `LoginSucceeded` | User → Antifraud | `user.security-events` | `data.userId` | [Схема](schemas/login-succeeded.schema.json), [пример](examples/login-succeeded.json) |
| `FraudCheckRequested` | Money → Antifraud | `money.fraud-check.requested` | `data.transferId` как десятичная строка | [Схема](schemas/fraud-check-requested.schema.json), [пример](examples/fraud-check-requested.json) |
| `FraudDecision` | Antifraud → Money | `antifraud.decision.made` | `data.transferId` как десятичная строка | [Схема](schemas/fraud-decision.schema.json), [автоматическое решение](examples/fraud-decision-automatic.json), [решение аналитика](examples/fraud-decision-analyst.json) |

Ключ передаётся отдельно от JSON в UTF-8. Отправитель использует ключ, чтобы направлять связанные события в одну партицию топика. Kafka сохраняет порядок записи внутри партиции.

## Формат сообщения

Все поля общей оболочки обязательны.

| Поле | Значение |
| --- | --- |
| `eventId` | UUIDv7 логического события. Генерируется один раз при создании записи Outbox. |
| `eventType` | Имя события из таблицы выше. |
| `occurredAt` | Время бизнес-события в RFC 3339 с `Z` или числовым смещением от UTC. |
| `data` | Объект с данными события. |

- **`LoginSucceeded`** передаёт `userId` (Keycloak `sub`) и IP клиента для истории входов. IP поступает от доверенного Nginx и передаётся без маски подсети.
- **`FraudCheckRequested`** создаётся при резервировании суммы и переходе перевода в `FRAUD_CHECK_PENDING`. Событие содержит ID перевода и счетов, Keycloak `sub` их владельцев и сумму в рублях JSON-числом с точностью до копейки, например `1000.50`. `deviceId` приходит от клиента. Поля `deviceId` и `ipAddress` передаются как `null`, если данных нет. `data.createdAt` хранит время создания перевода в том же формате, что и `occurredAt`.
- **`FraudDecision`** передаёт окончательное решение `APPROVED` или `REJECTED` и его источник `AUTOMATIC` или `ANALYST`. `riskScore` и `triggeredRules` отражают автоматическую проверку, в том числе после ручного разбора. Для каждого сработавшего правила передаются `ruleId`, начисленные баллы `score` и объяснение `reason`. При `REVIEW_REQUIRED` перевод остаётся в `FRAUD_CHECK_PENDING`, а событие публикуется после решения аналитика.

## Доставка и совместимость

Обмен работает по модели at least once, поэтому сообщения могут приходить повторно.

- **Outbox.** Отправитель сохраняет событие в одной транзакции с бизнес-изменением и отмечает его опубликованным после подтверждения Kafka. При повторной отправке Kafka key и JSON, включая `eventId`, остаются прежними.
- **Inbox.** Получатель проверяет сообщение и в одной транзакции сохраняет исходный JSON с уникальным `eventId`, применяет новое событие и отмечает его обработанным. Offset подтверждается после фиксации транзакции. Повторное событие не применяется. Если тот же `eventId` пришёл с изменёнными данными, получатель отклоняет сообщение и записывает расхождение в журнал.

## Go-модели

В общем пакете `internal/contracts/kafka` UUIDv7 представлен строкой, сумма декодируется в `decimal.Decimal`, а временные метки — в `time.Time`. `deviceId` и `ipAddress` в запросе проверки могут быть `null`. Решение без сработавших правил содержит `triggeredRules: []`.