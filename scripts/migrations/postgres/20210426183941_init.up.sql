BEGIN;

CREATE TABLE height (
    height BIGINT
);

INSERT INTO height VALUES(0);

CREATE TABLE profile (
    address TEXT NOT NULL PRIMARY KEY,
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    emails TEXT[],
    bio TEXT NOT NULL DEFAULT '',
    avatar TEXT NOT NULL DEFAULT '',
    gender TEXT NOT NULL DEFAULT '',
    birthday DATE NOT NULL DEFAULT '1900-01-01',
    updated_at TIMESTAMP WITHOUT TIME ZONE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER profile_updated_at_trigger
BEFORE UPDATE ON profile
FOR EACH ROW
EXECUTE PROCEDURE set_updated_at();

COMMIT;