-- Migration 22 UP: Seed the default single-tenant organization.
-- This ensures app.OrgID always exists for bootstrapping and runtime.

INSERT INTO {{ .Schema }}.{{ T "organizations" }} (
    id,
    name,
    slug,
    is_active,
    created_at,
    updated_at
)
VALUES (
    '8f1f6db2-7f89-4c0d-8f61-59dc7d3e7b91',
    'Aethel Organization',
    'aethel',
    true,
    now(),
    now()
)
ON CONFLICT (slug) DO NOTHING;