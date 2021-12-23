BEGIN;

CREATE TABLE pdv_rewards_distributed_date (
    date TIMESTAMP NOT NULL
);

INSERT INTO pdv_rewards_distributed_date(date) VALUES (CURRENT_TIMESTAMP);

ALTER TABLE profile
    ADD COLUMN banned BOOL NOT NULL DEFAULT FALSE;

ALTER TABLE pdv
    ADD COLUMN reward DECIMAL NOT NULL DEFAULT 0;

COMMIT;