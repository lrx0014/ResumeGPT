#!/usr/bin/env bash
set -euo pipefail

backup_dir="${1:-.data/backups}"
mkdir -p "${backup_dir}"
backup_path="${backup_dir}/resumegpt-$(date -u +%Y%m%dT%H%M%SZ).dump"

docker compose exec -T postgres pg_dump \
  --username=resumegpt \
  --dbname=resumegpt \
  --format=custom \
  --no-owner \
  --no-privileges >"${backup_path}"

printf '%s\n' "${backup_path}"
