import { useEffect, useState } from "react";
import { Link } from "react-router";
import { listWorkflows } from "../api/workflows";
import Pagination from "../components/Pagination";
import type { Workflow } from "../types";

const LIMIT = 12;

export default function DashboardPage() {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    listWorkflows(LIMIT, offset)
      .then((res) => {
        setWorkflows(res.data);
        setTotal(res.total);
      })
      .finally(() => setLoading(false));
  }, [offset]);

  if (loading) {
    return <div className="flex justify-center py-12 text-gray-500">Loading workflows...</div>;
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Workflows</h1>
        <Link
          to="/workflows/new"
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
        >
          New Workflow
        </Link>
      </div>

      {workflows.length === 0 ? (
        <div className="text-center py-16 text-gray-500">
          <p className="text-lg mb-2">No workflows yet</p>
          <p className="text-sm">
            Create your first workflow to get started.
          </p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {workflows.map((wf) => (
              <Link
                key={wf.id}
                to={`/workflows/${wf.id}`}
                className="block p-4 bg-white rounded-lg shadow hover:shadow-md transition-shadow"
              >
                <h2 className="font-semibold text-lg mb-1 truncate">{wf.name}</h2>
                {wf.description && (
                  <p className="text-sm text-gray-600 mb-3 line-clamp-2">{wf.description}</p>
                )}
                <div className="flex items-center justify-between text-xs text-gray-400">
                  <span>{wf.steps?.length ?? 0} step{(wf.steps?.length ?? 0) !== 1 ? "s" : ""}</span>
                  <span>{new Date(wf.updated_at).toLocaleDateString()}</span>
                </div>
              </Link>
            ))}
          </div>
          <div className="mt-6">
            <Pagination total={total} limit={LIMIT} offset={offset} onChange={setOffset} />
          </div>
        </>
      )}
    </div>
  );
}
