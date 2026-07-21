ALTER TABLE usage_events
    ADD COLUMN IF NOT EXISTS cache_write_input_tokens bigint NOT NULL DEFAULT 0;

ALTER TABLE session_summaries
    ADD COLUMN IF NOT EXISTS cache_write_input_tokens bigint NOT NULL DEFAULT 0;

ALTER TABLE usage_rollups
    ADD COLUMN IF NOT EXISTS cache_write_input_tokens bigint NOT NULL DEFAULT 0;

ALTER TABLE session_rollups
    ADD COLUMN IF NOT EXISTS cache_write_input_tokens bigint NOT NULL DEFAULT 0;
