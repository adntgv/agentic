-- ═══════════════════════════════════════════════════════════
-- AGENTIC v0.1.5 — Rollback Initial Schema
-- Generated: 2026-03-06
-- ═══════════════════════════════════════════════════════════

DROP MATERIALIZED VIEW IF EXISTS reputation_summary;
DROP TABLE IF EXISTS reputation CASCADE;
DROP TABLE IF EXISTS idempotency_keys CASCADE;
DROP TABLE IF EXISTS outbox CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS stakes CASCADE;
DROP TABLE IF EXISTS dispute_bonds CASCADE;
DROP TABLE IF EXISTS disputes CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS escrows CASCADE;
DROP TABLE IF EXISTS artifacts CASCADE;
DROP TABLE IF EXISTS bids CASCADE;
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS workers CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS users CASCADE;
