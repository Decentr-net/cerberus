CREATE TABLE rewards_queue
(
    address    TEXT PRIMARY KEY,
    reward     BIGINT    NOT NULL CHECK (reward > 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);