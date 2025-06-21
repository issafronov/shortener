CREATE TABLE urls (
    id SERIAL PRIMARY KEY,
    short_url TEXT NOT NULL UNIQUE,
    original_url TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE
);
