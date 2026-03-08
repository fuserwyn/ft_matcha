-- Convert sexual_preference from VARCHAR(20) to TEXT[] only if still VARCHAR
DO $$
BEGIN
    IF (SELECT data_type FROM information_schema.columns
        WHERE table_name = 'profiles' AND column_name = 'sexual_preference') = 'character varying' THEN
        ALTER TABLE profiles
            ALTER COLUMN sexual_preference TYPE TEXT[]
            USING CASE
                WHEN sexual_preference = 'male'   THEN ARRAY['male']
                WHEN sexual_preference = 'female' THEN ARRAY['female']
                WHEN sexual_preference = 'both'   THEN ARRAY['male', 'female']
                WHEN sexual_preference = 'other'  THEN ARRAY['non-binary', 'other']
                ELSE NULL
            END;
    END IF;
END$$;

-- Add relationship goal field
ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS relationship_goal VARCHAR(20);
