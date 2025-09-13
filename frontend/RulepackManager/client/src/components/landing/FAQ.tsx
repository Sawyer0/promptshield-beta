export function FAQ() {
  const items = [
    { q: "Will this slow my app?", a: "Lightweight middleware with async evaluation and caching—designed for low overhead." },
    { q: "Are we locked into a model/vendor?", a: "No. PoliSync Guard is model- and vendor-agnostic." },
    { q: "Can we deploy in a VPC or on‑prem?", a: "Yes—deployment options for regulated industries." },
    { q: "What compliance evidence is generated?", a: "Policy diffs, risk events, control mappings, and exportable audit reports." },
    { q: "How long does integration take?", a: "Many teams see value in under a day." },
  ];
  return (
    <section className="marketing-container marketing-section">
      <h3 className="marketing-h3 font-medium serif-display mb-10 text-foreground">FAQ</h3>
      <div className="space-y-6">
        {items.map((it) => (
          <div key={it.q} className="bg-marketing-card border rounded-xl p-6 md:p-7 shadow-sm">
            <div className="marketing-h3 font-medium text-foreground">{it.q}</div>
            <div className="marketing-body text-foreground/80 mt-3 leading-relaxed">{it.a}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

