BEGIN;

ALTER TABLE profile
    DROP COLUMN banned;

ALTER TABLE pdv
    DROP COLUMN reward;

DROP TABLE pdv_rewards_distributed_date;

COMMIT;