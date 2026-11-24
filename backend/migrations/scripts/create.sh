#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
. ../.env.test

# Аргументы: пользователь, пароль, порт, имя БД и файлы SQL.
create_database() {
    PGPASSWORD="$2" psql -X -h 127.0.0.1 -p "$3" -U "$1" -d postgres \
        -v ON_ERROR_STOP=1 -v db_name="$4" <<'SQL'
SELECT format('CREATE DATABASE %I', :'db_name')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'db_name')
\gexec
SQL

    db_user=$1 db_password=$2 db_port=$3 db_name=$4
    shift 4
    for sql_file do
        PGPASSWORD="$db_password" psql -X -h 127.0.0.1 -p "$db_port" -U "$db_user" -d "$db_name" \
            -v ON_ERROR_STOP=1 -f "$sql_file"
    done
}

create_database "$USER_DB_USER" "$USER_DB_PASSWORD" 5433 "$USER_DB_NAME" schema/user.sql schema/user_operations.sql schema/user_indexes.sql
create_database "$MONEY_DB_USER" "$MONEY_DB_PASSWORD" 5434 "$MONEY_DB_NAME" schema/money.sql schema/money_operations.sql schema/money_indexes.sql schema/money_reporting.sql
create_database "$ANTIFRAUD_DB_USER" "$ANTIFRAUD_DB_PASSWORD" 5435 "$ANTIFRAUD_DB_NAME" schema/antifraud.sql schema/antifraud_operations.sql schema/antifraud_indexes.sql schema/antifraud_rule_settings.sql
