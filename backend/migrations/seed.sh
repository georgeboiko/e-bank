#!/bin/sh
set -eu
cd "$(dirname "$0")"
. ../.env.test

PGPASSWORD="$USER_DB_PASSWORD" psql -X -h 127.0.0.1 -p 5433 \
    -U "$USER_DB_USER" -d "$USER_DB_NAME" -v ON_ERROR_STOP=1 -f user_seed.sql
PGPASSWORD="$MONEY_DB_PASSWORD" psql -X -h 127.0.0.1 -p 5434 \
    -U "$MONEY_DB_USER" -d "$MONEY_DB_NAME" -v ON_ERROR_STOP=1 -f money_seed.sql
PGPASSWORD="$ANTIFRAUD_DB_PASSWORD" psql -X -h 127.0.0.1 -p 5435 \
    -U "$ANTIFRAUD_DB_USER" -d "$ANTIFRAUD_DB_NAME" -v ON_ERROR_STOP=1 -f antifraud_seed.sql
