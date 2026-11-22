# E-Bank Backend

Проект информационной системы электронного банка: банковские счета, переводы, пополнения и проверка операций на мошенничество.

Архитектура предусматривает три сервиса:

- **User** — интеграция с Keycloak, фиксация успешных входов и отправка событий безопасности.
- **Money** — счета, балансы, резервирование средств, переводы, пополнения и проводки.
- **Antifraud** — проверка попыток переводов по восьми правилам, анализ связей и ручной разбор подозрительных операций.

У каждого сервиса отдельный экземпляр PostgreSQL. События между сервисами передаются через Kafka; для надёжной доставки используются Outbox и Inbox.

Стек — Go, Chi, pgxpool, PostgreSQL и Kafka.

## Структура Go-модуля

- `cmd/{user,money,antifraud}` — точки входа сервисов.
- `internal/{user,money,antifraud}` — код отдельных сервисов.
- `internal/infrastructure` — общая конфигурация, HTTP-сервер и пул PostgreSQL.

## HTTP-порты сервисов

При локальном запуске используются следующие порты:

| Сервис | HTTP-порт | Переменная окружения для адреса |
| --- | --- | --- |
| User | `8080` | `USER_HTTP_ADDR` |
| Money | `8081` | `MONEY_HTTP_ADDR` |
| Antifraud | `8082` | `ANTIFRAUD_HTTP_ADDR` |

По умолчанию сервисы слушают `localhost`. Полный адрес можно переопределить указанной переменной окружения.

## Сборка и запуск Go-сервисов

Из каталога `backend` проверь сборку командой `go build ./cmd/...`.

Для запуска [подними локальные БД и Kafka](#локальный-запуск-бд-и-kafka). В `.env.test` для каждого префикса `USER`, `MONEY`, `ANTIFRAUD` должны быть заданы `*_DB_ADDR`, `*_DB_USER`, `*_DB_PASSWORD`, `*_DB_NAME`. Локальные адреса БД по порядку — `127.0.0.1:5433`, `127.0.0.1:5434`, `127.0.0.1:5435`.

В каждом терминале экспортируй настройки и запусти нужный сервис.

```sh
set -a
. ./.env.test
set +a
go run ./cmd/user  # или go run ./cmd/money, go run ./cmd/antifraud
```

`GET /health` доступен без токена, проверяет БД сервиса и возвращает `200 UP` или `503 DOWN`.

## Запись успешных входов

User читает успешные события `LOGIN` банковского клиента через [Keycloak Admin Events API](https://www.keycloak.org/docs-api/latest/rest-api/index.html). Используй Keycloak 26.4 или новее с идентификаторами событий и фильтрами по времени в миллисекундах. Включи сохранение `LOGIN` в настройках realm и задай срок хранения событий с запасом на простой User. В realm используется стандартный `sub`, который равен `userId` события.

- `KEYCLOAK_LOGIN_CLIENT_ID` — ID OIDC-клиента банка.
- `KEYCLOAK_EVENTS_CLIENT_ID` и `KEYCLOAK_EVENTS_CLIENT_SECRET` — отдельный конфиденциальный клиент с Service Accounts и ролью `realm-management.view-events`. Если обе переменные отсутствуют, User запускается без опроса.
- `KEYCLOAK_EVENTS_REALM_URL` — при необходимости внутренний URL того же realm для access token и Admin API. По умолчанию используется `KEYCLOAK_ISSUER`.

За Nginx задай для Keycloak `KC_PROXY_HEADERS=xforwarded` и `KC_PROXY_TRUSTED_ADDRESSES` с адресом Nginx. Ограничь прямой доступ клиентов к Keycloak. Действующий конфиг Nginx устанавливает `X-Forwarded-For` из адреса подключения. User берёт IP и время из события Keycloak, сохраняет вход и Outbox одной операцией `record_login`. При повторном чтении того же Keycloak event ID запись не дублируется.

## Nginx

При локальном запуске Nginx слушает `127.0.0.1:8088`. Запусти Go-сервисы на портах из таблицы выше, затем из каталога `backend` выполни:

```sh
nginx -t -c "$PWD/nginx/nginx.conf"
nginx -c "$PWD/nginx/nginx.conf" -g 'daemon off;'
```

Проверка проксирования Money — `curl -i http://localhost:8088/api/money/v1/health`. В OpenAPI укажи `origin=http://localhost:8088`.

Для внешнего запуска используй `nginx/nginx.prod.conf`. Укажи реальный `server_name` в обоих блоках и пути к сертификату и ключу. Этот конфиг перенаправляет HTTP на HTTPS и закрывает публичный доступ к `/health`.


## Документация

- [docs/api](docs/api/) — HTTP-контракты OpenAPI и просмотр в Swagger UI
- [Kafka-контракты](docs/api/kafkaContract/) — топики, схемы и примеры событий
- [docs/reports](docs/reports/) — отчеты по этапам работы
- [docs/design-docs](docs/design-docs/) — дизайн-документы конкретных механик
- [docs/data-models](docs/data-models/) — модели данных и обоснование индексов

## Структура migrations

- `scripts/` — создание и удаление БД, загрузка обычных сидов.
- `schema/` — таблицы, ограничения, триггеры, функции и индексы.
- `seeds/` — обычные и массовые тестовые данные.
- `checks/` — сравнение запросов с индексами и без них через `EXPLAIN ANALYZE`.

## Локальный запуск БД и Kafka

Нужны запущенный Docker с Docker Compose и установленный клиент PostgreSQL (`psql`). Все команды выполнять из каталога `backend`. Имена БД, пользователи и пароли берутся из `.env.test`.

1. Поднять локальное окружение:

   ```sh
   docker compose --env-file .env.test up -d --wait
   ```

2. Создать таблицы, ограничения, триггеры, функции и индексы:

   ```sh
   ./migrations/scripts/create.sh
   ```

3. Заполнить БД тестовыми данными:

   ```sh
   ./migrations/scripts/seed.sh
   ```

### Подключение к Kafka

С хоста используй `KAFKA_BROKERS=localhost:9092`, из контейнера в той же сети Compose — `KAFKA_BROKERS=kafka:29092`.

[Топики](docs/api/kafkaContract/) появляются при первом обращении клиента с включённым автосозданием. Данные Kafka хранятся в томе `kafka_data`.

Топики, сообщения и группы потребителей доступны в Kafka UI. [Открыть Kafka UI](http://localhost:8085)

### Массовые данные для проверки индексов

Обычный `seed.sh` загружает небольшой демонстрационный набор. Отдельные SQL-файлы создают большой набор через `generate_series`:

| Файл | Дополнительные данные |
| --- | --- |
| `migrations/seeds/user_seed_large.sql` | 200 000 входов и 200 000 событий Outbox. |
| `migrations/seeds/money_seed_large.sql` | 20 000 счетов, 100 000 переводов, 200 000 проводок и 100 000 внешних платежей. |
| `migrations/seeds/antifraud_seed_large.sql` | 100 000 попыток и проверок, 800 000 результатов правил, 100 000 входов и 100 000 записей Inbox. |

Выполнить один раз в тестовых БД после создания структуры. Если нужны обычные демонстрационные данные, загрузить их до массовых. Команды из каталога `backend`:

```sh
. ./.env.test
PGPASSWORD="$USER_DB_PASSWORD" psql -X -h 127.0.0.1 -p 5433 -U "$USER_DB_USER" -d "$USER_DB_NAME" -v ON_ERROR_STOP=1 -f migrations/seeds/user_seed_large.sql
PGPASSWORD="$MONEY_DB_PASSWORD" psql -X -h 127.0.0.1 -p 5434 -U "$MONEY_DB_USER" -d "$MONEY_DB_NAME" -v ON_ERROR_STOP=1 -f migrations/seeds/money_seed_large.sql
PGPASSWORD="$ANTIFRAUD_DB_PASSWORD" psql -X -h 127.0.0.1 -p 5435 -U "$ANTIFRAUD_DB_USER" -d "$ANTIFRAUD_DB_NAME" -v ON_ERROR_STOP=1 -f migrations/seeds/antifraud_seed_large.sql
```

### Сравнение запросов с индексами и без них

После загрузки массовых данных выполнить команды из каталога `backend`:

```sh
. ./.env.test
PGPASSWORD="$USER_DB_PASSWORD" psql -X -a -P pager=off -h 127.0.0.1 -p 5433 -U "$USER_DB_USER" -d "$USER_DB_NAME" -v ON_ERROR_STOP=1 -f migrations/checks/user_indexes_explain.sql
PGPASSWORD="$MONEY_DB_PASSWORD" psql -X -a -P pager=off -h 127.0.0.1 -p 5434 -U "$MONEY_DB_USER" -d "$MONEY_DB_NAME" -v ON_ERROR_STOP=1 -f migrations/checks/money_indexes_explain.sql
PGPASSWORD="$ANTIFRAUD_DB_PASSWORD" psql -X -a -P pager=off -h 127.0.0.1 -p 5435 -U "$ANTIFRAUD_DB_USER" -d "$ANTIFRAUD_DB_NAME" -v ON_ERROR_STOP=1 -f migrations/checks/antifraud_indexes_explain.sql
```

Файлы сравнивают все 19 добавленных индексов через `EXPLAIN ANALYZE`. Для каждого индекса выполняется запрос, затем индекс временно удаляется и повторяется тот же запрос. `ROLLBACK` восстанавливает индекс. Флаг `-a` выводит SQL и комментарии, чтобы было видно, к какому индексу относится план. Подробнее в [indexes.md](docs/data-models/indexes.md).

БД доступны на `127.0.0.1`: User — порт `5433`, Money — `5434`, Antifraud — `5435`.

`create.sh` и `seed.sh` предназначены для первоначальной инициализации пустых БД. При повторном запуске контейнеров данные сохраняются в Docker volumes; заново создавать таблицы и загружать тестовые данные не нужно.

## Тикеты

- Перед началом работы завести тикет с задачей.
- Использовать идентификатор `PIGGYBANK-BACKEND-<номер>`.
- В одном тикете решать одну связанную задачу.
- В тикете отмечать существенные решения и препятствия, влияющие на выполнение задачи.
- Закрывать тикет после слияния соответствующего Pull Request в `main`.

## Ветки и Pull Request

- `main` — общая ветка проекта; изменения в неё вносить через Pull Request.
- Для каждого тикета создавать отдельную ветку от актуальной `main`.
- Формат имени: `backend-<номер>_<описание>`. Описание писать на английском, слова разделять подчёркиваниями. Пример: `backend-2_add_sql`.
- В ветку включать изменения, относящиеся к её тикету.
- Перед слиянием устранить конфликты с `main` и проверить изменения. При изменении модели данных или поведения обновить соответствующую документацию.
