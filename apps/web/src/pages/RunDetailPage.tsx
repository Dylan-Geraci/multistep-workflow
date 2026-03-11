import { useEffect, useState, useCallback } from "react";
import { Link, useParams } from "react-router";
import { getRun, cancelRun } from "../api/runs";
import { useRunEvents } from "../hooks/useWebSocket";
import RunStatusBadge from "../components/RunStatusBadge";
import StepProgress from "../components/StepProgress";
import type { Run } from "../types";

export default function RunDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [run, setRun] = useState<Run | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [cancelling, setCancelling] = useState(false);

  const fetchRun = useCallback(() => {
    getRun(id!)
      .then(setRun)
      .catch(() => setError("Failed to load run"));
  }, [id]);

  useEffect(() => {
    setLoading(true);
    getRun(id!)
      .then(setRun)
      .catch(() => setError("Failed to load run"))
      .finally(() => setLoading(false));
  }, [id]);

  useRunEvents(id!, () => {
    fetchRun();
  });

  async function handleCancel() {
    setCancelling(true);
    try {
      await cancelRun(id!);
      fetchRun();
    } catch {
      setError("Failed to cancel run");
    } finally {
      setCancelling(false);
    }
  }

  if (loading) {
    return <div className="flex justify-center py-12 text-gray-500">Loading run...</div>;
  }

  if (!run) {
    return <div className="text-center py-12 text-red-600">{error || "Run not found"}</div>;
  }

  const isActive = run.status === "pending" || run.status === "running";

  return (
    <div className="max-w-3xl mx-auto">
      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-700 rounded text-sm">{error}</div>
      )}

      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="flex items-center gap-3 mb-1">
            <h1 className="text-2xl font-bold">Run</h1>
            <RunStatusBadge status={run.status} />
          </div>
          <p className="text-sm text-gray-500 font-mono">{run.id}</p>
        </div>
        <div className="flex gap-2">
          {isActive && (
            <button
              onClick={handleCancel}
              disabled={cancelling}
              className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50"
            >
              {cancelling ? "Cancelling..." : "Cancel"}
            </button>
          )}
          <Link
            to={`/workflows/${run.workflow_id}`}
            className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50"
          >
            View Workflow
          </Link>
        </div>
      </div>

      {/* Details */}
      <div className="bg-white rounded-lg shadow p-4 mb-6">
        <h2 className="text-lg font-semibold mb-3">Details</h2>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <dt className="text-gray-500">Workflow</dt>
          <dd>
            <Link to={`/workflows/${run.workflow_id}`} className="text-blue-600 hover:underline font-mono">
              {run.workflow_id.slice(0, 8)}
            </Link>
          </dd>
          <dt className="text-gray-500">Started</dt>
          <dd>{run.started_at ? new Date(run.started_at).toLocaleString() : "—"}</dd>
          <dt className="text-gray-500">Completed</dt>
          <dd>{run.completed_at ? new Date(run.completed_at).toLocaleString() : "—"}</dd>
          <dt className="text-gray-500">Current Step</dt>
          <dd>{run.current_step ?? "—"}</dd>
        </dl>
      </div>

      {run.error_message && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
          <h2 className="text-sm font-semibold text-red-800 mb-1">Error</h2>
          <p className="text-sm text-red-700 font-mono whitespace-pre-wrap">{run.error_message}</p>
        </div>
      )}

      {/* Step Progress */}
      <div className="bg-white rounded-lg shadow p-4">
        <h2 className="text-lg font-semibold mb-3">Step Progress</h2>
        {run.steps && run.steps.length > 0 ? (
          <StepProgress steps={run.steps} />
        ) : (
          <p className="text-gray-500 text-sm">No step executions yet</p>
        )}
      </div>
    </div>
  );
}
