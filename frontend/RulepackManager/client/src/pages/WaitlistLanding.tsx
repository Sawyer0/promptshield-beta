import { useState } from "react";
import { useLocation } from "wouter";
import { useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { MarketingHeader, MarketingFooter, Hero, UserStories, FAQ, WaitlistForm } from "@/components/landing";
import { isDevBypassClient } from "@/lib/dev";
import { useTrack } from "@/hooks/useTrack";

export default function WaitlistLanding() {
  const [openWaitlist, setOpenWaitlist] = useState(false);
  const [, setLocation] = useLocation();

  const track = useTrack();

  const onGoToApp = () => {
    track('MarketingCtaClick', { page: 'home', cta: 'go_to_app' });
    try {
      const isDev = isDevBypassClient();
      if (!isDev) {
        // Route to the auth page; the page itself will open the modal
        setLocation('/auth?mode=signin');
        return;
      }
      const hasRoles = !!localStorage.getItem('ps_roles');
      const hasTenant = !!localStorage.getItem('promptshield_tenant_id');
      const role = localStorage.getItem('user_system_role');
      if (!hasRoles) { setLocation('/onboarding/role'); return; }
      if (!hasTenant) { setLocation('/choose-tenant'); return; }
      if (role === 'admin') { setLocation('/platform'); } else { setLocation('/dashboard'); }
    } catch {
      setLocation('/auth?mode=signin');
    }
  };

  useEffect(() => {
    try {
      const p = new URLSearchParams(window.location.search);
      if (p.get('open') === 'waitlist' || p.get('waitlist') === 'true') {
        setOpenWaitlist(true);
      }
    } catch {}
    try { track('MarketingPageView', { page: 'home' }); } catch {}
  }, []);

  return (
    <div className="marketing-root min-h-screen bg-gradient-to-br from-emerald-100 via-emerald-50 to-emerald-75 dark:from-emerald-950 dark:via-emerald-900 dark:to-emerald-950">
      <MarketingHeader />
      <Hero
        onCta={() => { track('MarketingCtaClick', { page: 'home', cta: 'join_waitlist' }); setOpenWaitlist(true); }}
        onGoToApp={onGoToApp}
        onSignUp={() => {
          track('MarketingCtaClick', { page: 'home', cta: 'create_account' });
          const isDev = isDevBypassClient();
          if (isDev) {
            setLocation('/onboarding/role');
          } else {
            // Route to the auth page; the page itself will open the modal
            setLocation('/auth?mode=signup');
          }
        }}
      />
      <UserStories />

      <FAQ />

      <Dialog open={openWaitlist} onOpenChange={setOpenWaitlist}>
        <DialogContent className="marketing-root sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Join the waitlist</DialogTitle>
          </DialogHeader>
          <WaitlistForm inline={true} />
        </DialogContent>
      </Dialog>


      <MarketingFooter />
    </div>
  );
}

