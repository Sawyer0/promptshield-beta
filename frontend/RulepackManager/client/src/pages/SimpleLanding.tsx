import { Shield, Users, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useEffect, useState } from 'react';
import { SignInModal } from '@/components/auth/SignInModal';
import { SignUpModal } from '@/components/auth/SignUpModal';
import { useToast } from '@/hooks/use-toast';
import { useTenant } from '@/contexts/TenantContext';
import { useLocation } from 'wouter';

export default function SimpleLanding() {
  const { toast } = useToast();
  const { setTenant } = useTenant();
  const [, setLocation] = useLocation();
  const [signInOpen, setSignInOpen] = useState(false);
  const [signUpOpen, setSignUpOpen] = useState(false);

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900 flex items-center justify-center p-4">
      <div className="w-full max-w-4xl">
        <div className="text-center mb-12">
          <div className="flex justify-center items-center mb-6">
            <div className="bg-blue-600 p-3 rounded-full">
              <Shield className="h-8 w-8 text-white" />
            </div>
          </div>
          
          <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
            Welcome to PromptShield
          </h1>
          
          <p className="text-xl text-gray-600 dark:text-gray-300 mb-8 max-w-2xl mx-auto">
            Advanced AI Security Management Platform. Protect your AI systems with comprehensive 
            security rulepacks, real-time monitoring, and intelligent threat detection.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm">
              <Shield className="h-8 w-8 text-blue-600 mx-auto mb-4" />
              <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Security Rules</h3>
              <p className="text-sm text-gray-600 dark:text-gray-300">
                Create and manage comprehensive security policies for your AI systems.
              </p>
            </div>
            
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm">
              <Users className="h-8 w-8 text-blue-600 mx-auto mb-4" />
              <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Multi-Tenant</h3>
              <p className="text-sm text-gray-600 dark:text-gray-300">
                Manage multiple organizations with isolated security contexts.
              </p>
            </div>
            
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm">
              <Zap className="h-8 w-8 text-blue-600 mx-auto mb-4" />
              <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Real-time Monitoring</h3>
              <p className="text-sm text-gray-600 dark:text-gray-300">
                Monitor threats and policy violations with advanced analytics.
              </p>
            </div>
          </div>

          <div className="space-y-4 max-w-sm mx-auto">
            <Button size="lg" className="w-full" data-testid="button-signin" onClick={() => setSignInOpen(true)}>
              Sign In
            </Button>
            
            <Button 
              size="lg" 
              variant="outline" 
              className="w-full" 
              onClick={() => setSignUpOpen(true)}
              data-testid="button-signup"
            >
              Create Account
            </Button>
          </div>
        </div>

        <SignInModal
          open={signInOpen}
          onOpenChange={setSignInOpen}
          onSuccess={() => {
            // After successful sign-in, try to hydrate session role and go home
            fetch('/api/auth/user', { credentials: 'include' }).catch(() => {});
            setLocation('/');
          }}
        />
        <SignUpModal
          open={signUpOpen}
          onOpenChange={setSignUpOpen}
          onSuccess={async (orgName?: string) => {
            if (orgName && orgName.trim()) {
              try {
                const resp = await fetch('/api/orgs/create', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  credentials: 'include',
                  body: JSON.stringify({ name: orgName.trim() }),
                });
                if (resp.ok) {
                  const created = await resp.json();
                  const orgId = created?.id || created?.tenant_id;
                  const orgNameResp = created?.name || orgName.trim();
                  // Persist selection on server and locally
                  if (orgId) {
                    await fetch('/api/orgs/select', {
                      method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
                      body: JSON.stringify({ orgId })
                    }).catch(() => {});
                    setTenant(orgId, orgNameResp);
                  }
                  toast({ title: 'Organization created', description: 'You are now admin of this organization.' });
                  setLocation('/');
                } else if (resp.status === 403) {
                  toast({ title: 'Signup disabled', description: 'Self-serve tenant signup is disabled. Contact your administrator.', variant: 'destructive' });
                }
              } catch (_) {}
            }
            // We triggered routing manually above on success
          }}
        />
      </div>
    </div>
  );
}
