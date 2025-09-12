export function TrustRow() {
  return (
    <div className="flex flex-wrap items-center gap-3 justify-center mt-4" aria-label="Trust badges">
      <div className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs">Powered by ProtectAI DeBERTa v2</div>
      <div className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs">SOC 2–aligned</div>
      <div className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs">HIPAA‑aligned</div>
      <div className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs">GDPR‑ready</div>
    </div>
  );
}

