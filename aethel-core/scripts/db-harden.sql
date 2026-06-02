-- Run once as the database superuser after migrations are applied.
-- Removes the application role's ability to modify the audit ledger,
-- making tamper-evidence guarantees enforceable at the database layer.
--
-- Usage: psql $DATABASE_SUPERUSER_DSN -f db-harden.sql
-- Replace 'aethel_app' with the actual role from AETHEL_DB_USER in server-database.yaml.

REVOKE UPDATE, DELETE ON TABLE audit_ledger FROM aethel_app;

-- Verify — should show no UPDATE or DELETE privilege for aethel_app:
\dp audit_ledger
