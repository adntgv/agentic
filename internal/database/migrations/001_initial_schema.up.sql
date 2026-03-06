-- ═══════════════════════════════════════════════════════════
-- AGENTIC v0.1.5 — Initial Schema
-- Generated: 2026-03-06
-- ═══════════════════════════════════════════════════════════

-- ═══════════════════════════════════════════════════════════
-- USERS (human accounts — posters, human workers, operators)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_address  VARCHAR(42) UNIQUE NOT NULL,        -- 0x... checksummed
    email           VARCHAR(255),
    display_name    VARCHAR(100),
    user_type       VARCHAR(20) NOT NULL,               -- 'human' | 'operator'
    status          VARCHAR(20) NOT NULL DEFAULT 'registered', -- registered|verified|active|suspended|banned|inactive
    kyc_verified    BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_wallet ON users(wallet_address);
CREATE INDEX idx_users_status ON users(status);

-- ═══════════════════════════════════════════════════════════
-- AGENTS (AI agents, linked to operator)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id     UUID NOT NULL REFERENCES users(id),
    wallet_address  VARCHAR(42) UNIQUE NOT NULL,
    display_name    VARCHAR(100) NOT NULL,
    api_key_hash    VARCHAR(128) NOT NULL,               -- bcrypt hash of API key
    is_ai           BOOLEAN NOT NULL DEFAULT TRUE,       -- clearly labeled
    skill_manifest  JSONB,                               -- capabilities/categories
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_agents_operator ON agents(operator_id);
CREATE INDEX idx_agents_status ON agents(status);

-- ═══════════════════════════════════════════════════════════
-- WORKERS (v0.1.x — Unified Identity Model)
-- Single abstraction for all participants. All FKs use worker_id.
-- ═══════════════════════════════════════════════════════════
CREATE TABLE workers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_type     VARCHAR(10) NOT NULL,                -- 'user' | 'agent'
    user_id         UUID REFERENCES users(id),           -- populated if worker_type='user'
    agent_id        UUID REFERENCES agents(id),          -- populated if worker_type='agent'
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_worker_type CHECK (
        (worker_type = 'user'  AND user_id  IS NOT NULL AND agent_id IS NULL) OR
        (worker_type = 'agent' AND agent_id IS NOT NULL AND user_id  IS NULL)
    ),
    CONSTRAINT uq_worker_user  UNIQUE (user_id),
    CONSTRAINT uq_worker_agent UNIQUE (agent_id)
);
CREATE INDEX idx_workers_type ON workers(worker_type);
CREATE INDEX idx_workers_user ON workers(user_id);
CREATE INDEX idx_workers_agent ON workers(agent_id);

-- ═══════════════════════════════════════════════════════════
-- TASKS
-- ═══════════════════════════════════════════════════════════
CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poster_worker_id UUID NOT NULL REFERENCES workers(id), -- who posted (worker_id)
    title           VARCHAR(200) NOT NULL,
    description     TEXT NOT NULL,
    category        VARCHAR(50),
    budget          NUMERIC(18,6) NOT NULL,              -- USDC, 6 decimals
    deadline        TIMESTAMPTZ,
    bid_deadline    TIMESTAMPTZ,
    worker_filter   VARCHAR(20) DEFAULT 'both',          -- 'human_only' | 'ai_only' | 'both'
    max_revisions   SMALLINT DEFAULT 2,
    revision_count  SMALLINT DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft|published|bidding|pending_deposit|assigned|in_progress|review|completed|refunded|split|disputed|abandoned|overdue|expired|cancelled|deleted
    assigned_worker_id UUID REFERENCES workers(id),      -- unified worker reference
    accepted_bid_id   UUID,                              -- FK added after bids table
    task_id_hash    VARCHAR(66),                         -- bytes32 = keccak256(abi.encodePacked(uuid_bytes16))
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_tasks_poster ON tasks(poster_worker_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_category ON tasks(category);
CREATE INDEX idx_tasks_worker_filter ON tasks(worker_filter);
CREATE INDEX idx_tasks_assigned_worker ON tasks(assigned_worker_id);

-- ═══════════════════════════════════════════════════════════
-- BIDS
-- ═══════════════════════════════════════════════════════════
CREATE TABLE bids (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES tasks(id),
    worker_id       UUID NOT NULL REFERENCES workers(id), -- unified worker reference
    amount          NUMERIC(18,6) NOT NULL,              -- proposed price in USDC
    eta_hours       INTEGER,
    cover_letter    TEXT,
    bid_hash        VARCHAR(66),                         -- keccak256 (see bidHash formula)
    status          VARCHAR(20) DEFAULT 'pending',       -- pending|accepted|rejected|withdrawn
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_bids_task ON bids(task_id);
CREATE INDEX idx_bids_worker ON bids(worker_id);
ALTER TABLE tasks ADD CONSTRAINT fk_accepted_bid FOREIGN KEY (accepted_bid_id) REFERENCES bids(id);

-- ═══════════════════════════════════════════════════════════
-- ARTIFACTS (v0.1.x — Unified artifact model)
-- Used for deliverables, evidence, and revision requests.
-- v0.1.5: Added status field for pending/finalized lifecycle.
-- ═══════════════════════════════════════════════════════════
CREATE TABLE artifacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES tasks(id),
    worker_id       UUID NOT NULL REFERENCES workers(id), -- who uploaded
    context         VARCHAR(30) NOT NULL,                -- 'submission' | 'evidence' | 'revision_request'
    kind            VARCHAR(10) NOT NULL,                -- 'file' | 'url' | 'text'
    sha256          VARCHAR(64),                         -- required for file and text kinds (text computed server-side)
    url             VARCHAR(500),                        -- S3 URL or external URL
    text_body       TEXT,                                -- for kind='text' (stored inline, not S3)
    mime            VARCHAR(100),                        -- MIME type
    bytes           BIGINT,                              -- file size in bytes
    status          VARCHAR(20) DEFAULT 'pending',       -- pending|finalized
    av_scan_status  VARCHAR(20) DEFAULT 'pending',       -- pending|clean|infected|skipped
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finalized_at    TIMESTAMPTZ,                         -- when upload verified + finalized
    CONSTRAINT chk_artifact_sha256 CHECK (
        (kind IN ('file','text') AND sha256 IS NOT NULL) OR kind = 'url'
    ),
    CONSTRAINT chk_artifact_text_body CHECK (
        (kind = 'text' AND text_body IS NOT NULL) OR kind IN ('file','url')
    )
);
CREATE INDEX idx_artifacts_task ON artifacts(task_id);
CREATE INDEX idx_artifacts_context ON artifacts(task_id, context);
CREATE INDEX idx_artifacts_worker ON artifacts(worker_id);
CREATE INDEX idx_artifacts_status ON artifacts(status);

-- ═══════════════════════════════════════════════════════════
-- ESCROWS (mirrors on-chain state, updated by chain indexer)
-- v0.1.5: Renamed agent_address to payee_address for clarity.
-- ═══════════════════════════════════════════════════════════
CREATE TABLE escrows (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID UNIQUE NOT NULL REFERENCES tasks(id),
    poster_address  VARCHAR(42) NOT NULL,
    payee_address   VARCHAR(42) NOT NULL,                -- v0.1.5: renamed from agent_address (off-chain naming convention)
    amount          NUMERIC(18,6) NOT NULL,
    bid_hash        VARCHAR(66) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'none', -- none|locked|released|refunded|split
    deposit_tx_hash VARCHAR(66),
    release_tx_hash VARCHAR(66),
    refund_tx_hash  VARCHAR(66),
    split_tx_hash   VARCHAR(66),
    deposited_at    TIMESTAMPTZ,
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_escrows_task ON escrows(task_id);
CREATE INDEX idx_escrows_status ON escrows(status);

-- ═══════════════════════════════════════════════════════════
-- TRANSACTIONS (all on-chain tx log with reorg handling)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    escrow_id       UUID NOT NULL REFERENCES escrows(id),
    tx_hash         VARCHAR(66) UNIQUE NOT NULL,
    tx_type         VARCHAR(20) NOT NULL,                -- deposit|release|refund|split|reassign|mark_submitted
    block_number    BIGINT,
    block_hash      VARCHAR(66),                         -- stored for reorg detection
    status          VARCHAR(20) DEFAULT 'pending',       -- pending|confirmed|failed|reorged
    gas_used        BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at    TIMESTAMPTZ
);
CREATE INDEX idx_tx_escrow ON transactions(escrow_id);
CREATE INDEX idx_tx_hash ON transactions(tx_hash);
CREATE INDEX idx_tx_block ON transactions(block_number);

-- ═══════════════════════════════════════════════════════════
-- DISPUTES
-- ═══════════════════════════════════════════════════════════
CREATE TABLE disputes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES tasks(id),
    poster_worker_id UUID NOT NULL REFERENCES workers(id),   -- task poster (worker_id)
    assigned_worker_id UUID NOT NULL REFERENCES workers(id), -- assigned worker (worker_id)
    raised_by_worker_id UUID NOT NULL REFERENCES workers(id),-- who raised the dispute
    reason          VARCHAR(50) NOT NULL,                -- incomplete|wrong_requirements|quality|fraud
    status          VARCHAR(20) NOT NULL DEFAULT 'raised', -- raised|evidence|arbitration|resolved
    outcome         VARCHAR(20),                         -- agent_wins|poster_wins|split
    agent_bps       SMALLINT,                            -- for split outcomes
    rationale       TEXT,
    evidence_deadline TIMESTAMPTZ,                       -- 48h from creation
    bond_response_deadline TIMESTAMPTZ,                  -- 48h for responder to post bond
    sla_response_deadline TIMESTAMPTZ,                   -- 48h from evidence close
    sla_resolution_deadline TIMESTAMPTZ,                 -- 7 days from creation
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ
);
CREATE INDEX idx_disputes_task ON disputes(task_id);
CREATE INDEX idx_disputes_status ON disputes(status);

-- ═══════════════════════════════════════════════════════════
-- DISPUTE BONDS (v0.1.x — on-chain USDC to treasury, tracked here)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE dispute_bonds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispute_id      UUID NOT NULL REFERENCES disputes(id),
    worker_id       UUID NOT NULL REFERENCES workers(id), -- who posted the bond
    role            VARCHAR(20) NOT NULL,                -- 'raiser' | 'responder'
    amount          NUMERIC(18,6) NOT NULL,              -- escrow_amount * 1%
    tx_hash         VARCHAR(66) NOT NULL,                -- on-chain USDC transfer to treasury
    status          VARCHAR(20) NOT NULL DEFAULT 'posted', -- posted|returned|retained
    return_tx_hash  VARCHAR(66),                         -- tx hash when bond is returned
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    settled_at      TIMESTAMPTZ
);
CREATE INDEX idx_dispute_bonds_dispute ON dispute_bonds(dispute_id);

-- ═══════════════════════════════════════════════════════════
-- STAKES (v0.1.x — custodial stake to treasury)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE stakes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id     UUID NOT NULL REFERENCES users(id),  -- operator user
    agent_id        UUID NOT NULL REFERENCES agents(id), -- staked agent
    amount          NUMERIC(18,6) NOT NULL,              -- USDC amount staked
    tx_hash         VARCHAR(66) NOT NULL,                -- on-chain USDC transfer to treasury
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active|slashed|withdrawn
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_stakes_operator ON stakes(operator_id);
CREATE INDEX idx_stakes_agent ON stakes(agent_id);

-- ═══════════════════════════════════════════════════════════
-- MESSAGES (task thread)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES tasks(id),
    sender_worker_id UUID NOT NULL REFERENCES workers(id), -- unified worker reference
    body            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_messages_task ON messages(task_id);
CREATE INDEX idx_messages_task_created ON messages(task_id, created_at);

-- ═══════════════════════════════════════════════════════════
-- WEBHOOKS
-- ═══════════════════════════════════════════════════════════
CREATE TABLE webhooks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_worker_id UUID NOT NULL REFERENCES workers(id), -- unified worker reference
    url             VARCHAR(500) NOT NULL,
    secret          VARCHAR(128) NOT NULL,               -- HMAC secret
    events          TEXT[] NOT NULL,                      -- subscribed event types
    status          VARCHAR(20) DEFAULT 'active',        -- active|disabled
    consecutive_failures INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_webhooks_owner ON webhooks(owner_worker_id);

-- ═══════════════════════════════════════════════════════════
-- OUTBOX (transactional outbox for events)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE outbox (
    id              BIGSERIAL PRIMARY KEY,
    event_type      VARCHAR(50) NOT NULL,
    payload         JSONB NOT NULL,
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    sequence_number BIGSERIAL,
    status          VARCHAR(20) DEFAULT 'pending',       -- pending|dispatched|failed
    retry_count     SMALLINT DEFAULT 0,
    next_retry_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at   TIMESTAMPTZ
);
CREATE INDEX idx_outbox_status ON outbox(status, next_retry_at);

-- ═══════════════════════════════════════════════════════════
-- IDEMPOTENCY KEYS (v0.1.x, fixed in v0.1.5)
-- v0.1.5: Changed PK from (worker_id, key) to (worker_id, endpoint, key)
-- to match the stated scope (actor_id, endpoint).
-- ═══════════════════════════════════════════════════════════
CREATE TABLE idempotency_keys (
    key             VARCHAR(128) NOT NULL,
    worker_id       UUID NOT NULL REFERENCES workers(id),
    endpoint        VARCHAR(200) NOT NULL,               -- normalized: "{METHOD} {ROUTE_TEMPLATE}"
    response_status SMALLINT NOT NULL,
    response_body   JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '24 hours',
    PRIMARY KEY (worker_id, endpoint, key)               -- v0.1.5: fixed PK scope
);
CREATE INDEX idx_idempotency_expires ON idempotency_keys(expires_at);

-- ═══════════════════════════════════════════════════════════
-- REPUTATION (off-chain, v0.1.x — uses worker_id)
-- ═══════════════════════════════════════════════════════════
CREATE TABLE reputation (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rated_worker_id UUID NOT NULL REFERENCES workers(id), -- who is rated
    rater_worker_id UUID NOT NULL REFERENCES workers(id), -- who gave the rating
    task_id         UUID REFERENCES tasks(id),
    rating          SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    role            VARCHAR(20) NOT NULL,                -- 'poster' | 'worker' (perspective of rater)
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_reputation_rated ON reputation(rated_worker_id);
CREATE INDEX idx_reputation_rater ON reputation(rater_worker_id);
CREATE INDEX idx_reputation_task ON reputation(task_id);

-- ═══════════════════════════════════════════════════════════
-- REPUTATION SUMMARY (materialized view)
-- ═══════════════════════════════════════════════════════════
CREATE MATERIALIZED VIEW reputation_summary AS
SELECT
    rated_worker_id,
    COUNT(*) as total_ratings,
    ROUND(AVG(rating), 2) as avg_rating,
    COUNT(*) FILTER (WHERE rating >= 4) as positive_count,
    COUNT(*) FILTER (WHERE rating <= 2) as negative_count
FROM reputation
GROUP BY rated_worker_id;

CREATE UNIQUE INDEX idx_rep_summary_worker ON reputation_summary(rated_worker_id);
