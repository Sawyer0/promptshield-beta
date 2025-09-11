export function isDevBypassClient(): boolean {
  // Disabled by default. Enable only with explicit flag during development.
  try { return import.meta.env.VITE_ALLOW_DEV_BYPASS === 'true'; } catch { return false; }
}
