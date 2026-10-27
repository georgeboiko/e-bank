#!/bin/sh
set -eu
cd "$(dirname "$0")"
. ../.env.test

# Аргументы: пользователь, пароль, порт, имя БД.
drop_database() {
    PGPASSWORD="$2" psql -X -h 127.0.0.1 -p "$3" -U "$1" -d postgres \
        -v ON_ERROR_STOP=1 -v db_name="$4" <<'SQL'
DROP DATABASE IF EXISTS :"db_name";
SQL
}

drop_database "$USER_DB_USER" "$USER_DB_PASSWORD" 5433 "$USER_DB_NAME"
drop_database "$MONEY_DB_USER" "$MONEY_DB_PASSWORD" 5434 "$MONEY_DB_NAME"
drop_database "$ANTIFRAUD_DB_USER" "$ANTIFRAUD_DB_PASSWORD" 5435 "$ANTIFRAUD_DB_NAME"
