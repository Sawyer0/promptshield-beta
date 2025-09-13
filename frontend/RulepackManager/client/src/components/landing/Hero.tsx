import { ArrowRight, Lock, Shield, Zap, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TrustRow } from "./TrustRow";

export function Hero({ onCta, onGoToApp, onSignUp }: { onCta: () => void; onGoToApp: () => void; onSignUp: () => void }) {
  return (
    <section className="marketing-hero">
      <div className="marketing-container marketing-section pt-20 md:pt-24">
        <div className="grid lg:grid-cols-2 gap-16 lg:gap-24 xl:gap-32 items-center">
          <div>
            <div className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs text-muted-foreground">
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: "var(--brand-accent)" }} />
              Secure and compliant AI, faster
            </div>
            <h1 className="mt-6 marketing-h1 font-medium serif-display text-foreground">
              Ship AI agents with enterprise guardrails
            </h1>
            <p className="mt-6 marketing-body text-foreground/80">
              Policy, approvals, and observability aligned with NIST AI RMF and EU AI Act—deployed fast with white‑glove help.
            </p>
            <ul className="mt-5 space-y-1.5 text-base">
              <li className="flex items-start gap-2"><Check className="h-4 w-4 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Policy and approvals for high‑risk actions</li>
              <li className="flex items-start gap-2"><Check className="h-4 w-4 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Live observability, violations, and exportable audits</li>
              <li className="flex items-start gap-2"><Check className="h-4 w-4 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Tool scopes and least‑privilege enforcement</li>
              <li className="flex items-start gap-2"><Check className="h-4 w-4 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Mappings to NIST AI RMF, EU AI Act, ISO/IEC 42001</li>
              <li className="flex items-start gap-2"><Check className="h-4 w-4 mt-0.5" style={{ color: "var(--brand-accent)" }} /> Cloud or self‑hosted deployment</li>
            </ul>
            <div className="mt-6 flex flex-col sm:flex-row gap-3">
              <Button onClick={onCta} className="gap-2 w-full sm:w-auto" style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }}>
                Join the waitlist <ArrowRight className="h-4 w-4" />
              </Button>
              <Button
                variant="secondary"
                className="gap-2 w-full sm:w-auto"
                onClick={onGoToApp}
              >
                Go to app <ArrowRight className="h-4 w-4" />
              </Button>
            </div>
            <div className="mt-6 text-sm text-muted-foreground">
              New here?{' '}
              <Button variant="link" className="px-0 h-auto text-sm" onClick={onSignUp}>
                Create account
              </Button>
            </div>
            <div className="mt-8">
              <TrustRow />
            </div>
          </div>
          <div className="mt-8 lg:mt-0">
            <div className="relative rounded-xl p-6 bg-card border shadow-sm">
              <div className="grid grid-cols-3 gap-3 sm:gap-4 text-sm">
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border" style={{ borderColor: "var(--brand-accent)", color: "var(--brand-accent)" }}>
                  <Shield className="h-4 w-4" />
                  <span>Guardrails</span>
                </div>
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border" style={{ borderColor: "var(--brand-accent)", color: "var(--brand-accent)" }}>
                  <Lock className="h-4 w-4" />
                  <span>Evidence</span>
                </div>
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border" style={{ borderColor: "var(--brand-accent)", color: "var(--brand-accent)" }}>
                  <Zap className="h-4 w-4" />
                  <span>Runtime</span>
                </div>
              </div>
              <div className="mt-8">
                <div className="h-48 rounded-md bg-gradient-to-br from-muted to-background border flex items-center justify-center text-muted-foreground">
                  <span>Security guardrails diagram (placeholder)</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

