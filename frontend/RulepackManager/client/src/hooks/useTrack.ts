export function useTrack() {
  return (event: string, payload?: Record<string, any>) => {
    try {
      // Replace with your analytics later
      if (typeof window !== 'undefined') {
        (window as any).__ps_events = (window as any).__ps_events || [];
        (window as any).__ps_events.push({ event, payload, ts: Date.now() });
      }
    } catch {}
  };
}

