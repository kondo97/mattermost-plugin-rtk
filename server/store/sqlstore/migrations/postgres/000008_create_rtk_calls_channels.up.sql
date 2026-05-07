-- rtk_calls_channels mirrors the row layout of the Calls plugin's `calls_channels` table
-- so that data can be migrated 1:1 on first activation. The table records, per-channel,
-- whether RTK calls are explicitly enabled and any opaque props (e.g. ringing settings)
-- carried over from Calls. Returned by GET /api/v1/channels for client compatibility.
CREATE TABLE IF NOT EXISTS rtk_calls_channels (
    channel_id VARCHAR(26) NOT NULL,
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    props      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (channel_id)
);
