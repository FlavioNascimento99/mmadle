-- 015: fighter activity status (gameplay-neutral; used by roster/views only).
ALTER TABLE fighters ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active';