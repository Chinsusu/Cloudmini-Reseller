-- Migration: 000010_add_vpm_provider.sql
-- Adds the VPS Proxy Manager (VPM) as a proxy provider in Cloudmini.
-- adapter_type = 'vpm' maps to the Go adapter registered in main.go.
-- The api_key field is intentionally empty — set via Admin UI or DB update after
-- creating an API key on VPM at POST /api/v1/api-keys.

INSERT INTO proxy.providers (
    id,
    name,
    display_name,
    adapter_type,
    config,
    is_active,
    priority
) VALUES (
    'b2000000-0000-0000-0000-000000000002',
    'vpm',
    'VPS Proxy Manager',
    'vpm',
    '{"base_url":"http://192.168.1.62:8080","api_key":""}',
    true,
    10
)
ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    adapter_type = EXCLUDED.adapter_type,
    is_active    = EXCLUDED.is_active,
    priority     = EXCLUDED.priority,
    updated_at   = NOW();
