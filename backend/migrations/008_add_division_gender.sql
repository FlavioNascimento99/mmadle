-- 008: division gender, so games can be limited to one pool (e.g. men only).
-- A fighter's pool follows their current division.
ALTER TABLE divisions ADD COLUMN gender TEXT CHECK (gender IN ('men', 'women'));
UPDATE divisions SET gender = CASE WHEN name LIKE 'Women''s %' THEN 'women' ELSE 'men' END;
ALTER TABLE divisions ALTER COLUMN gender SET NOT NULL;
