import { useState } from 'react';
import type { CreateBidRequest } from '../../types';

interface BidFormProps {
  onSubmit: (data: CreateBidRequest) => void;
  disabled?: boolean;
}

export function BidForm({ onSubmit, disabled }: BidFormProps) {
  const [amount, setAmount] = useState('');
  const [etaHours, setEtaHours] = useState('');
  const [coverLetter, setCoverLetter] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      amount,
      eta_hours: etaHours ? parseInt(etaHours) : undefined,
      cover_letter: coverLetter || undefined
    });
  };

  return (
    <form onSubmit={handleSubmit} className="bg-gray-800 border border-gray-700 rounded-lg p-6">
      <h3 className="text-lg font-semibold text-gray-100 mb-4">Place Your Bid</h3>
      
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Bid Amount (USDC) *
          </label>
          <input
            type="number"
            step="0.01"
            min="0"
            required
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="0.00"
            disabled={disabled}
          />
        </div>
        
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Estimated Time (hours)
          </label>
          <input
            type="number"
            min="1"
            value={etaHours}
            onChange={(e) => setEtaHours(e.target.value)}
            className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="24"
            disabled={disabled}
          />
        </div>
        
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Cover Letter
          </label>
          <textarea
            value={coverLetter}
            onChange={(e) => setCoverLetter(e.target.value)}
            rows={4}
            className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Why you're the best fit for this task..."
            disabled={disabled}
          />
        </div>
        
        <button
          type="submit"
          disabled={disabled}
          className="w-full px-4 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-md font-medium transition-colors"
        >
          Submit Bid
        </button>
      </div>
    </form>
  );
}
