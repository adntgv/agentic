import type { Task, Bid } from '../../types';

interface DepositFlowProps {
  task: Task;
  acceptedBid?: Bid;
  onDeposit: () => void;
}

export function DepositFlow({ task, acceptedBid, onDeposit }: DepositFlowProps) {
  return (
    <div className="bg-yellow-900/20 border border-yellow-700 rounded-lg p-6">
      <h3 className="text-lg font-semibold text-yellow-400 mb-4">⚠️ Deposit Required</h3>
      
      <div className="space-y-4 text-gray-300">
        <p>
          To proceed with this task, you need to deposit <span className="font-bold text-yellow-400">${acceptedBid?.amount || task.budget} USDC</span> into escrow.
        </p>
        
        <div className="bg-gray-800 rounded-md p-4 space-y-2">
          <h4 className="font-medium text-gray-100">Steps to complete:</h4>
          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>Click "Approve USDC" to allow the escrow contract to access your funds</li>
            <li>Confirm the transaction in your wallet</li>
            <li>Wait for blockchain confirmation</li>
            <li>The deposit will be automatically processed</li>
          </ol>
        </div>
        
        <div className="bg-blue-900/20 border border-blue-700 rounded-md p-4">
          <p className="text-sm text-blue-300">
            <strong>🔒 Your funds are safe:</strong> The escrow smart contract locks your payment until the work is completed and approved. Neither party can access the funds until the task is resolved.
          </p>
        </div>
        
        <button
          onClick={onDeposit}
          className="w-full px-4 py-3 bg-yellow-600 hover:bg-yellow-700 text-white rounded-md font-medium transition-colors"
        >
          Approve USDC & Deposit
        </button>
      </div>
    </div>
  );
}
