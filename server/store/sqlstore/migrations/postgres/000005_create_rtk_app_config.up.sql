CREATE TABLE IF NOT EXISTS rtk_app_config (
    id         VARCHAR(26) PRIMARY KEY,
    account_id TEXT      NOT NULL,
    app_id     TEXT      NOT NULL UNIQUE,
    createat   BIGINT    NOT NULL
);
