/**
 * API client for Agentic marketplace backend
 * All functions return typed promises and handle auth via JWT in localStorage
 */

import type {
  Task,
  Bid,
  Escrow,
  Dispute,
  Artifact,
  WorkerProfile,
  ReputationSummary,
  TaskFilterParams,
  CreateTaskRequest,
  CreateBidRequest,
  RaiseDisputeRequest,
  DisputeRulingRequest,
  AuthResponse,
  WebhookConfig,
} from '@/types'

const BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('auth_token')
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  })

  if (!res.ok) {
    const errorText = await res.text()
    throw new Error(errorText || `HTTP ${res.status}: ${res.statusText}`)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

// ============================================================================
// AUTH
// ============================================================================

export const walletAuth = (wallet_address: string, signature: string, message: string) =>
  request<AuthResponse>('/auth/wallet', {
    method: 'POST',
    body: JSON.stringify({ wallet_address, signature, message }),
  })

export const apiKeyAuth = (api_key: string) =>
  request<AuthResponse>('/auth/apikey', {
    method: 'POST',
    body: JSON.stringify({ api_key }),
  })

// ============================================================================
// TASKS
// ============================================================================

export const createTask = (data: CreateTaskRequest) =>
  request<Task>('/tasks', { method: 'POST', body: JSON.stringify(data) })

export const listTasks = (params?: TaskFilterParams) => {
  const query = params ? `?${new URLSearchParams(params as Record<string, string>)}` : ''
  return request<Task[]>(`/tasks${query}`)
}

export const getTask = (id: string) =>
  request<Task>(`/tasks/${id}`)

export const updateTask = (id: string, data: Partial<Task>) =>
  request<Task>(`/tasks/${id}`, { method: 'PATCH', body: JSON.stringify(data) })

export const deleteTask = (id: string) =>
  request<void>(`/tasks/${id}`, { method: 'DELETE' })

export const cancelTask = (id: string) =>
  request<Task>(`/tasks/${id}/cancel`, { method: 'POST' })

// ============================================================================
// BIDS
// ============================================================================

export const createBid = (taskId: string, data: CreateBidRequest) =>
  request<Bid>(`/tasks/${taskId}/bids`, { method: 'POST', body: JSON.stringify(data) })

export const listBids = (taskId: string) =>
  request<Bid[]>(`/tasks/${taskId}/bids`)

export const acceptBid = (taskId: string, bidId: string) =>
  request<Bid>(`/tasks/${taskId}/bids/${bidId}/accept`, { method: 'POST' })

// ============================================================================
// WORK LIFECYCLE
// ============================================================================

export const ackTask = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/ack`, { method: 'POST' })

export const submitTask = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/submit`, { method: 'POST' })

export const approveTask = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/approve`, { method: 'POST' })

export const requestRevision = (taskId: string, data?: { reason?: string }) =>
  request<Task>(`/tasks/${taskId}/revision`, {
    method: 'POST',
    body: JSON.stringify(data || {}),
  })

// ============================================================================
// ARTIFACTS
// ============================================================================

export const requestUploadURL = (
  taskId: string,
  data: { filename: string; mime: string; context: string }
) =>
  request<{ upload_url: string; artifact_id: string }>(
    `/tasks/${taskId}/artifacts/upload-url`,
    { method: 'POST', body: JSON.stringify(data) }
  )

export const listArtifacts = (taskId: string) =>
  request<Artifact[]>(`/tasks/${taskId}/artifacts`)

// ============================================================================
// ESCROW
// ============================================================================

export const getEscrow = (taskId: string) =>
  request<Escrow>(`/tasks/${taskId}/escrow`)

// ============================================================================
// DISPUTES
// ============================================================================

export const raiseDispute = (taskId: string, data: RaiseDisputeRequest) =>
  request<Dispute>(`/tasks/${taskId}/disputes`, {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const respondDispute = (disputeId: string, data: { response: string }) =>
  request<Dispute>(`/disputes/${disputeId}/respond`, {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const submitEvidence = (disputeId: string, data: { evidence: string }) =>
  request<void>(`/disputes/${disputeId}/evidence`, {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const submitRuling = (disputeId: string, data: DisputeRulingRequest) =>
  request<Dispute>(`/disputes/${disputeId}/ruling`, {
    method: 'POST',
    body: JSON.stringify(data),
  })

// ============================================================================
// WORKERS
// ============================================================================

export const getWorker = (id: string) =>
  request<WorkerProfile>(`/workers/${id}`)

export const getWorkerHistory = (id: string) =>
  request<Task[]>(`/workers/${id}/history`)

export const getOperator = (id: string) =>
  request<WorkerProfile>(`/operators/${id}`)

// ============================================================================
// WEBHOOKS
// ============================================================================

export const createWebhook = (data: { url: string; events: string[] }) =>
  request<WebhookConfig>('/webhooks', { method: 'POST', body: JSON.stringify(data) })

export const listWebhooks = () =>
  request<WebhookConfig[]>('/webhooks')

// ============================================================================
// ADMIN
// ============================================================================

export const adminReassign = (taskId: string, data: { new_worker_address: string }) =>
  request<Task>(`/tasks/${taskId}/reassign`, {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const adminRefund = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/refund`, { method: 'POST' })

export const adminRuling = (disputeId: string, data: DisputeRulingRequest) =>
  submitRuling(disputeId, data) // Same endpoint, admin-only access

// ============================================================================
// ADDITIONAL ADMIN / WORKER ENDPOINTS
// ============================================================================

export const listDisputes = () =>
  request<Dispute[]>('/disputes')

export const adminUnassign = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/unassign`, { method: 'POST' })

export const getReputation = (workerId: string) =>
  request<ReputationSummary>(`/workers/${workerId}/reputation`)

export const updateWorker = (workerId: string, data: Partial<WorkerProfile>) =>
  request<WorkerProfile>(`/workers/${workerId}`, { method: 'PATCH', body: JSON.stringify(data) })
