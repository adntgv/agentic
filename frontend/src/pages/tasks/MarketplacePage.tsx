import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Task, TaskFilterParams } from '../../types';
import { listTasks } from '../../lib/api';
import { TaskCard } from '../../components/tasks/TaskCard';
import { useAuth } from '../../contexts/AuthContext';

export function MarketplacePage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<string>('');
  const [category, setCategory] = useState<string>('');
  const [budgetMin, setBudgetMin] = useState('');
  const [budgetMax, setBudgetMax] = useState('');
  const [workerFilter, setWorkerFilter] = useState<string>('');
  const [page, setPage] = useState(1);
  
  useEffect(() => {
    loadTasks();
  }, [status, category, budgetMin, budgetMax, workerFilter, page]);
  
  const loadTasks = async () => {
    try {
      setLoading(true);
      const filters: TaskFilterParams = {
        ...(status && { status: status as TaskFilterParams['status'] }),
        ...(category && { category }),
        ...(budgetMin && { budget_min: parseFloat(budgetMin) }),
        ...(budgetMax && { budget_max: parseFloat(budgetMax) }),
        ...(workerFilter && { worker_filter: workerFilter }),
        page,
        limit: 12
      };
      const data = await listTasks(filters);
      setTasks(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tasks');
    } finally {
      setLoading(false);
    }
  };
  
  const filteredTasks = search
    ? tasks.filter(t => 
        t.title.toLowerCase().includes(search.toLowerCase()) ||
        t.description.toLowerCase().includes(search.toLowerCase())
      )
    : tasks;
  
  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-4xl font-bold mb-2">Task Marketplace</h1>
            <p className="text-gray-400">Browse and bid on available tasks</p>
          </div>
          {isAuthenticated && (
            <button
              onClick={() => navigate('/tasks/new')}
              className="px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
            >
              Post Task
            </button>
          )}
        </div>
        
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 mb-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
            <input
              type="text"
              placeholder="Search tasks..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">All Statuses</option>
              <option value="published">Published</option>
              <option value="bidding">Accepting Bids</option>
              <option value="in_progress">In Progress</option>
              <option value="review">Under Review</option>
              <option value="completed">Completed</option>
            </select>
            <input
              type="text"
              placeholder="Category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <select
              value={workerFilter}
              onChange={(e) => setWorkerFilter(e.target.value)}
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">All Workers</option>
              <option value="human_only">Human Only</option>
              <option value="ai_only">AI Only</option>
              <option value="both">Both</option>
            </select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <input
              type="number"
              placeholder="Min Budget (USDC)"
              value={budgetMin}
              onChange={(e) => setBudgetMin(e.target.value)}
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="number"
              placeholder="Max Budget (USDC)"
              value={budgetMax}
              onChange={(e) => setBudgetMax(e.target.value)}
              className="px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>
        
        {loading ? (
          <div className="text-center py-12 text-gray-400">Loading tasks...</div>
        ) : error ? (
          <div className="bg-red-900/20 border border-red-700 rounded-lg p-6 text-center text-red-400">{error}</div>
        ) : filteredTasks.length === 0 ? (
          <div className="text-center py-12 text-gray-400">No tasks found</div>
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
              {filteredTasks.map((task) => (
                <TaskCard key={task.id} task={task} onClick={() => navigate(`/tasks/${task.id}`)} />
              ))}
            </div>
            <div className="flex items-center justify-center gap-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-4 py-2 bg-gray-800 hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed text-gray-100 rounded-md transition-colors"
              >
                Previous
              </button>
              <span className="px-4 py-2 text-gray-400">Page {page}</span>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={filteredTasks.length < 12}
                className="px-4 py-2 bg-gray-800 hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed text-gray-100 rounded-md transition-colors"
              >
                Next
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
