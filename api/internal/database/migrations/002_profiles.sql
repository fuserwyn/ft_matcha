CREATE TABLE IF NOT EXISTS profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio TEXT,
    gender VARCHAR(20),
    sexual_preference VARCHAR(20),
    birth_date DATE,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    fame_rating INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
