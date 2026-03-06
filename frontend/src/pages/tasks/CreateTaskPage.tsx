import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { CreateTaskRequest } from '../../types';
import { createTask, updateTask } from '../../lib/api';
import { useAuth } from '../../contexts/AuthContext';

export function CreateTaskPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const [formData, setFormData] = useState<CreateTaskRequest>({
    title: '',
    description: '',
    category: '',
    budget: '',
    deadline: '',
    bid_deadline: '',
    worker_filter: 'both',
    max_revisions: 2
  });
  
  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-gray-900 text-gray-100 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold mb-4">Authentication Required</h2>
          <p className="text-gray-400">Please connect your wallet to post tasks</p>
        </div>
      </div>
    );
  }
  
  const handleSubmit = async (shouldPublish: boolean = false) => {
    const budgetNum = parseFloat(formData.budget);
    if (isNaN(budgetNum) || budgetNum < 10) {
      setError('Budget must be at least $10 USDC');
      return;
    }
    
    try {
      setLoading(true);
      setError(null);
      const task = await createTask(formData);
      if (shouldPublish) {
        await updateTask(task.id, { status: 'published' });
      }
      navigate(`/tasks/${task.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create task');
    } finally {
      setLoading(false);
    }
  };
  
  const handleChange = (field: keyof CreateTaskRequest, value: string | number) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };
  
  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <div className="max-w-3xl mx-auto px-4 py-8">
        <div className="mb-8">
          <button onClick={() => navigate('/')} className="text-blue-400 hover:text-blue-300 mb-4 inline-flex items-center">
            ← Back to Marketplace
          </button>
          <h1 className="text-4xl font-bold mb-2">Post a New Task</h1>
          <p className="text-gray-400">Create a task and start receiving bids from workers</p>
        </div>
        
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 space-y-6">
          {error && (
            <div className="bg-red-900/20 border border-red-700 rounded-lg p-4 text-red-400">{error}</div>
          )}
          
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Task Title *</label>
            <input
              type="text"
              required
              value={formData.title}
              onChange={(e) => handleChange('title', e.target.value)}
              className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="e.g., Build a landing page for my product"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Description *</label>
            <textarea
              required
              value={formData.description}
              onChange={(e) => handleChange('description', e.target.value)}
              rows={6}
              className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Describe the task in detail..."
            />
          </div>
          
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Category</label>
              <input
                type="text"
                value={formData.category}
                onChange={(e) => handleChange('category', e.target.value)}
                className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="e.g., Web Development"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Budget (USDC) *</label>
              <input
                type="number"
                step="0.01"
                min="10"
                required
                value={formData.budget}
                onChange={(e) => handleChange('budget', e.target.value)}
                className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="0.00"
              />
              <p className="text-xs text-gray-500 mt-1">Minimum: $10 USDC</p>
            </div>
          </div>
          
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Task Deadline</label>
              <input
                type="datetime-local"
                value={formData.deadline}
                onChange={(e) => handleChange('deadline', e.target.value)}
                className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Bid Deadline</label>
              <input
                type="datetime-local"
                value={formData.bid_deadline}
                onChange={(e) => handleChange('bid_deadline', e.target.value)}
                className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>
          
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Worker Type</label>
              <select
                value={formData.worker_filter}
                onChange={(e) => handleChange('worker_filter', e.target.value)}
                className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="both">Both Human & AI</option>
                <option value="human_only">Human Only</option>
                <option value="ai_only">AI Only</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Max Revisions</label>
              <input
                type="number"
                min="0"
                max="10"
                value={formData.max_revisions}
                onChange={(e) => handleChange('max_revisions', parseInt(e.target.value))}
                className="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>
          
          <div className="bg-blue-900/20 border border-blue-700 rounded-md p-4">
            <p className="text-sm text-blue-300">
              <strong>Note:</strong> Your task will be created as a draft. You can review it and publish immediately, or save and publish later.
            </p>
          </div>
          
          <div className="flex gap-4">
            <button
              type="button"
              onClick={() => handleSubmit(false)}
              disabled={loading}
              className="flex-1 px-6 py-3 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
            >
              {loading ? 'Creating...' : 'Save as Draft'}
            </button>
            <button
              type="button"
              onClick={() => handleSubmit(true)}
              disabled={loading}
              className="flex-1 px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-800 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
            >
              {loading ? 'Creating...' : 'Create & Publish'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
