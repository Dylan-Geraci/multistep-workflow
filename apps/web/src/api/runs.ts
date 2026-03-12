import type { PaginatedResponse, Run } from "../types";
import { apiFetch } from "./client";

export async function createRun(
  workflowId: string,
  context?: Record<string, unknown>,
): Promise<Run> {
  return apiFetch(`/api/v1/workflows/${workflowId}/runs`, {
    method: "POST",
    body: JSON.stringify({ context: context ?? {} }),
  });
}

export async function listRuns(
  workflowId: string,
  limit = 20,
  offset = 0,
): Promise<PaginatedResponse<Run>> {
  return apiFetch(
    `/api/v1/workflows/${workflowId}/runs?limit=${limit}&offset=${offset}`,
  );
}

export async function getRun(id: string): Promise<Run> {
  return apiFetch(`/api/v1/runs/${id}`);
}

export async function cancelRun(id: string): Promise<Run> {
  return apiFetch(`/api/v1/runs/${id}/cancel`, { method: "POST" });
}
