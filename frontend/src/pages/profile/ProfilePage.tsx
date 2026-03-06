import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { useAuth } from "../../contexts/AuthContext";
import { getWorker, getWorkerHistory, getReputation, updateWorker } from "../../lib/api";
import { ReputationDisplay } from "../../components/profile/ReputationDisplay";
import { WorkerBadge } from "../../components/shared/WorkerBadge";

export function ProfilePage() {
  const { id } = useParams<{ id: string }>();
  const { workerId } = useAuth();
  const [worker, setWorker] = useState<any>(null);
  const [reputation, setReputation] = useState<any>(null);
  const [taskHistory, setTaskHistory] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [isEditing, setIsEditing] = useState(false);
  const [editForm, setEditForm] = useState({ display_name: "" });

  const isOwnProfile = id === workerId;

  useEffect(() => {
    if (id) {
      loadProfile(id);
    }
  }, [id]);

  const loadProfile = async (profileId: string) => {
    setLoading(true);
    try {
      const [workerData, reputationData, historyData] = await Promise.all([
        getWorker(profileId),
        getReputation(profileId).catch(() => null),
        getWorkerHistory(profileId).catch(() => []),
      ]);

      setWorker(workerData);
      setReputation(reputationData);
      setTaskHistory(historyData);
      setEditForm({ display_name: workerData.display_name || "" });
    } catch (error) {
      console.error("Failed to load profile:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveProfile = async () => {
    if (!id) return;
    
    try {
      await updateWorker(id, editForm);
      alert("Profile updated successfully");
      loadProfile(id);
      setIsEditing(false);
    } catch (error) {
      console.error("Failed to update profile:", error);
      alert("Failed to update profile");
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 text-gray-100 flex items-center justify-center">
        <div className="text-xl">Loading profile...</div>
      </div>
    );
  }

  if (!worker) {
    return (
      <div className="min-h-screen bg-gray-900 text-gray-100 flex items-center justify-center">
        <div className="text-xl text-red-400">Worker not found</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      <div className="max-w-5xl mx-auto px-4 py-8">
        {/* Worker Info Card */}
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 mb-6">
          <div className="flex items-start justify-between mb-4">
            <div>
              {isEditing ? (
                <input
                  type="text"
                  value={editForm.display_name}
                  onChange={(e) => setEditForm({ ...editForm, display_name: e.target.value })}
                  className="bg-gray-900 border border-gray-700 rounded px-3 py-2 text-gray-100 text-2xl font-bold"
                  placeholder="Display Name"
                />
              ) : (
                <h1 className="text-3xl font-bold">{worker.display_name || "Anonymous Worker"}</h1>
              )}
              <p className="text-gray-400 mt-1 font-mono text-sm">
                {worker.wallet_address?.slice(0, 6)}...{worker.wallet_address?.slice(-4)}
              </p>
            </div>
            <div className="flex items-center space-x-3">
              <WorkerBadge type={worker.worker_type} isAi={worker.is_ai} />
              {isOwnProfile && !isEditing && (
                <button
                  onClick={() => setIsEditing(true)}
                  className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded transition-colors"
                >
                  Edit Profile
                </button>
              )}
              {isEditing && (
                <div className="flex space-x-2">
                  <button
                    onClick={handleSaveProfile}
                    className="bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-4 rounded transition-colors"
                  >
                    Save
                  </button>
                  <button
                    onClick={() => {
                      setIsEditing(false);
                      setEditForm({ display_name: worker.display_name || "" });
                    }}
                    className="bg-gray-700 hover:bg-gray-600 text-white font-medium py-2 px-4 rounded transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* Worker type specific info */}
          {worker.worker_type === "agent" && worker.agent_id && (
            <div className="bg-gray-900 rounded p-3 mt-4">
              <p className="text-gray-400 text-sm">Agent ID</p>
              <p className="text-gray-200 font-mono">{worker.agent_id}</p>
            </div>
          )}

          {worker.worker_type === "agent" && (
            <div className="bg-gray-900 rounded p-3 mt-4">
              <p className="text-gray-400 text-sm">Operator</p>
              <p className="text-gray-200">{worker.operator || "Not specified"}</p>
            </div>
          )}
        </div>

        {/* Reputation */}
        {reputation && (
          <div className="mb-6">
            <ReputationDisplay
              avgRating={reputation.avg_rating || 0}
              totalRatings={reputation.total_ratings || 0}
              positiveCount={reputation.positive_count || 0}
              negativeCount={reputation.negative_count || 0}
              completionRate={reputation.completion_rate}
              disputeRatio={reputation.dispute_ratio}
            />
          </div>
        )}

        {/* Task History */}
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
          <h2 className="text-xl font-semibold mb-4">Task History</h2>
          
          {taskHistory.length === 0 ? (
            <div className="text-center text-gray-400 py-8">
              No completed tasks yet
            </div>
          ) : (
            <div className="space-y-3">
              {taskHistory.map(task => {
                const role = task.poster_worker_id === id ? "poster" : "worker";
                
                return (
                  <div key={task.id} className="bg-gray-900 rounded-lg p-4">
                    <div className="flex justify-between items-start mb-2">
                      <div>
                        <h3 className="text-gray-100 font-medium">{task.title}</h3>
                        <p className="text-gray-400 text-sm mt-1">
                          Role: <span className="text-blue-400">{role === "poster" ? "Task Poster" : "Worker"}</span>
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-blue-400 font-medium">{task.budget} USDC</p>
                        <p className="text-gray-400 text-sm">{task.status}</p>
                      </div>
                    </div>
                    
                    {task.created_at && (
                      <p className="text-gray-500 text-xs">
                        {new Date(task.created_at).toLocaleDateString()}
                      </p>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
