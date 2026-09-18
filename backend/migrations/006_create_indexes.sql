-- 006: search + game indexes.
-- Search strategy: case-insensitive partial match via ILIKE '%q%' backed by a
-- pg_trgm GIN index. This avoids Elasticsearch and is plenty for a roster of
-- hundreds/thousands of fighters. If the roster grows large, switch the query
-- to similarity(name, $1) ordering; the same index serves both.
CREATE INDEX idx_fighters_name_trgm ON fighters USING gin (name gin_trgm_ops);
CREATE INDEX idx_fighters_name_lower ON fighters (lower(name));
CREATE INDEX idx_fighters_nationality ON fighters (nationality);
CREATE INDEX idx_events_date ON events (date DESC);
