ALTER TABLE proxy.products RENAME COLUMN provider_id TO old_provider_id;
ALTER TABLE proxy.products ADD COLUMN provider_ids uuid[] NOT NULL DEFAULT '{}';
UPDATE proxy.products SET provider_ids = ARRAY[old_provider_id];
ALTER TABLE proxy.products DROP COLUMN old_provider_id;
