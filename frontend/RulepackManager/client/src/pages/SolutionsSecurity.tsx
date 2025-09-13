import { MarketingHeader, MarketingFooter } from "@/components/landing";
import { Button } from "@/components/ui/button";
import { Link } from "wouter";
import { useEffect } from "react";
import { useTrack } from "@/hooks/useTrack";

export default function SolutionsSecurity() {
  const track = useTrack();
  useEffect(() => { try { track('MarketingPageView', { page: 'solutions_security' }); } catch {} }, []);
  return (
    <div className="marketing-root min-h-screen bg-background">
      <MarketingHeader />
      <section className="marketing-hero">
        <div className="marketing-container marketing-section">
          <h1 className="marketing-h1 font-medium serif-display">Security for AI agents</h1>
          <p className="mt-4 marketing-body text-foreground/80 max-w-3xl">Enforce tool scopes, prevent injection and exfiltration, and achieve end‑to‑end auditability—without slowing developers.</p>
          <ul className="mt-6 space-y-2 text-base">
            <li>Runtime guardrails: allow/deny, redaction, sandboxing</li>
            <li>Least‑privilege tool scopes and secrets boundaries</li>
            <li>Real‑time monitoring, violations, and alerts</li>
          </ul>
          <div className="mt-6">
            <Link href="/waitlist?open=waitlist"><Button onClick={() => { try { track('MarketingCtaClick', { page: 'solutions_security', cta: 'join_waitlist' }); } catch {} }} className="gap-2" style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }}>Join the waitlist</Button></Link>
          </div>
        </div>
      </section>
      <MarketingFooter />
    </div>
  );
}
