import { useState } from "react";

interface RulingFormProps {
  disputeId: string;
  onSubmit: (data: { outcome: string; agentBps?: number; rationale: string }) => void;
  disabled?: boolean;
}

export function RulingForm({ disputeId, onSubmit, disabled }: RulingFormProps) {
  const [outcome, setOutcome] = useState<"agent_wins" | "poster_wins" | "split">("agent_wins");
  const [agentBps, setAgentBps] = useState(5000); // 50%
  const [rationale, setRationale] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      outcome,
      agentBps: outcome === "split" ? agentBps : undefined,
      rationale,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-4">
      <h3 className="text-gray-100 font-medium mb-3">Issue Ruling</h3>

      {/* Outcome selector */}
      <div>
        <label className="block text-gray-300 text-sm font-medium mb-2">Outcome</label>
        <div className="space-y-2">
          <label className="flex items-center space-x-2 cursor-pointer">
            <input
              type="radio"
              value="agent_wins"
              checked={outcome === "agent_wins"}
              onChange={(e) => setOutcome(e.target.value as any)}
              className="text-blue-500"
            />
            <span className="text-gray-200">Release to Agent (Agent Wins)</span>
          </label>
          <label className="flex items-center space-x-2 cursor-pointer">
            <input
              type="radio"
              value="poster_wins"
              checked={outcome === "poster_wins"}
              onChange={(e) => setOutcome(e.target.value as any)}
              className="text-blue-500"
            />
            <span className="text-gray-200">Refund to Poster (Poster Wins)</span>
          </label>
          <label className="flex items-center space-x-2 cursor-pointer">
            <input
              type="radio"
              value="split"
              checked={outcome === "split"}
              onChange={(e) => setOutcome(e.target.value as any)}
              className="text-blue-500"
            />
            <span className="text-gray-200">Split Payment</span>
          </label>
        </div>
      </div>

      {/* Agent percentage slider (only for split) */}
      {outcome === "split" && (
        <div>
          <label className="block text-gray-300 text-sm font-medium mb-2">
            Agent Share: {(agentBps / 100).toFixed(0)}%
          </label>
          <input
            type="range"
            min="0"
            max="10000"
            step="100"
            value={agentBps}
            onChange={(e) => setAgentBps(parseInt(e.target.value))}
            className="w-full"
          />
          <div className="flex justify-between text-xs text-gray-400 mt-1">
            <span>0% (all to poster)</span>
            <span>100% (all to agent)</span>
          </div>
        </div>
      )}

      {/* Rationale textarea */}
      <div>
        <label className="block text-gray-300 text-sm font-medium mb-2">Rationale</label>
        <textarea
          value={rationale}
          onChange={(e) => setRationale(e.target.value)}
          required
          rows={4}
          className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-gray-100 placeholder-gray-500 focus:outline-none focus:border-blue-500"
          placeholder="Explain your ruling decision..."
        />
      </div>

      <button
        type="submit"
        disabled={disabled || !rationale.trim()}
        className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:cursor-not-allowed text-white font-medium py-2 px-4 rounded transition-colors"
      >
        Submit Ruling
      </button>
    </form>
  );
}
