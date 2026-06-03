-- Migration 22 DOWN: Remove seeded organization.

DELETE FROM {{ .Schema }}.{{ T "organizations" }}
WHERE id = '8f1f6db2-7f89-4c0d-8f61-59dc7d3e7b91';