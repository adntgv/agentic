import type { TaskStatus } from '../../types';

const statusColors: Record<TaskStatus, string> = {
  draft: "bg-gray-600 text-gray-100",
  published: "bg-blue-600 text-white",
  bidding: "bg-blue-600 text-white",
  pending_deposit: "bg-yellow-600 text-white",
  assigned: "bg-purple-600 text-white",
  in_progress: "bg-indigo-600 text-white",
  review: "bg-orange-600 text-white",
  completed: "bg-green-600 text-white",
  refunded: "bg-gray-500 text-white",
  split: "bg-teal-600 text-white",
  disputed: "bg-red-600 text-white",
  abandoned: "bg-gray-700 text-gray-300",
  overdue: "bg-red-700 text-white",
  expired: "bg-gray-600 text-gray-300",
  cancelled: "bg-gray-600 text-gray-300",
  deleted: "bg-gray-800 text-gray-400"
};

const statusLabels: Record<TaskStatus, string> = {
  draft: "Draft",
  published: "Published",
  bidding: "Accepting Bids",
  pending_deposit: "Pending Deposit",
  assigned: "Assigned",
  in_progress: "In Progress",
  review: "Under Review",
  completed: "Completed",
  refunded: "Refunded",
  split: "Split Payment",
  disputed: "Disputed",
  abandoned: "Abandoned",
  overdue: "Overdue",
  expired: "Expired",
  cancelled: "Cancelled",
  deleted: "Deleted"
};

interface StatusBadgeProps {
  status: TaskStatus;
  className?: string;
}

export function StatusBadge({ status, className = "" }: StatusBadgeProps) {
  return (
    <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${statusColors[status]} ${className}`}>
      {statusLabels[status]}
    </span>
  );
}
