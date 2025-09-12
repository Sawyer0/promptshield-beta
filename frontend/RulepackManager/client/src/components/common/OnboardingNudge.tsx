import { useEffect, useState } from 'react';
import { useLocation } from 'wouter';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';

type Nudge = { forPath: string; variant: 'A'|'B'; dismissed?: boolean; context?: any };

export function OnboardingNudge() {
  const [location] = useLocation();
  const [nudge, setNudge] = useState<Nudge | null>(null);

  useEffect(() => {
    try {
      const raw = localStorage.getItem('ps_onboarding_nudge');
      if (!raw) return;
      const obj: Nudge = JSON.parse(raw);
      if (!obj || obj.dismissed) return;
      if (obj.forPath && obj.forPath === location) setNudge(obj);
    } catch {}
  }, [location]);

  if (!nudge) return null;

  const { variant, context } = nudge;
  const role = context?.role;
  const useCase = context?.useCase;

  const headline = (() => {
    if (variant === 'A') {
      if (useCase === 'policy_enforcement') return 'Tip: Define your first Tool Policy to lock down tools.';
      if (useCase === 'monitoring') return 'Tip: Watch live events and set alerts from here.';
      if (useCase === 'compliance_reporting') return 'Tip: Export evidence mapped to your framework.';
      if (useCase === 'billing_analytics') return 'Tip: Export usage CSV for cost allocation.';
      return 'Tip: Get started with your workspace.';
    }
    if (useCase === 'policy_enforcement') return 'Pro move: Add allowed tools and required approvals now.';
    if (useCase === 'monitoring') return 'Pro move: Check request rate and redactions trends.';
    if (useCase === 'compliance_reporting') return 'Pro move: Filter controls by framework & export evidence.';
    if (useCase === 'billing_analytics') return 'Pro move: Segment usage by endpoint and export CSV.';
    return 'Pro move: Explore key actions to get value fast.';
  })();

  return (
    <div className="mb-4">
      <Alert className="flex items-center justify-between">
        <div className="text-sm">
          <span className="font-medium mr-2">{headline}</span>
          <span className="text-muted-foreground">You can dismiss this anytime.</span>
        </div>
        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            try {
              const raw = localStorage.getItem('ps_onboarding_nudge');
              const obj = raw ? JSON.parse(raw) : null;
              if (obj) {
                obj.dismissed = true;
                localStorage.setItem('ps_onboarding_nudge', JSON.stringify(obj));
              }
            } catch {}
            setNudge(null);
          }}
        >Dismiss</Button>
      </Alert>
    </div>
  );
}

