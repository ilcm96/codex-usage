CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    hostname text NOT NULL,
    platform text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (hostname)
);

CREATE TABLE repositories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_url text NOT NULL,
    repository_host text NOT NULL DEFAULT '',
    repository_owner text NOT NULL DEFAULT '',
    repository_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repository_url)
);

CREATE TABLE repository_aliases (
    id bigserial PRIMARY KEY,
    repository_id uuid NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    alias_url text NOT NULL,
    source text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (alias_url)
);

CREATE TABLE projects (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id uuid REFERENCES repositories(id) ON DELETE SET NULL,
    cwd text NOT NULL,
    git_root text NOT NULL DEFAULT '',
    relative_path text NOT NULL DEFAULT '',
    display_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repository_id, cwd)
);

CREATE TABLE sessions (
    id text PRIMARY KEY,
    device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    repository_id uuid REFERENCES repositories(id) ON DELETE SET NULL,
    project_id uuid REFERENCES projects(id) ON DELETE SET NULL,
    started_at timestamptz,
    updated_at timestamptz,
    cwd text NOT NULL DEFAULT '',
    originator text NOT NULL DEFAULT '',
    source text NOT NULL DEFAULT '',
    cli_version text NOT NULL DEFAULT '',
    model_provider text NOT NULL DEFAULT '',
    branch text NOT NULL DEFAULT '',
    commit_hash text NOT NULL DEFAULT '',
    raw_sha256 text NOT NULL,
    raw_size_bytes bigint NOT NULL DEFAULT 0,
    raw_file_path text NOT NULL DEFAULT '',
    ingested_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (raw_sha256)
);

CREATE TABLE archive_files (
    id bigserial PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    raw_file_path text NOT NULL,
    raw_sha256 text NOT NULL,
    raw_size_bytes bigint NOT NULL DEFAULT 0,
    archived_at timestamptz NOT NULL DEFAULT now(),
    verified_at timestamptz,
    status text NOT NULL DEFAULT 'archived',
    UNIQUE (session_id)
);

CREATE TABLE import_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id uuid REFERENCES devices(id) ON DELETE SET NULL,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    status text NOT NULL DEFAULT 'running',
    sessions_seen bigint NOT NULL DEFAULT 0,
    sessions_inserted bigint NOT NULL DEFAULT 0,
    sessions_updated bigint NOT NULL DEFAULT 0,
    raw_bytes bigint NOT NULL DEFAULT 0,
    error_message text NOT NULL DEFAULT ''
);

CREATE TABLE session_events (
    id bigserial PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    seq integer NOT NULL,
    event_hash text NOT NULL,
    occurred_at timestamptz,
    event_type text NOT NULL,
    payload_type text NOT NULL DEFAULT '',
    role text NOT NULL DEFAULT '',
    tool_name text NOT NULL DEFAULT '',
    call_id text NOT NULL DEFAULT '',
    content_text text NOT NULL DEFAULT '',
    payload_jsonb jsonb NOT NULL,
    UNIQUE (session_id, seq),
    UNIQUE (session_id, event_hash)
);

CREATE TABLE messages (
    id bigserial PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    seq integer NOT NULL,
    occurred_at timestamptz,
    role text NOT NULL,
    content_text text NOT NULL DEFAULT '',
    content_jsonb jsonb NOT NULL,
    UNIQUE (session_id, seq)
);

CREATE TABLE tool_events (
    id bigserial PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    seq integer NOT NULL,
    occurred_at timestamptz,
    kind text NOT NULL,
    tool_name text NOT NULL DEFAULT '',
    call_id text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT '',
    input_text text NOT NULL DEFAULT '',
    output_text text NOT NULL DEFAULT '',
    payload_jsonb jsonb NOT NULL,
    UNIQUE (session_id, seq)
);

CREATE TABLE session_summaries (
    session_id text PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    title text NOT NULL DEFAULT '',
    display_title text NOT NULL DEFAULT '',
    display_subtitle text NOT NULL DEFAULT '',
    user_intent text NOT NULL DEFAULT '',
    dominant_language text NOT NULL DEFAULT '',
    first_user_message text NOT NULL DEFAULT '',
    last_user_message text NOT NULL DEFAULT '',
    short_summary text NOT NULL DEFAULT '',
    message_count bigint NOT NULL DEFAULT 0,
    user_message_count bigint NOT NULL DEFAULT 0,
    assistant_message_count bigint NOT NULL DEFAULT 0,
    tool_call_count bigint NOT NULL DEFAULT 0,
    patch_added_lines bigint NOT NULL DEFAULT 0,
    patch_language_stats jsonb NOT NULL DEFAULT '{}'::jsonb,
    conversation_turn_count bigint NOT NULL DEFAULT 0,
    searchable_message_count bigint NOT NULL DEFAULT 0,
    searchable_tool_count bigint NOT NULL DEFAULT 0,
    main_model text NOT NULL DEFAULT '',
    models text NOT NULL DEFAULT '',
    input_tokens bigint NOT NULL DEFAULT 0,
    cached_input_tokens bigint NOT NULL DEFAULT 0,
    cache_write_input_tokens bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    reasoning_output_tokens bigint NOT NULL DEFAULT 0,
    total_tokens bigint NOT NULL DEFAULT 0,
    cost_usd numeric(18, 8) NOT NULL DEFAULT 0,
    duration_seconds bigint NOT NULL DEFAULT 0,
    cache_hit_rate numeric(8, 6) NOT NULL DEFAULT 0,
    started_at timestamptz,
    updated_at timestamptz,
    generated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE conversation_turns (
    id bigserial PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    turn_index integer NOT NULL,
    user_seq integer,
    assistant_seq integer,
    started_at timestamptz,
    ended_at timestamptz,
    user_text text NOT NULL DEFAULT '',
    assistant_text text NOT NULL DEFAULT '',
    tool_call_count bigint NOT NULL DEFAULT 0,
    tool_result_count bigint NOT NULL DEFAULT 0,
    tool_names text[] NOT NULL DEFAULT '{}',
    UNIQUE (session_id, turn_index)
);

CREATE TABLE search_documents (
    id bigserial PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    seq integer NOT NULL,
    turn_index integer,
    occurred_at timestamptz,
    kind text NOT NULL,
    document_scope text NOT NULL DEFAULT '',
    role text NOT NULL DEFAULT '',
    tool_name text NOT NULL DEFAULT '',
    title text NOT NULL DEFAULT '',
    body text NOT NULL DEFAULT '',
    snippet text NOT NULL DEFAULT '',
    rank_weight integer NOT NULL DEFAULT 0,
    default_searchable boolean NOT NULL DEFAULT true,
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(body, ''))
    ) STORED,
    UNIQUE (session_id, kind, seq)
);

CREATE TABLE usage_events (
    id bigserial PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    seq integer NOT NULL,
    occurred_at timestamptz,
    model text NOT NULL,
    input_tokens bigint NOT NULL DEFAULT 0,
    cached_input_tokens bigint NOT NULL DEFAULT 0,
    cache_write_input_tokens bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    reasoning_output_tokens bigint NOT NULL DEFAULT 0,
    total_tokens bigint NOT NULL DEFAULT 0,
    cost_usd numeric(18, 8) NOT NULL DEFAULT 0,
    UNIQUE (session_id, seq)
);

CREATE TABLE usage_rollups (
    id bigserial PRIMARY KEY,
    bucket_date date NOT NULL,
    bucket_month date NOT NULL,
    device_id uuid REFERENCES devices(id) ON DELETE CASCADE,
    repository_id uuid REFERENCES repositories(id) ON DELETE CASCADE,
    project_id uuid REFERENCES projects(id) ON DELETE CASCADE,
    model text NOT NULL,
    input_tokens bigint NOT NULL DEFAULT 0,
    cached_input_tokens bigint NOT NULL DEFAULT 0,
    cache_write_input_tokens bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    reasoning_output_tokens bigint NOT NULL DEFAULT 0,
    total_tokens bigint NOT NULL DEFAULT 0,
    cost_usd numeric(18, 8) NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (bucket_date, device_id, repository_id, project_id, model)
);

CREATE TABLE session_rollups (
    id bigserial PRIMARY KEY,
    bucket_date date NOT NULL,
    bucket_month date NOT NULL,
    device_id uuid REFERENCES devices(id) ON DELETE CASCADE,
    repository_id uuid REFERENCES repositories(id) ON DELETE CASCADE,
    project_id uuid REFERENCES projects(id) ON DELETE CASCADE,
    session_count bigint NOT NULL DEFAULT 0,
    input_tokens bigint NOT NULL DEFAULT 0,
    cached_input_tokens bigint NOT NULL DEFAULT 0,
    cache_write_input_tokens bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    reasoning_output_tokens bigint NOT NULL DEFAULT 0,
    total_tokens bigint NOT NULL DEFAULT 0,
    cost_usd numeric(18, 8) NOT NULL DEFAULT 0,
    patch_added_lines bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (bucket_date, device_id, repository_id, project_id)
);

CREATE INDEX sessions_device_id_idx ON sessions(device_id);
CREATE INDEX sessions_repository_id_idx ON sessions(repository_id);
CREATE INDEX sessions_project_id_idx ON sessions(project_id);
CREATE INDEX sessions_started_at_idx ON sessions(started_at DESC);
CREATE INDEX sessions_effective_updated_at_idx ON sessions((COALESCE(updated_at, started_at, ingested_at)) DESC, id DESC);
CREATE INDEX archive_files_device_id_idx ON archive_files(device_id);
CREATE INDEX archive_files_status_idx ON archive_files(status);
CREATE INDEX import_runs_started_at_idx ON import_runs(started_at DESC);
CREATE INDEX session_events_session_id_seq_idx ON session_events(session_id, seq);
CREATE INDEX messages_session_id_seq_idx ON messages(session_id, seq);
CREATE INDEX tool_events_session_id_seq_idx ON tool_events(session_id, seq);
CREATE INDEX session_summaries_main_model_idx ON session_summaries(main_model);
CREATE INDEX session_summaries_total_tokens_idx ON session_summaries(total_tokens DESC);
CREATE INDEX session_summaries_cost_usd_idx ON session_summaries(cost_usd DESC);
CREATE INDEX session_summaries_patch_added_lines_idx ON session_summaries(patch_added_lines DESC);
CREATE INDEX session_summaries_display_title_trgm_idx ON session_summaries USING gin (display_title gin_trgm_ops);
CREATE INDEX session_summaries_user_intent_trgm_idx ON session_summaries USING gin (user_intent gin_trgm_ops);
CREATE INDEX conversation_turns_session_id_turn_idx ON conversation_turns(session_id, turn_index);
CREATE INDEX search_documents_session_id_seq_idx ON search_documents(session_id, seq);
CREATE INDEX search_documents_kind_idx ON search_documents(kind);
CREATE INDEX search_documents_scope_idx ON search_documents(document_scope);
CREATE INDEX search_documents_default_idx ON search_documents(default_searchable);
CREATE INDEX search_documents_rank_idx ON search_documents(rank_weight DESC);
CREATE INDEX search_documents_occurred_at_idx ON search_documents(occurred_at DESC);
CREATE INDEX search_documents_search_idx ON search_documents USING gin (search_vector);
CREATE INDEX search_documents_title_trgm_idx ON search_documents USING gin (title gin_trgm_ops);
CREATE INDEX search_documents_body_trgm_idx ON search_documents USING gin (body gin_trgm_ops);
CREATE INDEX usage_events_session_id_seq_idx ON usage_events(session_id, seq);
CREATE INDEX usage_events_occurred_at_idx ON usage_events(occurred_at DESC) WHERE occurred_at IS NOT NULL;
CREATE INDEX usage_events_model_session_id_idx ON usage_events(model, session_id) WHERE model <> '';
CREATE INDEX messages_occurred_at_idx ON messages(occurred_at DESC) WHERE occurred_at IS NOT NULL;
CREATE INDEX tool_events_occurred_at_idx ON tool_events(occurred_at DESC) WHERE occurred_at IS NOT NULL;
CREATE INDEX usage_rollups_bucket_date_idx ON usage_rollups(bucket_date DESC);
CREATE INDEX usage_rollups_bucket_month_idx ON usage_rollups(bucket_month DESC);
CREATE INDEX session_rollups_bucket_date_idx ON session_rollups(bucket_date DESC);
CREATE INDEX session_rollups_bucket_month_idx ON session_rollups(bucket_month DESC);
CREATE INDEX session_rollups_repository_id_idx ON session_rollups(repository_id);
CREATE INDEX session_rollups_project_id_idx ON session_rollups(project_id);

CREATE INDEX session_events_content_trgm_idx ON session_events USING gin (content_text gin_trgm_ops);
CREATE INDEX messages_content_search_idx ON messages USING gin (to_tsvector('simple', content_text));
CREATE INDEX messages_content_trgm_idx ON messages USING gin (content_text gin_trgm_ops);
CREATE INDEX tool_events_input_search_idx ON tool_events USING gin (to_tsvector('simple', input_text));
CREATE INDEX tool_events_output_search_idx ON tool_events USING gin (to_tsvector('simple', output_text));
CREATE INDEX tool_events_output_trgm_idx ON tool_events USING gin (output_text gin_trgm_ops);
