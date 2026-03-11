// Auth
export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface UserResponse {
  id: string;
  email: string;
  display_name: string;
  created_at: string;
  updated_at: string;
}

// Workflows
export type StepAction = "http_call" | "delay" | "log" | "transform";

export interface WorkflowStep {
  id: string;
  step_index: number;
  action: StepAction;
  config: Record<string, unknown>;
  name: string;
}

export interface RetryPolicy {
  max_retries: number;
  initial_delay_ms: number;
  max_delay_ms: number;
  multiplier: number;
}

export interface Workflow {
  id: string;
  user_id: string;
  name: string;
  description: string;
  retry_policy: RetryPolicy;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  steps?: WorkflowStep[];
}

export interface CreateWorkflowRequest {
  name: string;
  description: string;
  retry_policy?: RetryPolicy;
  steps: {
    action: StepAction;
    config: Record<string, unknown>;
    name: string;
  }[];
}

// Runs
export type RunStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface StepExecution {
  id: string;
  run_id: string;
  step_index: number;
  attempt_id: string;
  attempt_number: number;
  action: string;
  status: string;
  input: unknown;
  output: unknown;
  error_message?: string;
  duration_ms?: number;
  started_at?: string;
  completed_at?: string;
}

export interface Run {
  id: string;
  workflow_id: string;
  user_id: string;
  status: RunStatus;
  context: Record<string, unknown>;
  current_step: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  steps?: StepExecution[];
}

// WebSocket
export interface WSEvent {
  type: string;
  run_id: string;
  data: Record<string, unknown>;
}

export interface WSIncomingMessage {
  type: "subscribe" | "unsubscribe";
  run_ids: string[];
}

// Pagination
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

// Errors
export interface APIErrorResponse {
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
}
