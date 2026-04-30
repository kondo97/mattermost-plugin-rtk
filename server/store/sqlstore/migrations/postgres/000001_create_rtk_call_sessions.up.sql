CREATE TABLE IF NOT EXISTS rtk_call_sessions (
    id                 VARCHAR(26) NOT NULL,
    channel_id         VARCHAR(26) NOT NULL,
    creator_id         VARCHAR(26) NOT NULL,
    meeting_id         VARCHAR(64) NOT NULL,
    participants       TEXT        NOT NULL,
    createat           BIGINT      NOT NULL,
    updateat           BIGINT      NOT NULL,
    endat              BIGINT      NOT NULL DEFAULT 0,
    post_id            VARCHAR(26) NOT NULL UNIQUE,
    rtk_channel_meeting_id VARCHAR(26) NOT NULL,
    session_id         VARCHAR(64) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

-- Cloudflare RTK の session UUID はグローバルにユニーク。
-- Webhook 受信前は空文字のため、空文字を除外した partial unique で重複登録を防ぐ。
CREATE UNIQUE INDEX IF NOT EXISTS rtk_call_sessions_session_id_unique
    ON rtk_call_sessions (session_id)
    WHERE session_id <> '';

-- 1 チャンネルにつき進行中 (endat = 0) の通話は最大 1 件。
CREATE UNIQUE INDEX IF NOT EXISTS rtk_call_sessions_active_channel_unique
    ON rtk_call_sessions (channel_id)
    WHERE endat = 0;

-- 1 RTK ミーティングにつき進行中 (endat = 0) の通話は最大 1 件。
CREATE UNIQUE INDEX IF NOT EXISTS rtk_call_sessions_active_meeting_unique
    ON rtk_call_sessions (meeting_id)
    WHERE endat = 0;
