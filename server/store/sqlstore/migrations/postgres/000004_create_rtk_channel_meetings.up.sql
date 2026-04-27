CREATE TABLE IF NOT EXISTS rtk_channel_meetings (
    channel_id    VARCHAR(26) NOT NULL,
    meeting_id    TEXT        NOT NULL,
    app_config_id VARCHAR(26) NOT NULL,
    createat      BIGINT      NOT NULL,
    updateat      BIGINT      NOT NULL,
    PRIMARY KEY (channel_id)
);
