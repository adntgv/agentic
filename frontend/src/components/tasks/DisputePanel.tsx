import { useState } from 'react';
import type { Dispute } from '../../types';

interface DisputePanelProps {
  dispute?: Dispute;
  canRaise: boolean;
  onRaise?: (reason: string) => void;
  onSubmitEvidence?: (evidence: string) => void;
}

export function DisputePanel({ dispute, canRaise, onRaise, onSubmitEvidence }: DisputePanelProps) {
  const [reason, setReason] = useState('incomplete');
  const [description, setDescription] = useState('');
  const [evidence, setEvidence] = useState('');

  if (!dispute && !canRaise) return null;

  if (!dispute && canRaise) {
    return (
      <div className="bg-red-900/20 border border-red-700 rounded-lg p-6">
        <h3 className="text-lg font-semibold text-red-400 mb-4">⚠️ Raise Dispute</h3>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Dispute Reason
            </label>
            <select
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-red-500"
            >
              <option value="incomplete">Work Incomplete</option>
              <option value="wrong_requirements">Wrong Requirements</option>
              <option value="quality">Quality Issues</option>
              <option value="fraud">Fraud</option>
            </select>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder="Explain the issue in detail..."
            />
          </div>
          
          <button
            onClick={() => onRaise?.(reason)}
            className="w-full px-4 py-3 bg-red-600 hover:bg-red-700 text-white rounded-md font-medium transition-colors"
          >
            Raise Dispute
          </button>
        </div>
      </div>
    );
  }

  if (dispute) {
    return (
      <div className="bg-red-900/20 border border-red-700 rounded-lg p-6">
        <h3 className="text-lg font-semibold text-red-400 mb-4">🚨 Dispute Active</h3>
        
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-gray-400">Status:</span>{' '}
              <span className="text-red-400 font-medium capitalize">{dispute.status}</span>
            </div>
            <div>
              <span className="text-gray-400">Reason:</span>{' '}
              <span className="text-gray-200 capitalize">{dispute.reason.replace('_', ' ')}</span>
            </div>
            <div>
              <span className="text-gray-400">Raised:</span>{' '}
              <span className="text-gray-200">{new Date(dispute.created_at).toLocaleDateString()}</span>
            </div>
            {dispute.outcome && (
              <div>
                <span className="text-gray-400">Outcome:</span>{' '}
                <span className="text-gray-200 capitalize">{dispute.outcome.replace('_', ' ')}</span>
              </div>
            )}
          </div>
          
          {dispute.rationale && (
            <div className="bg-gray-900 rounded-md p-4">
              <div className="text-sm font-medium text-gray-300 mb-2">Admin Rationale:</div>
              <p className="text-gray-400 text-sm">{dispute.rationale}</p>
            </div>
          )}
          
          {dispute.status === 'evidence' && onSubmitEvidence && (
            <div className="mt-4">
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Submit Evidence
              </label>
              <textarea
                value={evidence}
                onChange={(e) => setEvidence(e.target.value)}
                rows={4}
                className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-md text-gray-100 focus:outline-none focus:ring-2 focus:ring-red-500 mb-2"
                placeholder="Provide evidence for your case..."
              />
              <button
                onClick={() => {
                  onSubmitEvidence(evidence);
                  setEvidence('');
                }}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium"
              >
                Submit Evidence
              </button>
            </div>
          )}
        </div>
      </div>
    );
  }

  return null;
}
