import type { StepAction } from "../types";
import StepCard from "./StepCard";

export interface StepData {
  name: string;
  action: StepAction;
  config: Record<string, unknown>;
}

interface Props {
  steps: StepData[];
  onChange: (steps: StepData[]) => void;
}

export default function StepBuilder({ steps, onChange }: Props) {
  const addStep = () => {
    onChange([...steps, { name: "", action: "log", config: {} }]);
  };

  const updateStep = (index: number, step: StepData) => {
    const next = [...steps];
    next[index] = step;
    onChange(next);
  };

  const removeStep = (index: number) => {
    onChange(steps.filter((_, i) => i !== index));
  };

  const moveStep = (from: number, to: number) => {
    if (to < 0 || to >= steps.length) return;
    const next = [...steps];
    const [item] = next.splice(from, 1);
    next.splice(to, 0, item!);
    onChange(next);
  };

  return (
    <div>
      <div className="space-y-3">
        {steps.map((step, i) => (
          <StepCard
            key={i}
            step={step}
            index={i}
            total={steps.length}
            onChange={(s) => updateStep(i, s)}
            onRemove={() => removeStep(i)}
            onMoveUp={() => moveStep(i, i - 1)}
            onMoveDown={() => moveStep(i, i + 1)}
          />
        ))}
      </div>
      <button
        type="button"
        onClick={addStep}
        className="mt-3 rounded border border-dashed border-gray-300 px-4 py-2 text-sm text-gray-500 hover:border-gray-400 hover:text-gray-700"
      >
        + Add Step
      </button>
    </div>
  );
}
