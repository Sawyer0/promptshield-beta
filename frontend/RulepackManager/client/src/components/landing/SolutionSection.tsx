import { CheckCircle2 } from "lucide-react";

export function SolutionSection() {
  return (
    <section className="mx-auto max-w-[1600px] 2xl:max-w-[1760px] px-8 sm:px-12 xl:px-16 py-24">
      <div className="grid lg:grid-cols-2 gap-20 xl:gap-28 items-start">
        <div>
          <h2 className="text-4xl sm:text-5xl font-medium serif-display leading-tight">How PoliSync Guard works</h2>
          <p className="mt-6 text-xl text-muted-foreground leading-relaxed">
            Reduce security risk, speed approvals, and simplify audits—without slowing developers.
          </p>
          <ul className="mt-8 space-y-5 text-base">
            <li className="flex items-start gap-2"><CheckCircle2 className="h-5 w-5 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Runtime detection of injection, jailbreak, and exfiltration attempts.</li>
            <li className="flex items-start gap-2"><CheckCircle2 className="h-5 w-5 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Guardrails as code: allow/deny, redaction, sandboxing, rate limits, tool constraints.</li>
            <li className="flex items-start gap-2"><CheckCircle2 className="h-5 w-5 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Compliance evidence mapped to OWASP Top 10 for LLM, SOC 2, HIPAA, GDPR, NIST AI RMF.</li>
          </ul>
        </div>
        <div className="mt-8 lg:mt-0">
          <div className="rounded-xl p-10 bg-card border shadow-sm">
            <div className="h-72 rounded-md bg-gradient-to-br from-muted to-background border flex items-center justify-center text-muted-foreground">
              <span>Risk analytics preview (placeholder)</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

