BEGIN;

CREATE TABLE pdv (
    owner TEXT NOT NULL,
    id BIGINT NOT NULL,
    tx TEXT NOT NULL,
    meta JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT current_timestamp,

    PRIMARY KEY (owner, id)
);

COMMIT;