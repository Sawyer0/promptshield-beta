import { MarketingHeader, MarketingFooter, FeatureGrid, AIFrameworks } from "@/components/landing";
import { Button } from "@/components/ui/button";
import { Link } from "wouter";
import { useEffect } from "react";
import { useTrack } from "@/hooks/useTrack";

export default function FeaturesOverview() {
  const track = useTrack();
  useEffect(() => { try { track('MarketingPageView', { page: 'features' }); } catch {} }, []);
  return (
    <div className="marketing-root min-h-screen bg-background">
      <MarketingHeader />
      <section className="marketing-hero">
        <div className="marketing-container marketing-section">
          <h1 className="marketing-h1 font-medium serif-display">Capabilities</h1>
          <p className="mt-4 marketing-body text-foreground/80 max-w-3xl">Guardrails, observability, and governance—built for shipping AI safely and meeting auditors where they are.</p>
          <div className="mt-6">
            <Link href="/waitlist?open=waitlist"><Button onClick={() => { try { track('MarketingCtaClick', { page: 'features', cta: 'join_waitlist' }); } catch {} }} className="gap-2" style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }}>Join the waitlist</Button></Link>
          </div>
        </div>
      </section>
      <FeatureGrid />
      <AIFrameworks />
      <MarketingFooter />
    </div>
  );
}
