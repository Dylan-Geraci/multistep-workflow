import { useEffect, useRef } from "react";
import type { WSEvent } from "../types";
import { getAccessToken } from "../api/client";

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
const listeners = new Map<string, Set<(event: WSEvent) => void>>();

function getWsUrl(): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  const token = getAccessToken();
  return `${proto}//${window.location.host}/api/v1/ws${token ? `?token=${token}` : ""}`;
}

function connect() {
  if (ws?.readyState === WebSocket.OPEN || ws?.readyState === WebSocket.CONNECTING) return;

  ws = new WebSocket(getWsUrl());

  ws.onopen = () => {
    reconnectAttempt = 0;
    // Re-subscribe all active run IDs
    const runIds = Array.from(listeners.keys());
    if (runIds.length > 0) {
      ws?.send(JSON.stringify({ type: "subscribe", run_ids: runIds }));
    }
  };

  ws.onmessage = (e) => {
    try {
      const event: WSEvent = JSON.parse(e.data);
      const cbs = listeners.get(event.run_id);
      cbs?.forEach((cb) => cb(event));
    } catch {
      // ignore malformed messages
    }
  };

  ws.onclose = () => {
    ws = null;
    if (listeners.size > 0) {
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempt), 30000);
      reconnectAttempt++;
      reconnectTimer = setTimeout(connect, delay);
    }
  };

  ws.onerror = () => {
    ws?.close();
  };
}

function subscribe(runId: string, cb: (event: WSEvent) => void) {
  if (!listeners.has(runId)) {
    listeners.set(runId, new Set());
  }
  listeners.get(runId)!.add(cb);

  connect();

  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: "subscribe", run_ids: [runId] }));
  }
}

function unsubscribe(runId: string, cb: (event: WSEvent) => void) {
  const cbs = listeners.get(runId);
  if (cbs) {
    cbs.delete(cb);
    if (cbs.size === 0) {
      listeners.delete(runId);
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "unsubscribe", run_ids: [runId] }));
      }
    }
  }

  // Close connection if no more listeners
  if (listeners.size === 0) {
    if (reconnectTimer) clearTimeout(reconnectTimer);
    ws?.close();
    ws = null;
  }
}

export function useRunEvents(
  runId: string | undefined,
  onEvent: (event: WSEvent) => void,
) {
  const callbackRef = useRef(onEvent);
  callbackRef.current = onEvent;

  useEffect(() => {
    if (!runId) return;

    const handler = (event: WSEvent) => callbackRef.current(event);
    subscribe(runId, handler);
    return () => unsubscribe(runId, handler);
  }, [runId]);
}
