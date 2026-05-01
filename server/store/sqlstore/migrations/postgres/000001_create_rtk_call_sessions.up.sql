CREATE TABLE IF NOT EXISTS rtk_call_sessions (
    id                     VARCHAR(26) NOT NULL,
    channel_id             VARCHAR(26) NOT NULL,
    creator_id             VARCHAR(26) NOT NULL,
    meeting_id             VARCHAR(64) NOT NULL,
    createat               BIGINT      NOT NULL,
    updateat               BIGINT      NOT NULL,
    endat                  BIGINT      NOT NULL DEFAULT 0,
    post_id                VARCHAR(26) NOT NULL UNIQUE,
    rtk_channel_meeting_id VARCHAR(26) NOT NULL,
    session_id             VARCHAR(64) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

-- Cloudflare RTK session UUIDs are globally unique.
-- Before the first webhook arrives the column holds an empty string, so the partial unique index excludes empty strings to prevent duplicate entries.
CREATE UNIQUE INDEX IF NOT EXISTS rtk_call_sessions_session_id_unique
    ON rtk_call_sessions (session_id)
    WHERE session_id <> '';

-- At most one active call (endat = 0) per channel.
CREATE UNIQUE INDEX IF NOT EXISTS rtk_call_sessions_active_channel_unique
    ON rtk_call_sessions (channel_id)
    WHERE endat = 0;

-- At most one active call (endat = 0) per RTK meeting.
CREATE UNIQUE INDEX IF NOT EXISTS rtk_call_sessions_active_meeting_unique
    ON rtk_call_sessions (meeting_id)
    WHERE endat = 0;
