import { MarketingHeader, MarketingFooter } from "@/components/landing";
import { useEffect } from "react";
import { useTrack } from "@/hooks/useTrack";

export default function Trust() {
  const track = useTrack();
  useEffect(() => { try { track('MarketingPageView', { page: 'trust' }); } catch {} }, []);
  return (
    <div className="marketing-root min-h-screen bg-background">
      <MarketingHeader />
      <section className="marketing-hero">
        <div className="marketing-container marketing-section">
          <h1 className="marketing-h1 font-medium serif-display">Trust & Security</h1>
          <p className="mt-4 marketing-body text-foreground/80 max-w-3xl">We prioritize data protection, isolation, and transparency. Ask us for our detailed security overview and sub‑processor list.</p>
        </div>
      </section>
      <section className="marketing-container marketing-section">
        <div className="grid md:grid-cols-3 gap-6">
          <div className="rounded-lg border bg-card p-6 shadow-sm">
            <div className="font-medium">Data protection</div>
            <div className="text-sm text-muted-foreground mt-2 leading-relaxed">Encryption at rest and in transit. Options for VPC or self‑hosted deployments.</div>
          </div>
          <div className="rounded-lg border bg-card p-6 shadow-sm">
            <div className="font-medium">Access controls</div>
            <div className="text-sm text-muted-foreground mt-2 leading-relaxed">SSO, granular roles, approvals, and least‑privilege tool access.</div>
          </div>
          <div className="rounded-lg border bg-card p-6 shadow-sm">
            <div className="font-medium">Observability</div>
            <div className="text-sm text-muted-foreground mt-2 leading-relaxed">Audit trails, evidence exports, and real‑time event streams.</div>
          </div>
        </div>
      </section>
      <MarketingFooter />
    </div>
  );
}
