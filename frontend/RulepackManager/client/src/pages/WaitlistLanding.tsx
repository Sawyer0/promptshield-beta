import { useState } from "react";
import { useLocation } from "wouter";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Hero, SolutionSection, FeatureGrid, PromptRiskDemo, FAQ, WaitlistForm } from "@/components/landing";
import { isDevBypassClient } from "@/lib/dev";

export default function WaitlistLanding() {
  const [openWaitlist, setOpenWaitlist] = useState(false);
  const [, setLocation] = useLocation();

  const onGoToApp = () => {
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

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-white to-sky-50 dark:from-gray-950 dark:via-gray-925 dark:to-gray-950">
      <Hero
        onTryDemo={() => {
          const el = document.getElementById('prompt-demo');
          if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }}
        onCta={() => setOpenWaitlist(true)}
        onGoToApp={onGoToApp}
        onSignUp={() => {
          const isDev = isDevBypassClient();
          if (isDev) {
            setLocation('/onboarding/role');
          } else {
            // Route to the auth page; the page itself will open the modal
            setLocation('/auth?mode=signup');
          }
        }}
      />

      <SolutionSection />
      <FeatureGrid />

      <div id="prompt-demo">
        <PromptRiskDemo />
      </div>

      <FAQ />

      <Dialog open={openWaitlist} onOpenChange={setOpenWaitlist}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Join the waitlist</DialogTitle>
          </DialogHeader>
          <WaitlistForm inline={true} />
        </DialogContent>
      </Dialog>


      <footer className="border-t mt-8">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 py-8 text-sm text-muted-foreground">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
            <div>© {new Date().getFullYear()} Promptshield</div>
            <div className="flex items-center gap-4">
              <a href="#" className="hover:underline">Security</a>
              <a href="#" className="hover:underline">Compliance</a>
              <a href="/privacy" className="hover:underline">Privacy</a>
              <a href="#" className="hover:underline">Terms</a>
              <a href="#" className="hover:underline">Contact</a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}

