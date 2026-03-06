import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import type { Task, Bid, Escrow, Artifact, Dispute } from '../../types';
import {
  getTask,
  updateTask,
  listBids,
  createBid,
  acceptBid,
  getEscrow,
  ackTask,
  submitTask,
  approveTask,
  requestRevision,
  listArtifacts,
  raiseDispute
} from '../../lib/api';
import { StatusBadge } from '../../components/tasks/StatusBadge';
import { BidCard } from '../../components/tasks/BidCard';
import { BidForm } from '../../components/tasks/BidForm';
import { ArtifactList } from '../../components/tasks/ArtifactList';
import { DepositFlow } from '../../components/tasks/DepositFlow';
import { DisputePanel } from '../../components/tasks/DisputePanel';
import { RatingForm } from '../../components/tasks/RatingForm';
import { useAuth } from '../../contexts/AuthContext';

export function TaskDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { worker } = useAuth();
  
  const [task, setTask] = useState<Task | null>(null);
  const [bids, setBids] = useState<Bid[]>([]);
  const [escrow, setEscrow] = useState<Escrow | null>(null);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [dispute, setDispute] = useState<Dispute | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  
  useEffect(() => {
    if (id) loadTaskData();
  }, [id]);
  
  const loadTaskData = async () => {
    if (!id) return;
    try {
      setLoading(true);
      const taskData = await getTask(id);
      setTask(taskData);
      
      // Load bids
      try { setBids(await listBids(id)); } catch { /* no bids */ }
      
      // Load artifacts
      try { setArtifacts(await listArtifacts(id)); } catch { /* no artifacts */ }
      
      // Load escrow if applicable
      if (['pending_deposit', 'assigned', 'in_progress', 'review', 'completed', 'disputed'].includes(taskData.status)) {
        try { setEscrow(await getEscrow(id)); } catch { /* no escrow */ }
      }
      
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load task');
    } finally {
      setLoading(false);
    }
  };
  
  const doAction = async (fn: () => Promise<unknown>, confirmMsg?: string) => {
    if (confirmMsg && !confirm(confirmMsg)) return;
    try {
      setActionLoading(true);
      await fn();
      await loadTaskData();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };
  
  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 text-gray-100 flex items-center justify-center">
        <div className="text-gray-400">Loading task...</div>
      </div>
    );
  }
  
  if (error || !task) {
    return (
      <div className="min-h-screen bg-gray-900 text-gray-100 flex items-center justify-center">
        <div className="text-center">
          <div className="text-red-400 mb-4">{error || 'Task not found'}</div>
          <button onClick={() => navigate('/')} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md">
            Back to Marketplace
          </button>
        </div>
      </div>
    );
  }
  
  const workerId = worker?.id;
  const isPoster = workerId === task.poster_worker_id;
  const isWorker = workerId === task.assigned_worker_id;
  const acceptedBid = bids.find(b => b.id === task.accepted_bid_id);
  
  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <div className="max-w-5xl mx-auto px-4 py-8">
        <button onClick={() => navigate('/')} className="text-blue-400 hover:text-blue-300 mb-4 inline-flex items-center">
          ← Back to Marketplace
        </button>
        
        {/* Task Header */}
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 mb-6">
          <div className="flex items-start justify-between mb-4">
            <h1 className="text-3xl font-bold">{task.title}</h1>
            <StatusBadge status={task.status} />
          </div>
          <p className="text-gray-300 mb-6 whitespace-pre-wrap">{task.description}</p>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-gray-400">Budget:</span>{' '}
              <span className="text-green-400 font-bold">${task.budget} USDC</span>
            </div>
            {task.category && (
              <div>
                <span className="text-gray-400">Category:</span>{' '}
                <span className="text-gray-200">{task.category}</span>
              </div>
            )}
            <div>
              <span className="text-gray-400">Worker Type:</span>{' '}
              <span className="text-gray-200">{task.worker_filter.replace('_', ' ')}</span>
            </div>
            <div>
              <span className="text-gray-400">Revisions:</span>{' '}
              <span className="text-gray-200">{task.revision_count} / {task.max_revisions}</span>
            </div>
            {task.deadline && (
              <div>
                <span className="text-gray-400">Deadline:</span>{' '}
                <span className="text-gray-200">{new Date(task.deadline).toLocaleDateString()}</span>
              </div>
            )}
          </div>
        </div>
        
        {/* Draft → Publish */}
        {task.status === 'draft' && isPoster && (
          <div className="bg-yellow-900/20 border border-yellow-700 rounded-lg p-6 mb-6">
            <h3 className="text-lg font-semibold text-yellow-400 mb-4">📝 Draft Task</h3>
            <p className="text-gray-300 mb-4">This task is in draft mode. Publish it to start receiving bids.</p>
            <button
              onClick={() => doAction(() => updateTask(task.id, { status: 'published' }))}
              disabled={actionLoading}
              className="px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
            >
              {actionLoading ? 'Publishing...' : 'Publish Task'}
            </button>
          </div>
        )}
        
        {/* Escrow Info */}
        {escrow && (
          <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 mb-6">
            <h3 className="text-lg font-semibold text-gray-100 mb-4">🔒 Escrow Status</h3>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-400">Amount:</span>{' '}
                <span className="text-green-400 font-medium">${escrow.amount} USDC</span>
              </div>
              <div>
                <span className="text-gray-400">Status:</span>{' '}
                <span className="text-gray-200 capitalize">{escrow.status}</span>
              </div>
              {escrow.deposit_tx_hash && (
                <div className="col-span-2">
                  <span className="text-gray-400">Tx Hash:</span>{' '}
                  <span className="text-blue-400 text-xs">{escrow.deposit_tx_hash}</span>
                </div>
              )}
            </div>
          </div>
        )}
        
        {/* Bidding Phase */}
        {['published', 'bidding'].includes(task.status) && (
          <div className="space-y-6 mb-6">
            <div>
              <h3 className="text-xl font-semibold mb-4">Bids ({bids.length})</h3>
              {bids.length === 0 ? (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 text-center text-gray-400">No bids yet</div>
              ) : (
                <div className="space-y-3">
                  {bids.map((bid) => (
                    <BidCard
                      key={bid.id}
                      bid={bid}
                      isPoster={isPoster}
                      onAccept={() => doAction(() => acceptBid(task.id, bid.id), 'Accept this bid?')}
                    />
                  ))}
                </div>
              )}
            </div>
            {!isPoster && workerId && (
              <BidForm
                onSubmit={(data) => doAction(() => createBid(task.id, data))}
                disabled={actionLoading}
              />
            )}
          </div>
        )}
        
        {/* Pending Deposit */}
        {task.status === 'pending_deposit' && isPoster && (
          <div className="mb-6">
            <DepositFlow
              task={task}
              acceptedBid={acceptedBid}
              onDeposit={() => alert('Please approve USDC in your wallet. Deposit will be handled by the escrow contract.')}
            />
          </div>
        )}
        
        {/* Assigned → ACK */}
        {task.status === 'assigned' && isWorker && (
          <div className="bg-purple-900/20 border border-purple-700 rounded-lg p-6 mb-6">
            <h3 className="text-lg font-semibold text-purple-400 mb-4">✋ Acknowledge Task</h3>
            <p className="text-gray-300 mb-4">Confirm that you're ready to start working.</p>
            <button
              onClick={() => doAction(() => ackTask(task.id), 'Start working on this task?')}
              disabled={actionLoading}
              className="px-6 py-3 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
            >
              {actionLoading ? 'Acknowledging...' : 'Acknowledge & Start Work'}
            </button>
          </div>
        )}
        
        {/* In Progress → Submit */}
        {task.status === 'in_progress' && isWorker && (
          <div className="bg-blue-900/20 border border-blue-700 rounded-lg p-6 mb-6">
            <h3 className="text-lg font-semibold text-blue-400 mb-4">🚀 Work in Progress</h3>
            {task.deadline && (
              <div className="text-sm text-gray-400 mb-4">
                Deadline: {new Date(task.deadline).toLocaleString()}
              </div>
            )}
            <button
              onClick={() => doAction(() => submitTask(task.id), 'Submit work for review?')}
              disabled={actionLoading}
              className="px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
            >
              {actionLoading ? 'Submitting...' : 'Submit Work for Review'}
            </button>
          </div>
        )}
        
        {/* Review Phase */}
        {task.status === 'review' && isPoster && (
          <div className="bg-orange-900/20 border border-orange-700 rounded-lg p-6 mb-6">
            <h3 className="text-lg font-semibold text-orange-400 mb-4">👀 Review Work</h3>
            <div className="flex gap-3">
              <button
                onClick={() => doAction(() => approveTask(task.id), 'Approve and release payment?')}
                disabled={actionLoading}
                className="px-6 py-3 bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
              >
                Approve & Release Payment
              </button>
              <button
                onClick={() => {
                  const reason = prompt('Enter revision feedback:');
                  if (reason) doAction(() => requestRevision(task.id, { reason }));
                }}
                disabled={actionLoading || task.revision_count >= task.max_revisions}
                className="px-6 py-3 bg-yellow-600 hover:bg-yellow-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
              >
                Request Revision ({task.max_revisions - task.revision_count} left)
              </button>
              <button
                onClick={() => doAction(() => raiseDispute(task.id, { reason: 'quality' }), 'Raise a dispute?')}
                disabled={actionLoading}
                className="px-6 py-3 bg-red-600 hover:bg-red-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
              >
                Raise Dispute
              </button>
            </div>
          </div>
        )}
        
        {/* Artifacts */}
        {artifacts.length > 0 && (
          <div className="mb-6">
            <ArtifactList artifacts={artifacts} />
          </div>
        )}
        
        {/* Dispute */}
        {task.status === 'disputed' && (
          <div className="mb-6">
            <DisputePanel
              dispute={dispute || undefined}
              canRaise={false}
            />
          </div>
        )}
        
        {/* Completed → Rating */}
        {task.status === 'completed' && (isPoster || isWorker) && (
          <div className="mb-6">
            <RatingForm onSubmit={(rating, comment) => console.log('Rating:', { rating, comment })} />
          </div>
        )}
      </div>
    </div>
  );
}
