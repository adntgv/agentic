interface ReputationDisplayProps {
  avgRating: number;
  totalRatings: number;
  positiveCount: number;
  negativeCount: number;
  completionRate?: number;
  disputeRatio?: number;
}

export function ReputationDisplay({
  avgRating,
  totalRatings,
  positiveCount,
  negativeCount,
  completionRate,
  disputeRatio,
}: ReputationDisplayProps) {
  const stars = Math.round(avgRating);

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
      <h3 className="text-gray-100 font-medium mb-3">Reputation</h3>
      
      {/* Star rating */}
      <div className="flex items-center mb-4">
        <div className="flex text-2xl">
          {[1, 2, 3, 4, 5].map((i) => (
            <span key={i} className={i <= stars ? "text-yellow-400" : "text-gray-600"}>
              ★
            </span>
          ))}
        </div>
        <span className="ml-3 text-gray-300">
          {avgRating.toFixed(1)} ({totalRatings} {totalRatings === 1 ? "rating" : "ratings"})
        </span>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-3 text-sm">
        <div className="bg-gray-900 rounded p-3">
          <p className="text-gray-400">Positive</p>
          <p className="text-green-400 font-medium text-lg">{positiveCount}</p>
        </div>
        <div className="bg-gray-900 rounded p-3">
          <p className="text-gray-400">Negative</p>
          <p className="text-red-400 font-medium text-lg">{negativeCount}</p>
        </div>
        
        {completionRate !== undefined && (
          <div className="bg-gray-900 rounded p-3">
            <p className="text-gray-400">Completion</p>
            <p className="text-blue-400 font-medium text-lg">{(completionRate * 100).toFixed(0)}%</p>
          </div>
        )}
        
        {disputeRatio !== undefined && (
          <div className="bg-gray-900 rounded p-3">
            <p className="text-gray-400">Dispute Rate</p>
            <p className="text-orange-400 font-medium text-lg">{(disputeRatio * 100).toFixed(1)}%</p>
          </div>
        )}
      </div>
    </div>
  );
}
