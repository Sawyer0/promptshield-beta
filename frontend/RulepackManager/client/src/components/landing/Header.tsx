import { Link, useLocation } from "wouter";
import { useTrack } from "@/hooks/useTrack";
import { Button } from "@/components/ui/button";

function useActivePath() {
  const [path] = useLocation();
  return path;
}

export function MarketingHeader() {
  const track = useTrack();
  const active = useActivePath();
  const nav = [
    { href: "/features", label: "Features" },
    { href: "/solutions/compliance", label: "Compliance" },
    { href: "/solutions/security", label: "Security" },
    { href: "/trust", label: "Trust" },
    { href: "/privacy", label: "Privacy" },
  ];
  return (
    <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto max-w-[1600px] 2xl:max-w-[1760px] px-8 sm:px-12 xl:px-16 h-16 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link href="/">
            <a className="font-semibold text-lg tracking-tight">PoliSync Guard</a>
          </Link>
          <nav className="hidden md:flex items-center gap-5 text-sm">
            {nav.map((item) => {
              const isActive = active === item.href;
              return (
                <Link key={item.href} href={item.href}>
                  <a onClick={() => { try { track('MarketingNavClick', { to: item.href }); } catch {} }} className={`hover:text-foreground/80 transition-colors ${isActive ? 'text-foreground' : 'text-muted-foreground'}`}>{item.label}</a>
                </Link>
              );
            })}
          </nav>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/waitlist?open=waitlist">
            <Button size="sm" className="gap-2" onClick={() => { try { track('MarketingCtaClick', { page: 'header', cta: 'join_waitlist' }); } catch {} }} style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }}>Join the waitlist</Button>
          </Link>
        </div>
      </div>
    </header>
  );
}
