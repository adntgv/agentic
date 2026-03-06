// Type definitions for Agentic marketplace

export interface Task {
  id: string;
  poster_worker_id: string;
  title: string;
  description: string;
  category?: string;
  budget: string;
  deadline?: string;
  bid_deadline?: string;
  worker_filter: "human_only" | "ai_only" | "both";
  max_revisions: number;
  revision_count: number;
  status: TaskStatus;
  assigned_worker_id?: string;
  accepted_bid_id?: string;
  task_id_hash?: string;
  created_at: string;
  updated_at: string;
}

export type TaskStatus =
  | "draft" | "published" | "bidding" | "pending_deposit"
  | "assigned" | "in_progress" | "review" | "completed"
  | "refunded" | "split" | "disputed" | "abandoned"
  | "overdue" | "expired" | "cancelled" | "deleted";

export interface Bid {
  id: string;
  task_id: string;
  worker_id: string;
  amount: string;
  eta_hours?: number;
  cover_letter?: string;
  bid_hash?: string;
  status: "pending" | "accepted" | "rejected" | "withdrawn";
  created_at: string;
}

export interface Escrow {
  id: string;
  task_id: string;
  poster_address: string;
  payee_address: string;
  amount: string;
  bid_hash: string;
  status: "none" | "locked" | "released" | "refunded" | "split";
  deposit_tx_hash?: string;
  release_tx_hash?: string;
  refund_tx_hash?: string;
  split_tx_hash?: string;
  deposited_at?: string;
  resolved_at?: string;
  created_at: string;
}

export interface Dispute {
  id: string;
  task_id: string;
  poster_worker_id: string;
  assigned_worker_id: string;
  raised_by_worker_id: string;
  reason: "incomplete" | "wrong_requirements" | "quality" | "fraud";
  status: "raised" | "evidence" | "arbitration" | "resolved";
  outcome?: "agent_wins" | "poster_wins" | "split";
  agent_bps?: number;
  rationale?: string;
  evidence_deadline?: string;
  bond_response_deadline?: string;
  sla_response_deadline?: string;
  sla_resolution_deadline?: string;
  created_at: string;
  resolved_at?: string;
}

export interface DisputeBond {
  id: string;
  dispute_id: string;
  worker_id: string;
  role: "raiser" | "responder";
  amount: string;
  tx_hash: string;
  status: "posted" | "returned" | "retained";
  return_tx_hash?: string;
  created_at: string;
  settled_at?: string;
}

export interface Artifact {
  id: string;
  task_id: string;
  worker_id: string;
  context: "submission" | "evidence" | "revision_request";
  kind: "file" | "url" | "text";
  sha256?: string;
  url?: string;
  text_body?: string;
  mime?: string;
  bytes?: number;
  status: "pending" | "finalized";
  av_scan_status: "pending" | "clean" | "infected" | "skipped";
  created_at: string;
  finalized_at?: string;
}

export interface WorkerProfile {
  id: string;
  worker_type: "user" | "agent";
  user_id?: string;
  agent_id?: string;
  display_name?: string;
  wallet_address: string;
  is_ai?: boolean;
  skill_manifest?: Record<string, unknown>;
  status: string;
  created_at: string;
  operator?: string;
}

export interface ReputationSummary {
  rated_worker_id: string;
  total_ratings: number;
  avg_rating: number;
  positive_count: number;
  negative_count: number;
  completion_rate?: number;
  dispute_ratio?: number;
}

export interface TaskFilterParams {
  status?: TaskStatus | string;
  category?: string;
  budget_min?: number;
  budget_max?: number;
  worker_filter?: string;
  page?: number;
  limit?: number;
  poster_worker_id?: string;
  assigned_worker_id?: string;
}

export interface CreateTaskRequest {
  title: string;
  description: string;
  category?: string;
  budget: string;
  deadline?: string;
  bid_deadline?: string;
  worker_filter?: string;
  max_revisions?: number;
}

export interface CreateBidRequest {
  amount: string;
  eta_hours?: number;
  cover_letter?: string;
}

export interface RaiseDisputeRequest {
  reason: string;
}

export interface DisputeRulingRequest {
  outcome: "agent_wins" | "poster_wins" | "split";
  agent_bps?: number;
  rationale: string;
}

export interface AuthResponse {
  token: string;
}

export interface WebhookConfig {
  id: string;
  url: string;
  events: string[];
  active: boolean;
  created_at: string;
}
