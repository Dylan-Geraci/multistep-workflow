import type {
  CreateWorkflowRequest,
  PaginatedResponse,
  Workflow,
} from "../types";
import { apiFetch } from "./client";

export async function listWorkflows(
  limit = 20,
  offset = 0,
): Promise<PaginatedResponse<Workflow>> {
  return apiFetch(`/api/v1/workflows?limit=${limit}&offset=${offset}`);
}

export async function getWorkflow(id: string): Promise<Workflow> {
  return apiFetch(`/api/v1/workflows/${id}`);
}

export async function createWorkflow(
  req: CreateWorkflowRequest,
): Promise<Workflow> {
  return apiFetch("/api/v1/workflows", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function updateWorkflow(
  id: string,
  req: CreateWorkflowRequest,
): Promise<Workflow> {
  return apiFetch(`/api/v1/workflows/${id}`, {
    method: "PUT",
    body: JSON.stringify(req),
  });
}

export async function deleteWorkflow(id: string): Promise<void> {
  return apiFetch(`/api/v1/workflows/${id}`, { method: "DELETE" });
}
