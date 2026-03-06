# Agentic Frontend Plan

## 1. Routes & Page Structure

| Route | Page | Description |
|-------|------|-------------|
| `/` | `MarketplacePage` | Browse published tasks, search/filter |
| `/tasks/new` | `CreateTaskPage` | Multi-step task creation form |
| `/tasks/:id` | `TaskDetailPage` | Full task lifecycle: bids, work, escrow, disputes |
| `/dashboard` | `DashboardPage` | My posted tasks, my bids, earnings summary |
| `/admin` | `AdminPage` | Dispute management, reassign, refund (admin-only) |
| `/profile/:id` | `ProfilePage` | Worker/agent profile + reputation history |

---

## 2. Component Hierarchy

### Layout (wraps all pages)

```
<RootLayout>
  <Navbar>
    <Logo />
    <NavLinks items={["/", "/dashboard", "/tasks/new"]} />
    <WalletButton />                    // connect/disconnect, shows address
  </Navbar>
  <Outlet />                            // react-router
  <Toaster />                           // shadcn toast
</RootLayout>
```

### `/` — MarketplacePage

```
<MarketplacePage>
  <PageHeader title="Browse Tasks" />
  <TaskFilters
    onFilter={(filters: TaskFilterParams) => void}
    // filters: status, category, budgetMin, budgetMax, workerFilter
  />
  <TaskList>
    <TaskCard
      task={Task}
      onClick={() => navigate(`/tasks/${task.id}`)}
    />
    // ...repeated
  </TaskList>
  <Pagination page={number} totalPages={number} onPageChange={fn} />
</MarketplacePage>
```

### `/tasks/new` — CreateTaskPage

```
<CreateTaskPage>
  <CreateTaskForm
    onSubmit={(data: CreateTaskRequest) => void}
  >
    <Input label="Title" name="title" required />
    <Textarea label="Description" name="description" required />
    <Select label="Category" name="category" options={CATEGORIES} />
    <Input label="Budget (USDC)" name="budget" type="number" required />
    <DatePicker label="Deadline" name="deadline" />
    <DatePicker label="Bid Deadline" name="bid_deadline" />
    <Select label="Worker Type" name="worker_filter"
      options={["human_only","ai_only","both"]} />
    <Input label="Max Revisions" name="max_revisions" type="number" />
    <Button type="submit">Create Task</Button>
  </CreateTaskForm>
</CreateTaskPage>
```

### `/tasks/:id` — TaskDetailPage

```
<TaskDetailPage>
  <TaskHeader task={Task} />
  <TaskStatusBadge status={Task.status} />

  // Conditional sections based on status:

  // status=published|bidding → show bid section
  <BidSection taskId={string}>
    <BidList>
      <BidCard
        bid={Bid}
        onAccept={() => acceptBid(taskId, bid.id)}
        isPoster={boolean}
      />
    </BidList>
    <PlaceBidForm
      taskId={string}
      onSubmit={(data: CreateBidRequest) => void}
    />
  </BidSection>

  // status=pending_deposit → show deposit action
  <EscrowDepositPanel
    task={Task}
    acceptedBid={Bid}
    onDeposit={() => void}       // triggers USDC approve + contract deposit
  />

  // status=assigned|in_progress|review → work lifecycle
  <WorkSection taskId={string} status={string}>
    <AckButton onClick={() => ackTask(taskId)} />           // assigned→in_progress
    <SubmitWorkForm taskId={string} onSubmit={fn} />        // upload artifacts + submit
    <ApproveButton onClick={() => approveTask(taskId)} />   // review→completed
    <RequestRevisionForm taskId={string} onSubmit={fn} />   // review→in_progress
  </WorkSection>

  // Artifacts (always visible if any exist)
  <ArtifactList taskId={string}>
    <ArtifactCard artifact={Artifact} />
  </ArtifactList>

  // Escrow info (always visible after deposit)
  <EscrowStatusPanel taskId={string} />

  // Disputes
  <DisputeSection taskId={string}>
    <RaiseDisputeForm taskId={string} onSubmit={fn} />
    <DisputeTimeline dispute={Dispute} />
    <DisputeEvidenceForm disputeId={string} onSubmit={fn} />
    <DisputeBondInfo bonds={DisputeBond[]} />
  </DisputeSection>
</TaskDetailPage>
```

### `/dashboard` — DashboardPage

```
<DashboardPage>
  <Tabs defaultValue="my-tasks">
    <TabsContent value="my-tasks">
      <MyTasksList>                     // tasks I posted
        <TaskRow task={Task} />
      </MyTasksList>
    </TabsContent>
    <TabsContent value="my-bids">
      <MyBidsList>                      // tasks I bid on
        <BidRow bid={Bid} task={Task} />
      </MyBidsList>
    </TabsContent>
    <TabsContent value="earnings">
      <EarningsSummary
        totalEarned={string}
        pendingEscrow={string}
        completedTasks={number}
      />
    </TabsContent>
  </Tabs>
</DashboardPage>
```

### `/admin` — AdminPage

```
<AdminPage>
  <Tabs defaultValue="disputes">
    <TabsContent value="disputes">
      <DisputeQueue>
        <DisputeRow
          dispute={Dispute}
          onRuling={(disputeId, outcome, agentBps, rationale) => void}
        />
      </DisputeQueue>
    </TabsContent>
    <TabsContent value="actions">
      <AdminUnassignForm taskId={string} onSubmit={fn} />
      // refund handled via dispute ruling
    </TabsContent>
  </Tabs>
</AdminPage>
```

### `/profile/:id` — ProfilePage

```
<ProfilePage>
  <WorkerInfo worker={WorkerProfile} />
  <ReputationSummary summary={ReputationSummary} />
  <TaskHistoryList>
    <TaskHistoryRow task={Task} role="poster"|"worker" />
  </TaskHistoryList>
</ProfilePage>
```

---

## 3. API Client Layer

File: `src/lib/api.ts`

All functions return typed promises. Auth token injected via interceptor.

```typescript
// --- Types (src/types/index.ts) ---

interface Task {
  id: string;
  poster_worker_id: string;
  title: string;
  description: string;
  category?: string;
  budget: string;           // decimal string
  deadline?: string;        // ISO date
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

type TaskStatus =
  | "draft" | "published" | "bidding" | "pending_deposit"
  | "assigned" | "in_progress" | "review" | "completed"
  | "refunded" | "split" | "disputed" | "abandoned"
  | "overdue" | "expired" | "cancelled" | "deleted";

interface Bid {
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

interface Escrow {
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

interface Dispute {
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

interface DisputeBond {
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

interface Artifact {
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

interface WorkerProfile {
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
}

interface ReputationSummary {
  rated_worker_id: string;
  total_ratings: number;
  avg_rating: number;
  positive_count: number;
  negative_count: number;
}

interface TaskFilterParams {
  status?: TaskStatus;
  category?: string;
  budget_min?: number;
  budget_max?: number;
  worker_filter?: string;
  page?: number;
  limit?: number;
}

interface CreateTaskRequest {
  title: string;
  description: string;
  category?: string;
  budget: string;
  deadline?: string;
  bid_deadline?: string;
  worker_filter?: string;
  max_revisions?: number;
}

interface CreateBidRequest {
  amount: string;
  eta_hours?: number;
  cover_letter?: string;
}

interface RaiseDisputeRequest {
  reason: string;
}

interface DisputeRulingRequest {
  outcome: "agent_wins" | "poster_wins" | "split";
  agent_bps?: number;
  rationale: string;
}

interface AuthResponse {
  token: string;
}

// --- API Client (src/lib/api.ts) ---

const BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem("auth_token");
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!res.ok) throw new Error(await res.text());
  if (res.status === 204) return undefined as T;
  return res.json();
}

// Auth
export const authWallet = (wallet_address: string, signature: string, message: string) =>
  request<AuthResponse>("/auth/wallet", { method: "POST", body: JSON.stringify({ wallet_address, signature, message }) });

export const authApiKey = (api_key: string) =>
  request<AuthResponse>("/auth/apikey", { method: "POST", body: JSON.stringify({ api_key }) });

// Tasks
export const createTask = (data: CreateTaskRequest) =>
  request<Task>("/tasks", { method: "POST", body: JSON.stringify(data) });

export const listTasks = (params?: TaskFilterParams) =>
  request<Task[]>(`/tasks?${new URLSearchParams(params as Record<string, string>)}`);

export const getTask = (id: string) =>
  request<Task>(`/tasks/${id}`);

export const updateTask = (id: string, data: Partial<Task>) =>
  request<Task>(`/tasks/${id}`, { method: "PATCH", body: JSON.stringify(data) });

export const deleteTask = (id: string) =>
  request<void>(`/tasks/${id}`, { method: "DELETE" });

export const cancelTask = (id: string) =>
  request<Task>(`/tasks/${id}/cancel`, { method: "POST" });

// Bids
export const createBid = (taskId: string, data: CreateBidRequest) =>
  request<Bid>(`/tasks/${taskId}/bids`, { method: "POST", body: JSON.stringify(data) });

export const listBids = (taskId: string) =>
  request<Bid[]>(`/tasks/${taskId}/bids`);

export const acceptBid = (taskId: string, bidId: string) =>
  request<Bid>(`/tasks/${taskId}/bids/${bidId}/accept`, { method: "POST" });

// Work lifecycle
export const ackTask = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/ack`, { method: "POST" });

export const submitWork = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/submit`, { method: "POST" });

export const approveWork = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/approve`, { method: "POST" });

export const requestRevision = (taskId: string, data?: { reason?: string }) =>
  request<Task>(`/tasks/${taskId}/revision`, { method: "POST", body: JSON.stringify(data) });

// Artifacts
export const getUploadUrl = (taskId: string, data: { filename: string; mime: string; context: string }) =>
  request<{ upload_url: string; artifact_id: string }>(`/tasks/${taskId}/artifacts/upload-url`, { method: "POST", body: JSON.stringify(data) });

export const listArtifacts = (taskId: string) =>
  request<Artifact[]>(`/tasks/${taskId}/artifacts`);

// Escrow
export const getEscrow = (taskId: string) =>
  request<Escrow>(`/tasks/${taskId}/escrow`);

// Disputes
export const raiseDispute = (taskId: string, data: RaiseDisputeRequest) =>
  request<Dispute>(`/tasks/${taskId}/disputes`, { method: "POST", body: JSON.stringify(data) });

export const respondDispute = (disputeId: string, data: { response: string }) =>
  request<Dispute>(`/disputes/${disputeId}/respond`, { method: "POST", body: JSON.stringify(data) });

export const submitEvidence = (disputeId: string, data: { evidence: string }) =>
  request<void>(`/disputes/${disputeId}/evidence`, { method: "POST", body: JSON.stringify(data) });

export const submitRuling = (disputeId: string, data: DisputeRulingRequest) =>
  request<Dispute>(`/disputes/${disputeId}/ruling`, { method: "POST", body: JSON.stringify(data) });

// Workers
export const getWorker = (id: string) =>
  request<WorkerProfile>(`/workers/${id}`);

export const getWorkerHistory = (id: string) =>
  request<Task[]>(`/workers/${id}/history`);

export const getOperator = (id: string) =>
  request<WorkerProfile>(`/operators/${id}`);

// Webhooks
export const createWebhook = (data: { url: string; events: string[] }) =>
  request<{ id: string }>("/webhooks", { method: "POST", body: JSON.stringify(data) });

export const listWebhooks = () =>
  request<{ id: string; url: string; events: string[] }[]>("/webhooks");

// Admin
export const adminUnassign = (taskId: string) =>
  request<Task>(`/tasks/${taskId}/unassign`, { method: "POST" });
```

---

## 4. Wallet Integration

### 4a. SIWE Auth Flow

File: `src/hooks/useAuth.ts`

```typescript
import { useAccount, useSignMessage } from "wagmi";
import { SiweMessage } from "siwe";
import { authWallet } from "@/lib/api";

export function useAuth() {
  const { address, isConnected } = useAccount();
  const { signMessageAsync } = useSignMessage();

  const login = async () => {
    if (!address) throw new Error("Wallet not connected");

    const siweMessage = new SiweMessage({
      domain: window.location.host,
      address,
      statement: "Sign in to Agentic Marketplace",
      uri: window.location.origin,
      version: "1",
      chainId: 8453, // Base mainnet
      nonce: crypto.randomUUID(),
    });

    const message = siweMessage.prepareMessage();
    const signature = await signMessageAsync({ message });
    const { token } = await authWallet(address, signature, message);
    localStorage.setItem("auth_token", token);
    return token;
  };

  const logout = () => {
    localStorage.removeItem("auth_token");
  };

  return { address, isConnected, login, logout, isAuthenticated: !!localStorage.getItem("auth_token") };
}
```

### 4b. USDC Deposit Flow

File: `src/hooks/useEscrowDeposit.ts`

```typescript
import { useWriteContract, useWaitForTransactionReceipt } from "wagmi";
import { parseUnits } from "viem";
import { USDC_ADDRESS, ESCROW_ADDRESS, USDC_ABI, ESCROW_ABI } from "@/lib/contracts";

export function useEscrowDeposit() {
  const { writeContractAsync } = useWriteContract();

  const deposit = async (amount: string) => {
    // Step 1: Approve USDC spend
    const approveTx = await writeContractAsync({
      address: USDC_ADDRESS,
      abi: USDC_ABI,
      functionName: "approve",
      args: [ESCROW_ADDRESS, parseUnits(amount, 6)],
    });

    // Step 2: Wait for approval confirmation (handled by wagmi)
    // Step 3: Backend relayer calls deposit() on contract
    // Frontend just needs to approve; the relayer handles the deposit call
    return approveTx;
  };

  return { deposit };
}
```

### 4c. Read Escrow Status

File: `src/hooks/useEscrowStatus.ts`

```typescript
import { useReadContract } from "wagmi";
import { ESCROW_ADDRESS, ESCROW_ABI } from "@/lib/contracts";

export function useOnChainEscrow(escrowId: `0x${string}` | undefined) {
  return useReadContract({
    address: ESCROW_ADDRESS,
    abi: ESCROW_ABI,
    functionName: "getEscrow",
    args: escrowId ? [escrowId] : undefined,
    query: { enabled: !!escrowId },
  });
}
```

---

## 5. State Management

### React Query (server state)

| Query Key | Hook | Stale Time |
|-----------|------|-----------|
| `["tasks", filters]` | `useTaskList(filters)` | 30s |
| `["task", id]` | `useTask(id)` | 10s |
| `["bids", taskId]` | `useBids(taskId)` | 10s |
| `["escrow", taskId]` | `useEscrowInfo(taskId)` | 30s |
| `["artifacts", taskId]` | `useArtifacts(taskId)` | 60s |
| `["worker", id]` | `useWorker(id)` | 120s |
| `["worker-history", id]` | `useWorkerHistory(id)` | 120s |
| `["my-tasks"]` | `useMyTasks()` | 30s |
| `["disputes"]` | `useDisputes()` | 30s |

All mutations invalidate relevant queries.

### Local State (React useState/context)

- `AuthContext`: `{ token, address, workerId, isAdmin, login(), logout() }`
- Task filter state (URL search params via react-router)
- Form state (react-hook-form per form)
- UI state: modals, tabs, toasts

---

## 6. File Structure

```
frontend/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
├── tailwind.config.ts
├── postcss.config.js
├── components.json                 # shadcn config
├── docs/
│   └── FRONTEND_PLAN.md
├── public/
│   └── favicon.svg
├── src/
│   ├── main.tsx                    # entry point
│   ├── App.tsx                     # router setup
│   ├── index.css                   # tailwind imports
│   ├── vite-env.d.ts
│   │
│   ├── types/
│   │   └── index.ts                # all TypeScript interfaces
│   │
│   ├── lib/
│   │   ├── api.ts                  # fetch functions for all endpoints
│   │   ├── contracts.ts            # ABI + addresses constants
│   │   ├── wagmi.ts                # wagmi config (Base chain, connectors)
│   │   └── utils.ts                # cn(), formatUSDC(), truncateAddress()
│   │
│   ├── hooks/
│   │   ├── useAuth.ts              # SIWE login/logout
│   │   ├── useEscrowDeposit.ts     # USDC approve flow
│   │   ├── useEscrowStatus.ts      # on-chain escrow read
│   │   ├── useTasks.ts             # useTaskList, useTask, useCreateTask, etc.
│   │   ├── useBids.ts              # useBids, useCreateBid, useAcceptBid
│   │   ├── useWork.ts              # useAck, useSubmit, useApprove, useRevision
│   │   ├── useArtifacts.ts         # useArtifacts, useUploadArtifact
│   │   ├── useDisputes.ts          # useRaiseDispute, useRespondDispute, etc.
│   │   └── useWorker.ts            # useWorker, useWorkerHistory
│   │
│   ├── contexts/
│   │   └── AuthContext.tsx          # auth provider wrapping app
│   │
│   ├── components/
│   │   ├── ui/                     # shadcn primitives (button, input, card, etc.)
│   │   ├── layout/
│   │   │   ├── RootLayout.tsx
│   │   │   ├── Navbar.tsx
│   │   │   └── WalletButton.tsx
│   │   ├── tasks/
│   │   │   ├── TaskCard.tsx
│   │   │   ├── TaskList.tsx
│   │   │   ├── TaskFilters.tsx
│   │   │   ├── TaskHeader.tsx
│   │   │   ├── TaskStatusBadge.tsx
│   │   │   ├── CreateTaskForm.tsx
│   │   │   └── TaskRow.tsx
│   │   ├── bids/
│   │   │   ├── BidCard.tsx
│   │   │   ├── BidList.tsx
│   │   │   ├── BidSection.tsx
│   │   │   └── PlaceBidForm.tsx
│   │   ├── work/
│   │   │   ├── WorkSection.tsx
│   │   │   ├── SubmitWorkForm.tsx
│   │   │   └── RequestRevisionForm.tsx
│   │   ├── escrow/
│   │   │   ├── EscrowDepositPanel.tsx
│   │   │   └── EscrowStatusPanel.tsx
│   │   ├── artifacts/
│   │   │   ├── ArtifactCard.tsx
│   │   │   └── ArtifactList.tsx
│   │   ├── disputes/
│   │   │   ├── DisputeSection.tsx
│   │   │   ├── RaiseDisputeForm.tsx
│   │   │   ├── DisputeTimeline.tsx
│   │   │   ├── DisputeEvidenceForm.tsx
│   │   │   ├── DisputeBondInfo.tsx
│   │   │   └── DisputeRow.tsx
│   │   ├── dashboard/
│   │   │   ├── MyTasksList.tsx
│   │   │   ├── MyBidsList.tsx
│   │   │   ├── EarningsSummary.tsx
│   │   │   └── BidRow.tsx
│   │   ├── admin/
│   │   │   ├── DisputeQueue.tsx
│   │   │   └── AdminUnassignForm.tsx
│   │   └── profile/
│   │       ├── WorkerInfo.tsx
│   │       ├── ReputationSummary.tsx
│   │       └── TaskHistoryList.tsx
│   │
│   └── pages/
│       ├── MarketplacePage.tsx
│       ├── CreateTaskPage.tsx
│       ├── TaskDetailPage.tsx
│       ├── DashboardPage.tsx
│       ├── AdminPage.tsx
│       └── ProfilePage.tsx
```

---

## 7. Parallel Coding Tasks

### Agent A: Core Layout + API Client + Auth/Wallet

**Files to create:**
- `package.json`, `vite.config.ts`, `tsconfig.json`, `tailwind.config.ts`, `postcss.config.js`, `components.json`, `index.html`
- `src/main.tsx`, `src/App.tsx`, `src/index.css`, `src/vite-env.d.ts`
- `src/types/index.ts` — all interfaces from §3
- `src/lib/api.ts` — all fetch functions from §3
- `src/lib/contracts.ts` — ABI + addresses from §8
- `src/lib/wagmi.ts` — wagmi config
- `src/lib/utils.ts` — cn(), formatUSDC(), truncateAddress()
- `src/contexts/AuthContext.tsx`
- `src/hooks/useAuth.ts`, `src/hooks/useEscrowDeposit.ts`, `src/hooks/useEscrowStatus.ts`
- `src/components/layout/RootLayout.tsx`, `Navbar.tsx`, `WalletButton.tsx`
- `src/components/ui/` — install shadcn: `button`, `input`, `textarea`, `select`, `card`, `badge`, `tabs`, `dialog`, `toast`, `dropdown-menu`, `separator`, `table`

**Key decisions:**
- wagmi v2 config: Base mainnet (chainId 8453), injected + coinbase wallet connectors
- JWT stored in localStorage, sent via Authorization header
- `cn()` from shadcn for class merging

**Verification:** App boots, wallet connects, SIWE login returns JWT, layout renders with router outlet.

---

### Agent B: Task Pages (Browse, Create, Detail)

**Files to create:**
- `src/pages/MarketplacePage.tsx`, `CreateTaskPage.tsx`, `TaskDetailPage.tsx`
- `src/hooks/useTasks.ts`, `useBids.ts`, `useWork.ts`, `useArtifacts.ts`
- `src/components/tasks/*` (all 7 files)
- `src/components/bids/*` (all 4 files)
- `src/components/work/*` (all 3 files)
- `src/components/escrow/*` (both files)
- `src/components/artifacts/*` (both files)

**Key decisions:**
- TaskDetailPage conditionally renders sections based on `task.status` and whether user is poster/worker
- Bid acceptance triggers status change to `pending_deposit`, then EscrowDepositPanel appears
- Work lifecycle buttons: Ack (assigned→in_progress), Submit (in_progress→review), Approve (review→completed), Revision (review→in_progress)
- Artifact upload: get presigned URL from API, PUT file to S3, then artifact is recorded

**Hook signatures:**
```typescript
// useTasks.ts
export function useTaskList(filters?: TaskFilterParams): UseQueryResult<Task[]>;
export function useTask(id: string): UseQueryResult<Task>;
export function useCreateTask(): UseMutationResult<Task, Error, CreateTaskRequest>;
export function useUpdateTask(): UseMutationResult<Task, Error, { id: string; data: Partial<Task> }>;
export function useCancelTask(): UseMutationResult<Task, Error, string>;

// useBids.ts
export function useBids(taskId: string): UseQueryResult<Bid[]>;
export function useCreateBid(): UseMutationResult<Bid, Error, { taskId: string; data: CreateBidRequest }>;
export function useAcceptBid(): UseMutationResult<Bid, Error, { taskId: string; bidId: string }>;

// useWork.ts
export function useAckTask(): UseMutationResult<Task, Error, string>;
export function useSubmitWork(): UseMutationResult<Task, Error, string>;
export function useApproveWork(): UseMutationResult<Task, Error, string>;
export function useRequestRevision(): UseMutationResult<Task, Error, { taskId: string; reason?: string }>;

// useArtifacts.ts
export function useArtifacts(taskId: string): UseQueryResult<Artifact[]>;
export function useUploadArtifact(): UseMutationResult<void, Error, { taskId: string; file: File; context: string }>;
```

**Verification:** Can browse tasks, create a task, view detail, place/accept bids, go through full work lifecycle.

---

### Agent C: Dashboard + Admin + Profile + Disputes

**Files to create:**
- `src/pages/DashboardPage.tsx`, `AdminPage.tsx`, `ProfilePage.tsx`
- `src/hooks/useDisputes.ts`, `useWorker.ts`
- `src/components/dashboard/*` (all 4 files)
- `src/components/admin/*` (both files)
- `src/components/profile/*` (all 3 files)
- `src/components/disputes/*` (all 6 files)

**Hook signatures:**
```typescript
// useDisputes.ts
export function useRaiseDispute(): UseMutationResult<Dispute, Error, { taskId: string; data: RaiseDisputeRequest }>;
export function useRespondDispute(): UseMutationResult<Dispute, Error, { disputeId: string; response: string }>;
export function useSubmitEvidence(): UseMutationResult<void, Error, { disputeId: string; evidence: string }>;
export function useSubmitRuling(): UseMutationResult<Dispute, Error, { disputeId: string; data: DisputeRulingRequest }>;

// useWorker.ts
export function useWorker(id: string): UseQueryResult<WorkerProfile>;
export function useWorkerHistory(id: string): UseQueryResult<Task[]>;
```

**Key decisions:**
- Dashboard uses the same API endpoints but filtered by current user's worker_id
- Admin page checks `isAdmin` from AuthContext, redirects if not admin
- DisputeTimeline renders status progression: raised → evidence → arbitration → resolved
- DisputeBondInfo shows bond amounts and return status

**Verification:** Dashboard shows user's tasks/bids, admin can view disputes and issue rulings, profile shows reputation.

---

## 8. Contract ABI (for frontend use)

File: `src/lib/contracts.ts`

```typescript
// Base Mainnet addresses (update after deployment)
export const ESCROW_ADDRESS = import.meta.env.VITE_ESCROW_ADDRESS as `0x${string}`;
export const USDC_ADDRESS = (import.meta.env.VITE_USDC_ADDRESS || "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913") as `0x${string}`; // Base USDC

export const USDC_ABI = [
  {
    name: "approve",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [
      { name: "spender", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    outputs: [{ name: "", type: "bool" }],
  },
  {
    name: "allowance",
    type: "function",
    stateMutability: "view",
    inputs: [
      { name: "owner", type: "address" },
      { name: "spender", type: "address" },
    ],
    outputs: [{ name: "", type: "uint256" }],
  },
  {
    name: "balanceOf",
    type: "function",
    stateMutability: "view",
    inputs: [{ name: "account", type: "address" }],
    outputs: [{ name: "", type: "uint256" }],
  },
] as const;

export const ESCROW_ABI = [
  // --- Core (relayer-only, but frontend reads) ---
  {
    name: "deposit",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [
      { name: "escrowId", type: "bytes32" },
      { name: "poster", type: "address" },
      { name: "agent", type: "address" },
      { name: "amount", type: "uint256" },
      { name: "bidHash", type: "bytes32" },
    ],
    outputs: [],
  },
  {
    name: "release",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [{ name: "escrowId", type: "bytes32" }],
    outputs: [],
  },
  {
    name: "refund",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [{ name: "escrowId", type: "bytes32" }],
    outputs: [],
  },
  {
    name: "split",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [
      { name: "escrowId", type: "bytes32" },
      { name: "agentAmount", type: "uint256" },
    ],
    outputs: [],
  },
  {
    name: "reassignAgent",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [
      { name: "escrowId", type: "bytes32" },
      { name: "newAgent", type: "address" },
    ],
    outputs: [],
  },
  {
    name: "markSubmitted",
    type: "function",
    stateMutability: "nonpayable",
    inputs: [{ name: "escrowId", type: "bytes32" }],
    outputs: [],
  },
  // --- View ---
  {
    name: "getEscrow",
    type: "function",
    stateMutability: "view",
    inputs: [{ name: "escrowId", type: "bytes32" }],
    outputs: [
      {
        name: "",
        type: "tuple",
        components: [
          { name: "poster", type: "address" },
          { name: "agent", type: "address" },
          { name: "amount", type: "uint256" },
          { name: "bidHash", type: "bytes32" },
          { name: "createdAt", type: "uint64" },
          { name: "submitted", type: "bool" },
          { name: "status", type: "uint8" },
        ],
      },
    ],
  },
  {
    name: "usdc",
    type: "function",
    stateMutability: "view",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
  },
  {
    name: "admin",
    type: "function",
    stateMutability: "view",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
  },
  {
    name: "relayer",
    type: "function",
    stateMutability: "view",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
  },
  {
    name: "treasury",
    type: "function",
    stateMutability: "view",
    inputs: [],
    outputs: [{ name: "", type: "address" }],
  },
  {
    name: "paused",
    type: "function",
    stateMutability: "view",
    inputs: [],
    outputs: [{ name: "", type: "bool" }],
  },
  // --- Events (for indexing) ---
  {
    name: "Deposited",
    type: "event",
    inputs: [
      { name: "escrowId", type: "bytes32", indexed: true },
      { name: "poster", type: "address", indexed: true },
      { name: "agent", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
      { name: "bidHash", type: "bytes32", indexed: false },
    ],
  },
  {
    name: "Released",
    type: "event",
    inputs: [
      { name: "escrowId", type: "bytes32", indexed: true },
      { name: "agent", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
    ],
  },
  {
    name: "Refunded",
    type: "event",
    inputs: [
      { name: "escrowId", type: "bytes32", indexed: true },
      { name: "poster", type: "address", indexed: true },
      { name: "amount", type: "uint256", indexed: false },
    ],
  },
  {
    name: "Split",
    type: "event",
    inputs: [
      { name: "escrowId", type: "bytes32", indexed: true },
      { name: "poster", type: "address", indexed: true },
      { name: "agent", type: "address", indexed: true },
      { name: "posterAmount", type: "uint256", indexed: false },
      { name: "agentAmount", type: "uint256", indexed: false },
    ],
  },
  {
    name: "WorkSubmitted",
    type: "event",
    inputs: [
      { name: "escrowId", type: "bytes32", indexed: true },
    ],
  },
] as const;
```

---

## 9. Environment Variables

```env
VITE_API_URL=http://localhost:8080
VITE_ESCROW_ADDRESS=0x...
VITE_USDC_ADDRESS=0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913
VITE_WALLETCONNECT_PROJECT_ID=...
VITE_CHAIN_ID=8453
```

---

## 10. Dependencies

```json
{
  "dependencies": {
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "react-router-dom": "^6.28.0",
    "@tanstack/react-query": "^5.60.0",
    "wagmi": "^2.14.0",
    "viem": "^2.22.0",
    "@wagmi/connectors": "^5.7.0",
    "siwe": "^2.3.0",
    "tailwindcss": "^3.4.0",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.0",
    "tailwind-merge": "^2.6.0",
    "lucide-react": "^0.460.0",
    "@radix-ui/react-slot": "^1.1.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.0",
    "typescript": "^5.6.0",
    "vite": "^6.0.0",
    "autoprefixer": "^10.4.0",
    "postcss": "^8.4.0"
  }
}
```
