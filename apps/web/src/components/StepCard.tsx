import type { StepAction } from "../types";

const ACTIONS: StepAction[] = ["log", "delay", "http_call", "transform"];

interface StepData {
  name: string;
  action: StepAction;
  config: Record<string, unknown>;
}

interface Props {
  step: StepData;
  index: number;
  total: number;
  onChange: (step: StepData) => void;
  onRemove: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
}

export default function StepCard({
  step,
  index,
  total,
  onChange,
  onRemove,
  onMoveUp,
  onMoveDown,
}: Props) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm font-medium text-gray-500">
          Step {index + 1}
        </span>
        <div className="flex gap-1">
          <button
            type="button"
            onClick={onMoveUp}
            disabled={index === 0}
            className="rounded px-1.5 py-0.5 text-xs text-gray-400 hover:text-gray-600 disabled:opacity-30"
          >
            &#9650;
          </button>
          <button
            type="button"
            onClick={onMoveDown}
            disabled={index === total - 1}
            className="rounded px-1.5 py-0.5 text-xs text-gray-400 hover:text-gray-600 disabled:opacity-30"
          >
            &#9660;
          </button>
          <button
            type="button"
            onClick={onRemove}
            className="rounded px-1.5 py-0.5 text-xs text-red-400 hover:text-red-600"
          >
            Remove
          </button>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-600">
            Name
          </label>
          <input
            type="text"
            value={step.name}
            onChange={(e) => onChange({ ...step, name: e.target.value })}
            className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-sm"
            placeholder="Step name"
          />
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-600">
            Action
          </label>
          <select
            value={step.action}
            onChange={(e) =>
              onChange({ ...step, action: e.target.value as StepAction })
            }
            className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-sm"
          >
            {ACTIONS.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="mt-3">
        <label className="mb-1 block text-xs font-medium text-gray-600">
          Config (JSON)
        </label>
        <textarea
          value={JSON.stringify(step.config, null, 2)}
          onChange={(e) => {
            try {
              onChange({ ...step, config: JSON.parse(e.target.value) });
            } catch {
              // Allow intermediate invalid JSON while typing
            }
          }}
          rows={3}
          className="w-full rounded border border-gray-300 px-2.5 py-1.5 font-mono text-sm"
        />
      </div>
    </div>
  );
}
