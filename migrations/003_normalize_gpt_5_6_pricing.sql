BEGIN;

CREATE TEMP TABLE affected_gpt_5_6_sessions ON COMMIT DROP AS
SELECT DISTINCT session_id
FROM usage_events
WHERE model IN ('gpt-5.6', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna');

CREATE TEMP TABLE affected_gpt_5_6_usage_dates ON COMMIT DROP AS
SELECT DISTINCT occurred_at::date AS bucket_date
FROM usage_events
WHERE model IN ('gpt-5.6', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna')
  AND occurred_at IS NOT NULL;

CREATE TEMP TABLE affected_gpt_5_6_session_dates ON COMMIT DROP AS
SELECT DISTINCT COALESCE(
    session_summaries.updated_at,
    session_summaries.started_at,
    sessions.updated_at,
    sessions.started_at,
    sessions.ingested_at
)::date AS bucket_date
FROM affected_gpt_5_6_sessions
JOIN sessions ON sessions.id = affected_gpt_5_6_sessions.session_id
LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id;

WITH token_events AS (
    SELECT
        session_id,
        seq,
        NULLIF(payload_jsonb #>> '{payload,info,last_token_usage,cache_write_input_tokens}', '')::bigint AS last_write_tokens,
        NULLIF(payload_jsonb #>> '{payload,info,total_token_usage,cache_write_input_tokens}', '')::bigint AS total_write_tokens
    FROM session_events
    WHERE payload_type = 'token_count'
),
token_deltas AS (
    SELECT
        token_events.*,
        COALESCE((
            SELECT prior.total_write_tokens
            FROM token_events AS prior
            WHERE prior.session_id = token_events.session_id
              AND prior.seq < token_events.seq
              AND prior.total_write_tokens IS NOT NULL
            ORDER BY prior.seq DESC
            LIMIT 1
        ), 0) AS previous_total_write_tokens
    FROM token_events
),
normalized_writes AS (
    SELECT
        usage_events.id,
        CASE
            WHEN token_deltas.last_write_tokens IS NOT NULL
                THEN GREATEST(token_deltas.last_write_tokens, 0)
            WHEN token_deltas.total_write_tokens IS NOT NULL
                THEN GREATEST(token_deltas.total_write_tokens - token_deltas.previous_total_write_tokens, 0)
            ELSE 0
        END AS cache_write_input_tokens
    FROM usage_events
    JOIN token_deltas
      ON token_deltas.session_id = usage_events.session_id
     AND token_deltas.seq = usage_events.seq
    WHERE usage_events.model IN ('gpt-5.6', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna')
)
UPDATE usage_events
SET cache_write_input_tokens = LEAST(
    normalized_writes.cache_write_input_tokens,
    GREATEST(usage_events.input_tokens - usage_events.cached_input_tokens, 0)
)
FROM normalized_writes
WHERE usage_events.id = normalized_writes.id;

WITH model_pricing(model, input_cost, cache_read_cost, cache_write_cost, output_cost) AS (
    VALUES
        ('gpt-5.6',       0.000005::numeric,  0.0000005::numeric,  0.00000625::numeric,  0.00003::numeric),
        ('gpt-5.6-sol',   0.000005::numeric,  0.0000005::numeric,  0.00000625::numeric,  0.00003::numeric),
        ('gpt-5.6-terra', 0.0000025::numeric, 0.00000025::numeric, 0.000003125::numeric, 0.000015::numeric),
        ('gpt-5.6-luna',  0.000001::numeric,  0.0000001::numeric,  0.00000125::numeric,  0.000006::numeric)
),
billable_tokens AS (
    SELECT
        usage_events.id,
        model_pricing.input_cost,
        model_pricing.cache_read_cost,
        model_pricing.cache_write_cost,
        model_pricing.output_cost,
        GREATEST(usage_events.input_tokens, 0) AS input_tokens,
        LEAST(
            GREATEST(usage_events.cached_input_tokens, 0),
            GREATEST(usage_events.input_tokens, 0)
        ) AS cache_read_tokens,
        LEAST(
            GREATEST(usage_events.cache_write_input_tokens, 0),
            GREATEST(usage_events.input_tokens - usage_events.cached_input_tokens, 0)
        ) AS cache_write_tokens,
        GREATEST(usage_events.output_tokens, 0) AS output_tokens
    FROM usage_events
    JOIN model_pricing ON model_pricing.model = usage_events.model
)
UPDATE usage_events
SET cost_usd =
    GREATEST(billable_tokens.input_tokens - billable_tokens.cache_read_tokens - billable_tokens.cache_write_tokens, 0)
        * billable_tokens.input_cost
    + billable_tokens.cache_read_tokens * billable_tokens.cache_read_cost
    + billable_tokens.cache_write_tokens * billable_tokens.cache_write_cost
    + billable_tokens.output_tokens * billable_tokens.output_cost
FROM billable_tokens
WHERE usage_events.id = billable_tokens.id;

UPDATE session_summaries
SET
    cache_write_input_tokens = usage_totals.cache_write_input_tokens,
    cost_usd = usage_totals.cost_usd,
    generated_at = now()
FROM (
    SELECT
        usage_events.session_id,
        COALESCE(sum(usage_events.cache_write_input_tokens), 0)::bigint AS cache_write_input_tokens,
        COALESCE(sum(usage_events.cost_usd), 0) AS cost_usd
    FROM usage_events
    JOIN affected_gpt_5_6_sessions
      ON affected_gpt_5_6_sessions.session_id = usage_events.session_id
    GROUP BY usage_events.session_id
) AS usage_totals
WHERE session_summaries.session_id = usage_totals.session_id;

DELETE FROM usage_rollups
WHERE bucket_date IN (SELECT bucket_date FROM affected_gpt_5_6_usage_dates);

INSERT INTO usage_rollups (
    bucket_date, bucket_month, device_id, repository_id, project_id, model,
    input_tokens, cached_input_tokens, cache_write_input_tokens,
    output_tokens, reasoning_output_tokens, total_tokens, cost_usd
)
SELECT
    usage_events.occurred_at::date,
    date_trunc('month', usage_events.occurred_at)::date,
    sessions.device_id,
    sessions.repository_id,
    sessions.project_id,
    usage_events.model,
    sum(usage_events.input_tokens)::bigint,
    sum(usage_events.cached_input_tokens)::bigint,
    sum(usage_events.cache_write_input_tokens)::bigint,
    sum(usage_events.output_tokens)::bigint,
    sum(usage_events.reasoning_output_tokens)::bigint,
    sum(usage_events.total_tokens)::bigint,
    sum(usage_events.cost_usd)
FROM usage_events
JOIN sessions ON sessions.id = usage_events.session_id
WHERE usage_events.occurred_at::date IN (SELECT bucket_date FROM affected_gpt_5_6_usage_dates)
GROUP BY
    usage_events.occurred_at::date,
    date_trunc('month', usage_events.occurred_at)::date,
    sessions.device_id,
    sessions.repository_id,
    sessions.project_id,
    usage_events.model;

DELETE FROM session_rollups
WHERE bucket_date IN (SELECT bucket_date FROM affected_gpt_5_6_session_dates);

INSERT INTO session_rollups (
    bucket_date, bucket_month, device_id, repository_id, project_id, session_count,
    input_tokens, cached_input_tokens, cache_write_input_tokens,
    output_tokens, reasoning_output_tokens, total_tokens, cost_usd, patch_added_lines
)
SELECT
    COALESCE(
        session_summaries.updated_at,
        session_summaries.started_at,
        sessions.updated_at,
        sessions.started_at,
        sessions.ingested_at
    )::date,
    date_trunc('month', COALESCE(
        session_summaries.updated_at,
        session_summaries.started_at,
        sessions.updated_at,
        sessions.started_at,
        sessions.ingested_at
    ))::date,
    sessions.device_id,
    sessions.repository_id,
    sessions.project_id,
    count(sessions.id)::bigint,
    COALESCE(sum(session_summaries.input_tokens), 0)::bigint,
    COALESCE(sum(session_summaries.cached_input_tokens), 0)::bigint,
    COALESCE(sum(session_summaries.cache_write_input_tokens), 0)::bigint,
    COALESCE(sum(session_summaries.output_tokens), 0)::bigint,
    COALESCE(sum(session_summaries.reasoning_output_tokens), 0)::bigint,
    COALESCE(sum(session_summaries.total_tokens), 0)::bigint,
    COALESCE(sum(session_summaries.cost_usd), 0),
    COALESCE(sum(session_summaries.patch_added_lines), 0)::bigint
FROM sessions
LEFT JOIN session_summaries ON session_summaries.session_id = sessions.id
WHERE COALESCE(
    session_summaries.updated_at,
    session_summaries.started_at,
    sessions.updated_at,
    sessions.started_at,
    sessions.ingested_at
)::date IN (SELECT bucket_date FROM affected_gpt_5_6_session_dates)
GROUP BY
    COALESCE(
        session_summaries.updated_at,
        session_summaries.started_at,
        sessions.updated_at,
        sessions.started_at,
        sessions.ingested_at
    )::date,
    date_trunc('month', COALESCE(
        session_summaries.updated_at,
        session_summaries.started_at,
        sessions.updated_at,
        sessions.started_at,
        sessions.ingested_at
    ))::date,
    sessions.device_id,
    sessions.repository_id,
    sessions.project_id;

COMMIT;
