import { useState, useEffect } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { listTasks, listBids, getEscrow } from "../../lib/api";
import { StatsCard } from "../../components/dashboard/StatsCard";

type TabType = "my-tasks" | "my-work";

export function DashboardPage() {
  const { workerId } = useAuth();
  const [activeTab, setActiveTab] = useState<TabType>("my-tasks");
  const [myTasks, setMyTasks] = useState<any[]>([]);
  const [myBids, setMyBids] = useState<any[]>([]);
  const [myWork, setMyWork] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    totalTasks: 0,
    activeTasks: 0,
    completedTasks: 0,
    inDispute: 0,
    totalEarned: 0,
    pendingEarnings: 0,
  });

  useEffect(() => {
    loadDashboardData();
  }, [workerId]);

  const loadDashboardData = async () => {
    if (!workerId) return;
    
    setLoading(true);
    try {
      // Load tasks I posted
      const tasksData = await listTasks({ poster_worker_id: workerId });
      setMyTasks(tasksData || []);

      // Load my bids
      // Note: This is a simplified version. Full implementation would need a /workers/:id/bids endpoint
      const bidsData: any[] = []; // TODO: implement proper endpoint
      setMyBids(bidsData);

      // Load tasks I'm working on
      const workData = await listTasks({ assigned_worker_id: workerId });
      setMyWork(workData || []);

      // Calculate stats
      const tasks = tasksData || [];
      const work = workData || [];
      
      setStats({
        totalTasks: tasks.length,
        activeTasks: tasks.filter(t => ["published", "bidding", "assigned", "in_progress", "review"].includes(t.status)).length,
        completedTasks: tasks.filter(t => t.status === "completed").length,
        inDispute: tasks.filter(t => t.status === "disputed").length,
        totalEarned: work.filter(t => t.status === "completed").reduce((sum, t) => sum + parseFloat(t.budget || "0"), 0),
        pendingEarnings: work.filter(t => ["in_progress", "review"].includes(t.status)).reduce((sum, t) => sum + parseFloat(t.budget || "0"), 0),
      });
    } catch (error) {
      console.error("Failed to load dashboard:", error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 text-gray-100 flex items-center justify-center">
        <div className="text-xl">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold mb-8">Dashboard</h1>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          <StatsCard label="Total Tasks" value={stats.totalTasks} icon="📋" />
          <StatsCard label="Active" value={stats.activeTasks} icon="⚡" />
          <StatsCard label="Completed" value={stats.completedTasks} icon="✅" />
          <StatsCard label="In Dispute" value={stats.inDispute} icon="⚠️" />
        </div>

        {/* Tabs */}
        <div className="border-b border-gray-700 mb-6">
          <div className="flex space-x-6">
            <button
              onClick={() => setActiveTab("my-tasks")}
              className={`pb-3 px-1 border-b-2 transition-colors ${
                activeTab === "my-tasks"
                  ? "border-blue-500 text-blue-400"
                  : "border-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              My Tasks (Posted)
            </button>
            <button
              onClick={() => setActiveTab("my-work")}
              className={`pb-3 px-1 border-b-2 transition-colors ${
                activeTab === "my-work"
                  ? "border-blue-500 text-blue-400"
                  : "border-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              My Work (Worker)
            </button>
          </div>
        </div>

        {/* Tab Content */}
        {activeTab === "my-tasks" && (
          <div>
            <h2 className="text-xl font-semibold mb-4">Tasks I Posted</h2>
            
            {myTasks.length === 0 ? (
              <div className="bg-gray-800 border border-gray-700 rounded-lg p-8 text-center text-gray-400">
                No tasks posted yet. <a href="/tasks/new" className="text-blue-400 hover:underline">Create your first task</a>
              </div>
            ) : (
              <div className="space-y-4">
                {/* Group by status */}
                {["published", "bidding", "assigned", "in_progress", "review", "completed", "disputed"].map(status => {
                  const tasksInStatus = myTasks.filter(t => t.status === status);
                  if (tasksInStatus.length === 0) return null;
                  
                  return (
                    <div key={status}>
                      <h3 className="text-lg font-medium text-gray-300 mb-3 capitalize">{status.replace("_", " ")}</h3>
                      <div className="space-y-2">
                        {tasksInStatus.map(task => (
                          <a
                            key={task.id}
                            href={`/tasks/${task.id}`}
                            className="block bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-gray-600 transition-colors"
                          >
                            <div className="flex justify-between items-start">
                              <div>
                                <h4 className="text-gray-100 font-medium">{task.title}</h4>
                                <p className="text-gray-400 text-sm mt-1">{task.description?.slice(0, 100)}...</p>
                              </div>
                              <span className="text-blue-400 font-medium">{task.budget} USDC</span>
                            </div>
                          </a>
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {activeTab === "my-work" && (
          <div>
            <h2 className="text-xl font-semibold mb-4">My Work</h2>
            
            {/* Earnings Summary */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <StatsCard label="Total Earned" value={`${stats.totalEarned.toFixed(2)} USDC`} icon="💰" />
              <StatsCard label="Pending Earnings" value={`${stats.pendingEarnings.toFixed(2)} USDC`} icon="⏳" />
            </div>

            {/* My Bids */}
            <div className="mb-8">
              <h3 className="text-lg font-medium text-gray-300 mb-3">My Bids</h3>
              {myBids.length === 0 ? (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 text-center text-gray-400">
                  No bids placed yet
                </div>
              ) : (
                <div className="space-y-2">
                  {myBids.map(bid => (
                    <div key={bid.id} className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                      <div className="flex justify-between">
                        <span className="text-gray-200">{bid.amount} USDC</span>
                        <span className={`text-sm ${
                          bid.status === "accepted" ? "text-green-400" :
                          bid.status === "rejected" ? "text-red-400" :
                          "text-yellow-400"
                        }`}>
                          {bid.status}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Tasks I'm Working On */}
            <div>
              <h3 className="text-lg font-medium text-gray-300 mb-3">Active Work</h3>
              {myWork.length === 0 ? (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 text-center text-gray-400">
                  No active work. <a href="/" className="text-blue-400 hover:underline">Browse tasks</a>
                </div>
              ) : (
                <div className="space-y-2">
                  {myWork.map(task => (
                    <a
                      key={task.id}
                      href={`/tasks/${task.id}`}
                      className="block bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-gray-600 transition-colors"
                    >
                      <div className="flex justify-between items-start">
                        <div>
                          <h4 className="text-gray-100 font-medium">{task.title}</h4>
                          <p className="text-gray-400 text-sm mt-1">Status: {task.status}</p>
                        </div>
                        <span className="text-blue-400 font-medium">{task.budget} USDC</span>
                      </div>
                    </a>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
