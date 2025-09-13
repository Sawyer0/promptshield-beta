import { MarketingHeader, MarketingFooter, AIFrameworks } from "@/components/landing";
import { Button } from "@/components/ui/button";
import { Link } from "wouter";
import { useEffect } from "react";
import { useTrack } from "@/hooks/useTrack";

export default function SolutionsCompliance() {
  const track = useTrack();
  useEffect(() => { try { track('MarketingPageView', { page: 'solutions_compliance' }); } catch {} }, []);
  return (
    <div className="marketing-root min-h-screen bg-background">
      <MarketingHeader />
      <section className="marketing-hero">
        <div className="marketing-container marketing-section">
          <h1 className="marketing-h1 font-medium serif-display">Compliance for AI systems</h1>
          <p className="mt-4 marketing-body text-foreground/80 max-w-3xl">Evidence-grade logging, mapped controls, and exportable reports aligned to NIST AI RMF, EU AI Act, ISO/IEC 42001, and ISO/IEC 23894.</p>
          <ul className="mt-6 space-y-2 text-base">
            <li>Control mappings and evidence exports</li>
            <li>Retention, separation of duties, approvals</li>
            <li>Bridges to SOC 2, GDPR, HIPAA</li>
          </ul>
          <div className="mt-6">
            <Link href="/waitlist?open=waitlist"><Button onClick={() => { try { track('MarketingCtaClick', { page: 'solutions_compliance', cta: 'join_waitlist' }); } catch {} }} className="gap-2" style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }}>Join the waitlist</Button></Link>
          </div>
        </div>
      </section>
      <AIFrameworks />
      <MarketingFooter />
    </div>
  );
}
