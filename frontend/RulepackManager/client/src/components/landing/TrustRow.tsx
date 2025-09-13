export function TrustRow() {
  return (
    <div className="flex flex-wrap items-center gap-2.5 sm:gap-3 justify-center" aria-label="Trust badges">
      <div className="inline-flex items-center gap-2 rounded-full border px-2.5 py-1.5 text-[11px] sm:text-xs font-medium">Powered by ProtectAI DeBERTa v2</div>
      <div className="inline-flex items-center gap-2 rounded-full border px-2.5 py-1.5 text-[11px] sm:text-xs font-medium">SOC 2–aligned</div>
      <div className="inline-flex items-center gap-2 rounded-full border px-2.5 py-1.5 text-[11px] sm:text-xs font-medium">HIPAA‑aligned</div>
      <div className="inline-flex items-center gap-2 rounded-full border px-2.5 py-1.5 text-[11px] sm:text-xs font-medium">GDPR‑ready</div>
    </div>
  );
}

