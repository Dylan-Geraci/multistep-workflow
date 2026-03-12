import { useEffect, useState, type FormEvent } from "react";
import { useNavigate, useParams } from "react-router";
import { getWorkflow, updateWorkflow } from "../api/workflows";
import StepBuilder, { type StepData } from "../components/StepBuilder";
import { ApiError } from "../api/client";

export default function WorkflowEditPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [steps, setSteps] = useState<StepData[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getWorkflow(id!)
      .then((wf) => {
        setName(wf.name);
        setDescription(wf.description ?? "");
        setSteps(
          (wf.steps ?? []).map((s) => ({
            name: s.name,
            action: s.action,
            config: s.config,
          })),
        );
      })
      .catch((err) => {
        setError(err instanceof ApiError ? err.message : "Failed to load workflow");
      })
      .finally(() => setLoading(false));
  }, [id]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");

    if (!name.trim()) {
      setError("Name is required");
      return;
    }
    if (steps.length === 0) {
      setError("At least one step is required");
      return;
    }

    setSaving(true);
    try {
      await updateWorkflow(id!, {
        name: name.trim(),
        description: description.trim(),
        steps: steps.map((s, i) => ({
          step_index: i,
          action: s.action,
          name: s.name,
          config: s.config,
        })),
      });
      navigate(`/workflows/${id}`);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return <div className="flex justify-center py-12 text-gray-500">Loading workflow...</div>;
  }

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Edit Workflow</h1>
      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-700 rounded text-sm">{error}</div>
      )}
      <form onSubmit={handleSubmit} className="space-y-6">
        <div>
          <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
            Name
          </label>
          <input
            id="name"
            type="text"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
            Description
          </label>
          <textarea
            id="description"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-3">Steps</h2>
          <StepBuilder steps={steps} onChange={setSteps} />
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? "Saving..." : "Save Changes"}
          </button>
          <button
            type="button"
            onClick={() => navigate(`/workflows/${id}`)}
            className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
