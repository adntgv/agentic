import type { Bid } from '../../types';

interface BidCardProps {
  bid: Bid;
  isPoster: boolean;
  onAccept?: () => void;
}

export function BidCard({ bid, isPoster, onAccept }: BidCardProps) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="text-sm text-gray-400 mb-1">Worker ID: {bid.worker_id.slice(0, 8)}...</div>
          <div className="text-2xl font-bold text-green-400">${bid.amount} USDC</div>
        </div>
        {bid.eta_hours && (
          <div className="text-right">
            <div className="text-sm text-gray-400">Estimated time</div>
            <div className="text-lg font-medium text-gray-200">{bid.eta_hours}h</div>
          </div>
        )}
      </div>
      
      {bid.cover_letter && (
        <div className="mb-3">
          <div className="text-sm text-gray-400 mb-1">Cover letter:</div>
          <p className="text-gray-300 text-sm">{bid.cover_letter}</p>
        </div>
      )}
      
      <div className="flex items-center justify-between">
        <div className="text-xs text-gray-500">
          Submitted {new Date(bid.created_at).toLocaleDateString()}
        </div>
        
        {isPoster && bid.status === 'pending' && onAccept && (
          <button
            onClick={onAccept}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium transition-colors"
          >
            Accept Bid
          </button>
        )}
        
        {bid.status === 'accepted' && (
          <span className="text-sm font-medium text-green-400">✓ Accepted</span>
        )}
        {bid.status === 'rejected' && (
          <span className="text-sm font-medium text-red-400">✗ Rejected</span>
        )}
      </div>
    </div>
  );
}
