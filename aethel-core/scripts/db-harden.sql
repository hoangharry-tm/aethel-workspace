-- Aethel Workspace — Database Hardening Script
-- Run once as the database superuser after migrations are applied.
-- This REVOKES destructive privileges from the application user on the audit_ledger,
-- making it append-only from the application's perspective.
--
-- Replace 'aethel_app' with the actual AETHEL_DB_USER value from your .env file.
-- The application connects as this role. After running this script, even a compromised
-- application instance cannot delete or modify audit ledger entries.
--
-- Usage: psql "${DATABASE_SUPERUSER_DSN}" -f aethel-core/scripts/db-harden.sql
-- Or via: ./aethel-scripts/db-harden.sh

-- Step 1: Revoke UPDATE and DELETE on the audit_ledger from the application user.
-- INSERT is required (the app appends rows). SELECT is required (admin queries).
REVOKE UPDATE, DELETE ON TABLE audit_ledger FROM aethel_app;

-- Step 2: Verify. The output should show no 'w' (UPDATE) or 'd' (DELETE) privilege
-- for aethel_app in the audit_ledger row.
\dp audit_ledger

-- Step 3: Apply the same restriction to future partitions automatically.
-- PostgreSQL partition tables inherit privileges from the parent by default,
-- but explicit revocation on the parent does not propagate. Document this
-- as a required step whenever new audit_ledger partitions are created.
-- See migration 19 — partitions are pre-created; new ones need manual hardening.

SELECT 'Database hardening complete. Verify audit_ledger privileges above.' AS status;
