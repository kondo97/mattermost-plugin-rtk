CREATE TABLE IF NOT EXISTS rtk_webhook_config (
    id             VARCHAR(26) PRIMARY KEY,
    webhook_id     TEXT        NOT NULL,
    app_config_id  VARCHAR(26) NOT NULL,
    createat       BIGINT      NOT NULL
);
