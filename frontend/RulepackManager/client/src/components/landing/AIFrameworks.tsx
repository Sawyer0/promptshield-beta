export function AIFrameworks() {
  const items = [
    { code: "NIST AI RMF 1.0", desc: "Risk management and governance for AI systems." },
    { code: "EU AI Act (readiness)", desc: "Controls alignment for high‑risk categories." },
    { code: "ISO/IEC 42001", desc: "AI management system (AIMS) alignment." },
    { code: "ISO/IEC 23894", desc: "AI risk management guidance mapping." },
  ];
  return (
    <section className="mx-auto max-w-[1600px] 2xl:max-w-[1760px] px-8 sm:px-12 xl:px-16 py-24" aria-label="AI frameworks alignment">
      <div className="grid lg:grid-cols-2 gap-20 xl:gap-28 items-start">
        <div>
          <h3 className="text-3xl font-semibold">Aligned to AI‑first frameworks</h3>
          <p className="mt-4 text-base text-muted-foreground leading-relaxed">
            Mapped controls and evidence to AI‑first frameworks: NIST AI RMF, EU AI Act, ISO/IEC 42001, and ISO/IEC 23894—plus bridges to SOC 2 and GDPR.
          </p>
          <ul className="mt-6 space-y-2 text-sm">
            <li className="text-muted-foreground">Evidence exports, control mappings, and audit trails.</li>
            <li className="text-muted-foreground">Separation of duties, approvals, and retention policies.</li>
            <li className="text-muted-foreground">Works cloud or self‑hosted for regulated environments.</li>
          </ul>
        </div>
        <div className="grid sm:grid-cols-2 gap-4">
          {items.map((it) => (
            <div key={it.code} className="rounded-lg border bg-card p-6 shadow-sm">
              <div className="text-base font-medium">{it.code}</div>
              <div className="text-sm text-muted-foreground mt-2 leading-relaxed">{it.desc}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

