# E-Bank Backend

Проект информационной системы электронного банка: банковские счета, переводы, пополнения и проверка операций на мошенничество.

Архитектура предусматривает три сервиса:

- **User** — интеграция с Keycloak, фиксация успешных входов и отправка событий безопасности.
- **Money** — счета, балансы, резервирование средств, переводы, пополнения и проводки.
- **Antifraud** — проверка попыток переводов по восьми правилам, анализ связей и ручной разбор подозрительных операций.

У каждого сервиса отдельный экземпляр PostgreSQL. События между сервисами передаются через Kafka; для надёжной доставки используются Outbox и Inbox.

## Документация

- [docs/api](docs/api/) — HTTP-контракты OpenAPI и просмотр в Swagger UI
- [docs/reports](docs/reports/) — отчеты по этапам работы
- [docs/design-docs](docs/design-docs/) — дизайн-документы конкретных механик
- [docs/data-models](docs/data-models/) — модели данных и обоснование индексов

## Структура migrations

- `scripts/` — создание и удаление БД, загрузка обычных сидов.
- `schema/` — таблицы, ограничения, триггеры, функции и индексы.
- `seeds/` — обычные и массовые тестовые данные.
- `checks/` — сравнение запросов с индексами и без них через `EXPLAIN ANALYZE`.

## Локальный запуск БД

Нужны запущенный Docker с Docker Compose и установленный клиент PostgreSQL (`psql`). Все команды выполнять из каталога `backend`. Имена БД, пользователи и пароли берутся из `.env.test`.

1. Поднять три экземпляра PostgreSQL и дождаться их готовности:

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
