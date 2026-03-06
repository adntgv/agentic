import type { Task } from '../../types';
import { StatusBadge } from './StatusBadge';

interface TaskCardProps {
  task: Task;
  onClick?: () => void;
}

export function TaskCard({ task, onClick }: TaskCardProps) {
  const deadline = task.deadline ? new Date(task.deadline).toLocaleDateString() : 'No deadline';
  
  return (
    <div
      onClick={onClick}
      className="bg-gray-800 border border-gray-700 rounded-lg p-6 hover:border-gray-600 transition-colors cursor-pointer"
    >
      <div className="flex items-start justify-between mb-3">
        <h3 className="text-xl font-semibold text-gray-100">{task.title}</h3>
        <StatusBadge status={task.status} />
      </div>
      
      <p className="text-gray-300 mb-4 line-clamp-3">{task.description}</p>
      
      <div className="flex items-center justify-between text-sm">
        <div className="flex gap-4">
          <div>
            <span className="text-gray-400">Budget:</span>{' '}
            <span className="text-green-400 font-medium">${task.budget} USDC</span>
          </div>
          {task.category && (
            <div>
              <span className="text-gray-400">Category:</span>{' '}
              <span className="text-gray-200">{task.category}</span>
            </div>
          )}
        </div>
        
        <div className="text-gray-400">
          Deadline: {deadline}
        </div>
      </div>
      
      <div className="mt-3 pt-3 border-t border-gray-700 flex gap-3 text-sm text-gray-400">
        <span>Worker: {task.worker_filter.replace('_', ' ')}</span>
        <span>•</span>
        <span>Max revisions: {task.max_revisions}</span>
      </div>
    </div>
  );
}
