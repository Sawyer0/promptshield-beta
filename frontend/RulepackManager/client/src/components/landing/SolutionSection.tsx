import { CheckCircle2 } from "lucide-react";

export function SolutionSection() {
  return (
    <section className="mx-auto max-w-6xl px-4 sm:px-6 py-12">
      <div className="grid lg:grid-cols-2 gap-10 items-start">
        <div>
          <h2 className="text-2xl font-medium serif-display">How Promptshield works</h2>
          <p className="mt-3 text-muted-foreground">
            Reduce security risk, speed approvals, and simplify audits—without slowing developers.
          </p>
          <ul className="mt-6 space-y-3 text-sm">
            <li className="flex items-start gap-2"><CheckCircle2 className="h-5 w-5 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Runtime detection of injection, jailbreak, and exfiltration attempts.</li>
            <li className="flex items-start gap-2"><CheckCircle2 className="h-5 w-5 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Guardrails as code: allow/deny, redaction, sandboxing, rate limits, tool constraints.</li>
            <li className="flex items-start gap-2"><CheckCircle2 className="h-5 w-5 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Compliance evidence mapped to OWASP Top 10 for LLM, SOC 2, HIPAA, GDPR, NIST AI RMF.</li>
          </ul>
        </div>
        <div className="rounded-xl p-6 bg-card border shadow-sm">
          <div className="h-56 rounded-md bg-gradient-to-br from-muted to-background border flex items-center justify-center text-muted-foreground">
            <span>Risk analytics preview (placeholder)</span>
          </div>
        </div>
      </div>
    </section>
  );
}

