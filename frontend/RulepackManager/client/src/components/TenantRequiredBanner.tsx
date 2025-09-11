import { useEffect, useState } from 'react';
import { AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useTenant } from '@/contexts/TenantContext';
import { useLocation } from 'wouter';

export function TenantRequiredBanner() {
  const [visible, setVisible] = useState(false);
  const { tenantId } = useTenant();
  const [, setLocation] = useLocation();

  useEffect(() => {
    const handler = () => {
      setVisible(true);
      // Navigate to root so Router can render TenantSelector overlay
      setLocation('/');
    };
    window.addEventListener('ps:tenant-required', handler);
    return () => window.removeEventListener('ps:tenant-required', handler);
  }, [setLocation]);

  useEffect(() => {
    // Auto-hide when a tenant is selected
    if (tenantId) setVisible(false);
  }, [tenantId]);

  if (!visible) return null;

  return (
    <div className="w-full bg-amber-50 border-b border-amber-200 text-amber-900">
      <div className="max-w-screen-xl mx-auto px-4 py-2 flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 text-sm">
          <AlertTriangle className="h-4 w-4" />
          <span>Select an organization to continue.</span>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => setLocation('/')}>Choose tenant</Button>
          <Button size="sm" variant="ghost" onClick={() => setVisible(false)}>Dismiss</Button>
        </div>
      </div>
    </div>
  );
}

