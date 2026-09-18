-- 002: divisions (canonical weight classes)
CREATE TABLE divisions (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    weight_limit_lb INTEGER
);
