CREATE TABLE IF NOT EXISTS rtk_call_sessions (
    id            VARCHAR(26) NOT NULL,
    channel_id    VARCHAR(26) NOT NULL,
    creator_id    VARCHAR(26) NOT NULL,
    meeting_id    VARCHAR(64) NOT NULL,
    participants  TEXT        NOT NULL,
    createat      BIGINT      NOT NULL,
    updateat      BIGINT      NOT NULL,
    endat         BIGINT      NOT NULL DEFAULT 0,
    post_id       VARCHAR(26) NOT NULL,
    app_config_id VARCHAR(26) NOT NULL,
    session_id    VARCHAR(64) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);
