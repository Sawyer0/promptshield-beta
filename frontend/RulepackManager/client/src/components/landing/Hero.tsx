import { ArrowRight, Lock, Shield, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TrustRow } from "./TrustRow";

export function Hero({ onTryDemo, onCta, onGoToApp, onSignUp }: { onTryDemo: () => void; onCta: () => void; onGoToApp: () => void; onSignUp: () => void }) {
  return (
    <section className="relative overflow-hidden">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 py-14">
        <div className="grid lg:grid-cols-2 gap-10 items-center">
          <div>
            <div className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs text-muted-foreground">
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: "var(--brand-accent)" }} />
              Secure and compliant AI, faster
            </div>
            <h1 className="mt-4 text-4xl sm:text-5xl font-medium tracking-tight serif-display">
              Ship secure, compliant LLM apps—faster
            </h1>
            <p className="mt-4 text-lg text-muted-foreground">
              Protect your LLM apps from prompt injection, data exfiltration, and tool abuse—while automating evidence for SOC 2, HIPAA, and GDPR.
            </p>
            <div className="mt-6 flex flex-col sm:flex-row gap-3">
              <Button size="lg" onClick={onCta} className="gap-2" style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }}>
                Join the waitlist <ArrowRight className="h-4 w-4" />
              </Button>
              <Button size="lg" variant="outline" onClick={onTryDemo} className="gap-2">
                Try the prompt risk demo
              </Button>
              <Button
                size="lg"
                variant="secondary"
                className="gap-2"
                onClick={onGoToApp}
              >
                Go to app <ArrowRight className="h-4 w-4" />
              </Button>
            </div>
            <div className="mt-2 text-sm text-muted-foreground">
              New here?{' '}
              <Button variant="link" className="px-0 h-auto text-sm" onClick={onSignUp}>
                Create account
              </Button>
            </div>
            <TrustRow />
          </div>
          <div>
            <div className="relative rounded-xl p-6 bg-card border shadow-sm">
              <div className="grid grid-cols-3 gap-4 text-sm">
                <div className="flex items-center gap-2"><Shield className="h-5 w-5 text-green-600" /><span>Guardrails</span></div>
                <div className="flex items-center gap-2"><Lock className="h-5 w-5 text-green-600" /><span>Evidence</span></div>
                <div className="flex items-center gap-2"><Zap className="h-5 w-5 text-green-600" /><span>Runtime</span></div>
              </div>
              <div className="mt-6">
                <div className="h-40 rounded-md bg-gradient-to-br from-muted to-background border flex items-center justify-center text-muted-foreground">
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

