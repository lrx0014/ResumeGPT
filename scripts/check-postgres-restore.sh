#!/usr/bin/env bash
set -euo pipefail

restore_database="resumegpt_restore_check"

cleanup() {
  docker compose exec -T postgres dropdb \
    --username=resumegpt \
    --if-exists \
    "${restore_database}" >/dev/null
}
trap cleanup EXIT

cleanup
docker compose exec -T postgres createdb \
  --username=resumegpt \
  "${restore_database}"

docker compose exec -T postgres pg_dump \
  --username=resumegpt \
  --dbname=resumegpt \
  --format=custom \
  --no-owner \
  --no-privileges |
  docker compose exec -T postgres pg_restore \
    --username=resumegpt \
    --dbname="${restore_database}" \
    --exit-on-error \
    --no-owner \
    --no-privileges

restored_tables="$(docker compose exec -T postgres psql \
  --username=resumegpt \
  --dbname="${restore_database}" \
  --tuples-only \
  --no-align \
  --command="SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('workspaces', 'profiles', 'jobs', 'durable_jobs', 'outbox_events', 'audit_events');")"

if [[ "${restored_tables}" != "6" ]]; then
  printf 'Restore verification failed: expected 6 core tables, found %s.\n' "${restored_tables}" >&2
  exit 1
fi

printf 'PostgreSQL backup and restore check passed.\n'
