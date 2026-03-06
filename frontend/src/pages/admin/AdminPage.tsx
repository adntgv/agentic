import { useState, useEffect } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { useNavigate } from "react-router-dom";
import { listDisputes, listTasks, adminUnassign, submitRuling } from "../../lib/api";
import type { Dispute } from "../../types";
import { DisputeCard } from "../../components/admin/DisputeCard";
import { RulingForm } from "../../components/admin/RulingForm";
import { StatsCard } from "../../components/dashboard/StatsCard";

type TabType = "disputes" | "abandoned" | "stats";

export function AdminPage() {
  const { isAdmin } = useAuth();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<TabType>("disputes");
  const [disputes, setDisputes] = useState<any[]>([]);
  const [abandonedTasks, setAbandonedTasks] = useState<any[]>([]);
  const [selectedDispute, setSelectedDispute] = useState<any | null>(null);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    totalEscrowLocked: 0,
    feesEarned: 0,
    disputeCount: 0,
    activeDisputes: 0,
  });

  useEffect(() => {
    if (!isAdmin) {
      navigate("/");
      return;
    }
    loadAdminData();
  }, [isAdmin, navigate]);

  const loadAdminData = async () => {
    setLoading(true);
    try {
      // Load disputes
      const disputesData = await listDisputes();
      setDisputes(disputesData || []);

      // Load abandoned/overdue tasks
      const tasksData = await listTasks({ status: "abandoned,overdue" });
      setAbandonedTasks(tasksData || []);

      // Calculate stats (simplified)
      setStats({
        totalEscrowLocked: 0, // TODO: sum all locked escrows
        feesEarned: 0, // TODO: calculate from completed tasks
        disputeCount: disputesData?.length || 0,
        activeDisputes: disputesData?.filter((d: Dispute) => d.status !== "resolved").length || 0,
      });
    } catch (error) {
      console.error("Failed to load admin data:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleRuling = async (disputeId: string, data: any) => {
    try {
      await submitRuling(disputeId, data);
      alert("Ruling submitted successfully");
      loadAdminData();
      setSelectedDispute(null);
    } catch (error) {
      console.error("Failed to submit ruling:", error);
      alert("Failed to submit ruling");
    }
  };

  const handleUnassign = async (taskId: string) => {
    if (!confirm("Unassign this task and return to bidding?")) return;
    
    try {
      await adminUnassign(taskId);
      alert("Task unassigned successfully");
      loadAdminData();
    } catch (error) {
      console.error("Failed to unassign task:", error);
      alert("Failed to unassign task");
    }
  };

  if (!isAdmin) {
    return null;
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 text-gray-100 flex items-center justify-center">
        <div className="text-xl">Loading admin panel...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold mb-8">Admin Panel</h1>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          <StatsCard label="Escrow Locked" value={`${stats.totalEscrowLocked} USDC`} icon="🔒" />
          <StatsCard label="Fees Earned" value={`${stats.feesEarned} USDC`} icon="💵" />
          <StatsCard label="Total Disputes" value={stats.disputeCount} icon="⚖️" />
          <StatsCard label="Active Disputes" value={stats.activeDisputes} icon="🔥" />
        </div>

        {/* Tabs */}
        <div className="border-b border-gray-700 mb-6">
          <div className="flex space-x-6">
            <button
              onClick={() => setActiveTab("disputes")}
              className={`pb-3 px-1 border-b-2 transition-colors ${
                activeTab === "disputes"
                  ? "border-blue-500 text-blue-400"
                  : "border-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              Disputes
            </button>
            <button
              onClick={() => setActiveTab("abandoned")}
              className={`pb-3 px-1 border-b-2 transition-colors ${
                activeTab === "abandoned"
                  ? "border-blue-500 text-blue-400"
                  : "border-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              Abandoned Tasks
            </button>
            <button
              onClick={() => setActiveTab("stats")}
              className={`pb-3 px-1 border-b-2 transition-colors ${
                activeTab === "stats"
                  ? "border-blue-500 text-blue-400"
                  : "border-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              Platform Stats
            </button>
          </div>
        </div>

        {/* Disputes Tab */}
        {activeTab === "disputes" && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Dispute list */}
            <div>
              <h2 className="text-xl font-semibold mb-4">Active Disputes</h2>
              {disputes.length === 0 ? (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-8 text-center text-gray-400">
                  No active disputes
                </div>
              ) : (
                <div className="space-y-3">
                  {disputes.map(dispute => (
                    <DisputeCard
                      key={dispute.id}
                      dispute={dispute}
                      onClick={() => setSelectedDispute(dispute)}
                    />
                  ))}
                </div>
              )}
            </div>

            {/* Ruling form */}
            <div>
              <h2 className="text-xl font-semibold mb-4">Dispute Details & Ruling</h2>
              {selectedDispute ? (
                <div className="space-y-4">
                  {/* Dispute info */}
                  <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <h3 className="text-gray-100 font-medium mb-3">Dispute #{selectedDispute.id.slice(0, 8)}</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-400">Task:</span>
                        <a href={`/tasks/${selectedDispute.task_id}`} className="text-blue-400 hover:underline">
                          {selectedDispute.task_id.slice(0, 8)}
                        </a>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Poster:</span>
                        <span className="text-gray-200">{selectedDispute.poster_worker_id.slice(0, 8)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Worker:</span>
                        <span className="text-gray-200">{selectedDispute.assigned_worker_id.slice(0, 8)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Raised by:</span>
                        <span className="text-gray-200">{selectedDispute.raised_by_worker_id.slice(0, 8)}</span>
                      </div>
                    </div>
                  </div>

                  {/* Evidence placeholder */}
                  <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <h4 className="text-gray-100 font-medium mb-2">Evidence</h4>
                    <p className="text-gray-400 text-sm">Evidence viewing would be implemented here</p>
                  </div>

                  {/* Ruling form */}
                  {selectedDispute.status !== "resolved" && (
                    <RulingForm
                      disputeId={selectedDispute.id}
                      onSubmit={(data) => handleRuling(selectedDispute.id, data)}
                    />
                  )}
                </div>
              ) : (
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-8 text-center text-gray-400">
                  Select a dispute to view details and issue ruling
                </div>
              )}
            </div>
          </div>
        )}

        {/* Abandoned Tasks Tab */}
        {activeTab === "abandoned" && (
          <div>
            <h2 className="text-xl font-semibold mb-4">Abandoned & Overdue Tasks</h2>
            {abandonedTasks.length === 0 ? (
              <div className="bg-gray-800 border border-gray-700 rounded-lg p-8 text-center text-gray-400">
                No abandoned or overdue tasks
              </div>
            ) : (
              <div className="space-y-3">
                {abandonedTasks.map(task => (
                  <div key={task.id} className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                    <div className="flex justify-between items-start mb-3">
                      <div>
                        <h3 className="text-gray-100 font-medium">{task.title}</h3>
                        <p className="text-gray-400 text-sm mt-1">Status: {task.status}</p>
                      </div>
                      <span className="text-blue-400 font-medium">{task.budget} USDC</span>
                    </div>
                    <div className="flex space-x-2">
                      <button
                        onClick={() => handleUnassign(task.id)}
                        className="bg-orange-600 hover:bg-orange-700 text-white font-medium py-2 px-4 rounded transition-colors text-sm"
                      >
                        Reassign
                      </button>
                      <a
                        href={`/tasks/${task.id}`}
                        className="bg-gray-700 hover:bg-gray-600 text-white font-medium py-2 px-4 rounded transition-colors text-sm inline-block"
                      >
                        View Details
                      </a>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Platform Stats Tab */}
        {activeTab === "stats" && (
          <div>
            <h2 className="text-xl font-semibold mb-4">Platform Statistics</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <StatsCard label="Total Escrow Locked" value={`${stats.totalEscrowLocked} USDC`} icon="🔒" />
              <StatsCard label="Platform Fees Earned" value={`${stats.feesEarned} USDC`} icon="💵" />
              <StatsCard label="Total Disputes" value={stats.disputeCount} icon="⚖️" />
              <StatsCard label="Active Disputes" value={stats.activeDisputes} icon="🔥" />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
