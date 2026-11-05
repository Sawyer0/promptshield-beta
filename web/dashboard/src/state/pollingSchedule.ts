/**
 * Runs `task` now and then every `intervalMs` until the returned stop
 * function is called. Kept outside React so hooks can treat polling as an
 * external subscription.
 */
export function startPolling(task: () => void, intervalMs: number): () => void {
  const timer = setInterval(task, intervalMs);
  queueMicrotask(task);
  return () => clearInterval(timer);
}
