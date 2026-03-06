interface Dispute {
  id: string;
  task_id: string;
  poster_worker_id: string;
  assigned_worker_id: string;
  raised_by_worker_id: string;
  reason: string;
  status: string;
  created_at: string;
}

interface DisputeCardProps {
  dispute: Dispute;
  onClick?: () => void;
}

export function DisputeCard({ dispute, onClick }: DisputeCardProps) {
  const statusColors: Record<string, string> = {
    raised: "bg-yellow-500/20 text-yellow-300 border-yellow-500/30",
    evidence: "bg-orange-500/20 text-orange-300 border-orange-500/30",
    arbitration: "bg-red-500/20 text-red-300 border-red-500/30",
    resolved: "bg-green-500/20 text-green-300 border-green-500/30",
  };

  return (
    <div
      className="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-gray-600 transition-colors cursor-pointer"
      onClick={onClick}
    >
      <div className="flex items-start justify-between mb-3">
        <div>
          <h3 className="text-gray-100 font-medium">Dispute #{dispute.id.slice(0, 8)}</h3>
          <p className="text-gray-400 text-sm mt-1">Task: {dispute.task_id.slice(0, 8)}</p>
        </div>
        <span
          className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium border ${
            statusColors[dispute.status] || "bg-gray-500/20 text-gray-300 border-gray-500/30"
          }`}
        >
          {dispute.status}
        </span>
      </div>
      
      <div className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-gray-400">Reason:</span>
          <span className="text-gray-200">{dispute.reason}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-400">Raised by:</span>
          <span className="text-gray-200">{dispute.raised_by_worker_id.slice(0, 8)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-400">Created:</span>
          <span className="text-gray-200">{new Date(dispute.created_at).toLocaleDateString()}</span>
        </div>
      </div>
    </div>
  );
}
