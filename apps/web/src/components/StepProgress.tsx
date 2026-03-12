import type { StepExecution } from "../types";

function statusIcon(status: string) {
  switch (status) {
    case "completed":
      return (
        <div className="flex h-6 w-6 items-center justify-center rounded-full bg-green-500 text-white text-xs">
          &#10003;
        </div>
      );
    case "failed":
      return (
        <div className="flex h-6 w-6 items-center justify-center rounded-full bg-red-500 text-white text-xs">
          &#10005;
        </div>
      );
    case "running":
      return (
        <div className="h-6 w-6 animate-pulse rounded-full bg-yellow-400" />
      );
    default:
      return <div className="h-6 w-6 rounded-full border-2 border-gray-300" />;
  }
}

export default function StepProgress({
  steps,
}: {
  steps: StepExecution[];
}) {
  if (steps.length === 0) {
    return <p className="text-sm text-gray-500">No steps executed yet.</p>;
  }

  return (
    <div className="space-y-0">
      {steps.map((step, i) => (
        <div key={step.id} className="flex gap-3">
          <div className="flex flex-col items-center">
            {statusIcon(step.status)}
            {i < steps.length - 1 && (
              <div className="w-px flex-1 bg-gray-200" />
            )}
          </div>
          <div className="pb-6">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-gray-900">
                Step {step.step_index}
              </span>
              <span className="text-xs text-gray-500">{step.action}</span>
              {step.attempt_number > 1 && (
                <span className="text-xs text-orange-600">
                  attempt {step.attempt_number}
                </span>
              )}
            </div>
            {step.duration_ms != null && (
              <p className="text-xs text-gray-500">{step.duration_ms}ms</p>
            )}
            {step.error_message && (
              <p className="mt-1 text-xs text-red-600">{step.error_message}</p>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
