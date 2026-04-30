CREATE TABLE IF NOT EXISTS rtk_app_config (
    id         VARCHAR(26) PRIMARY KEY,
    account_id TEXT      NOT NULL,
    app_id     TEXT      NOT NULL UNIQUE,
    status     TEXT      NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    createat   BIGINT    NOT NULL,
    updateat   BIGINT    NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS rtk_app_config_one_active
    ON rtk_app_config (status)
    WHERE status = 'active';
