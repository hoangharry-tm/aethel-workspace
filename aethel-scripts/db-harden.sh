#!/usr/bin/env bash
# Usage: DATABASE_SUPERUSER_DSN="postgres://superuser:pass@localhost/aethel" ./db-harden.sh
# Applies db-harden.sql as the superuser. Run once after initial database setup.
# Removes UPDATE/DELETE privileges on audit_ledger from the application role.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -z "${DATABASE_SUPERUSER_DSN:-}" ]]; then
  echo "ERROR: DATABASE_SUPERUSER_DSN is not set." >&2
  echo "       Export it before running this script." >&2
  exit 1
fi

echo "Applying database hardening..."
psql "${DATABASE_SUPERUSER_DSN}" -f "${SCRIPT_DIR}/../aethel-core/scripts/db-harden.sql"
echo "Database hardening complete."
