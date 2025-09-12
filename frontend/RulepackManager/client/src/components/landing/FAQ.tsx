export function FAQ() {
  const items = [
    { q: "Will this slow my app?", a: "Lightweight middleware with async evaluation and caching—designed for low overhead." },
    { q: "Are we locked into a model/vendor?", a: "No. Promptshield is model- and vendor-agnostic." },
    { q: "Can we deploy in a VPC or on‑prem?", a: "Yes—deployment options for regulated industries." },
    { q: "What compliance evidence is generated?", a: "Policy diffs, risk events, control mappings, and exportable audit reports." },
    { q: "How long does integration take?", a: "Many teams see value in under a day." },
  ];
  return (
    <section className="mx-auto max-w-3xl px-4 sm:px-6 py-12">
      <h3 className="text-xl font-semibold mb-4">FAQ</h3>
      <div className="space-y-4">
        {items.map((it) => (
          <div key={it.q} className="border rounded-md p-4">
            <div className="font-medium">{it.q}</div>
            <div className="text-sm text-muted-foreground mt-1">{it.a}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

