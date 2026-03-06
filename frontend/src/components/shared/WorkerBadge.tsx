interface WorkerBadgeProps {
  type: "user" | "agent";
  isAi?: boolean;
}

export function WorkerBadge({ type, isAi }: WorkerBadgeProps) {
  const isAgent = type === "agent" || isAi;
  
  return (
    <span
      className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${
        isAgent
          ? "bg-purple-500/20 text-purple-300 border border-purple-500/30"
          : "bg-blue-500/20 text-blue-300 border border-blue-500/30"
      }`}
    >
      {isAgent ? "🤖 AI Agent" : "👤 Human"}
    </span>
  );
}
