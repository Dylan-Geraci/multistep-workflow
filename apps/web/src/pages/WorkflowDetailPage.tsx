import { useEffect, useState, useCallback } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { getWorkflow, deleteWorkflow } from "../api/workflows";
import { createRun, listRuns } from "../api/runs";
import RunStatusBadge from "../components/RunStatusBadge";
import Pagination from "../components/Pagination";
import type { Workflow, Run } from "../types";

const RUNS_LIMIT = 10;

export default function WorkflowDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [workflow, setWorkflow] = useState<Workflow | null>(null);
  const [runs, setRuns] = useState<Run[]>([]);
  const [runsTotal, setRunsTotal] = useState(0);
  const [runsOffset, setRunsOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [running, setRunning] = useState(false);

  useEffect(() => {
    setLoading(true);
    getWorkflow(id!)
      .then(setWorkflow)
      .catch(() => setError("Failed to load workflow"))
      .finally(() => setLoading(false));
  }, [id]);

  const fetchRuns = useCallback(() => {
    listRuns(id!, RUNS_LIMIT, runsOffset).then((res) => {
      setRuns(res.data);
      setRunsTotal(res.total);
    });
  }, [id, runsOffset]);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  async function handleRun() {
    setRunning(true);
    try {
      const run = await createRun(id!);
      navigate(`/runs/${run.id}`);
    } catch {
      setError("Failed to start run");
      setRunning(false);
    }
  }

  async function handleDelete() {
    if (!confirm("Are you sure you want to delete this workflow?")) return;
    setDeleting(true);
    try {
      await deleteWorkflow(id!);
      navigate("/");
    } catch {
      setError("Failed to delete workflow");
      setDeleting(false);
    }
  }

  if (loading) {
    return <div className="flex justify-center py-12 text-gray-500">Loading workflow...</div>;
  }

  if (!workflow) {
    return <div className="text-center py-12 text-red-600">{error || "Workflow not found"}</div>;
  }

  return (
    <div className="max-w-4xl mx-auto">
      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-700 rounded text-sm">{error}</div>
      )}

      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">{workflow.name}</h1>
          {workflow.description && (
            <p className="text-gray-600 mt-1">{workflow.description}</p>
          )}
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleRun}
            disabled={running}
            className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 disabled:opacity-50"
          >
            {running ? "Starting..." : "Run"}
          </button>
          <Link
            to={`/workflows/${id}/edit`}
            className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Edit
          </Link>
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50"
          >
            {deleting ? "Deleting..." : "Delete"}
          </button>
        </div>
      </div>

      {/* Steps */}
      <div className="bg-white rounded-lg shadow p-4 mb-8">
        <h2 className="text-lg font-semibold mb-3">Steps</h2>
        {workflow.steps && workflow.steps.length > 0 ? (
          <ol className="space-y-2">
            {workflow.steps.map((step, i) => (
              <li key={step.id ?? i} className="flex items-center gap-3 p-2 bg-gray-50 rounded">
                <span className="text-xs font-mono text-gray-400 w-6 text-right">{i + 1}</span>
                <span className="font-medium">{step.name || `Step ${i + 1}`}</span>
                <span className="text-xs px-2 py-0.5 bg-gray-200 rounded">{step.action}</span>
              </li>
            ))}
          </ol>
        ) : (
          <p className="text-gray-500 text-sm">No steps defined</p>
        )}
      </div>

      {workflow.retry_policy && (
        <div className="bg-white rounded-lg shadow p-4 mb-8">
          <h2 className="text-lg font-semibold mb-2">Retry Policy</h2>
          <div className="text-sm text-gray-600 space-y-1">
            <p>Max retries: {workflow.retry_policy.max_retries}</p>
            <p>Initial delay: {workflow.retry_policy.initial_delay_ms}ms</p>
            <p>Max delay: {workflow.retry_policy.max_delay_ms}ms</p>
            <p>Multiplier: {workflow.retry_policy.multiplier}</p>
          </div>
        </div>
      )}

      {/* Runs */}
      <div className="bg-white rounded-lg shadow p-4">
        <h2 className="text-lg font-semibold mb-3">Runs</h2>
        {runs.length === 0 ? (
          <p className="text-gray-500 text-sm">No runs yet</p>
        ) : (
          <>
            <div className="space-y-2">
              {runs.map((run) => (
                <Link
                  key={run.id}
                  to={`/runs/${run.id}`}
                  className="flex items-center justify-between p-3 bg-gray-50 rounded hover:bg-gray-100"
                >
                  <div className="flex items-center gap-3">
                    <RunStatusBadge status={run.status} />
                    <span className="text-sm font-mono text-gray-500">{run.id.slice(0, 8)}</span>
                  </div>
                  <span className="text-xs text-gray-400">
                    {new Date(run.created_at).toLocaleString()}
                  </span>
                </Link>
              ))}
            </div>
            <div className="mt-4">
              <Pagination
                total={runsTotal}
                limit={RUNS_LIMIT}
                offset={runsOffset}
                onChange={setRunsOffset}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
}
