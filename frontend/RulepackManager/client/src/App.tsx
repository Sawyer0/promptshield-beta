import { Switch, Route, Redirect, useLocation } from "wouter";
import { useEffect } from "react";
import { useSafeClerk, useSafeClerkAuth } from '@/clerk/shim';
import { queryClient } from "./lib/queryClient";
import { QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ThemeProvider } from "@/components/ThemeProvider";
import { useAuth } from "@/hooks/useAuth";
import { TenantProvider, useTenant } from "@/contexts/TenantContext";
import { TenantSelector } from "@/components/TenantSelector";
import { ClerkProvider } from '@clerk/clerk-react';
import { isDevBypassClient } from "@/lib/dev";


// Pages
import WaitlistLanding from "@/pages/WaitlistLanding";
import SignIn from "@/pages/SignIn";
import SignUp from "@/pages/SignUp";
import Dashboard from "@/pages/Dashboard";
import RulePacks from "@/pages/RulePacks";
import Tenants from "@/pages/Tenants";
import PolicyAssignments from "@/pages/PolicyAssignments";
import Violations from "@/pages/Violations";
import Services from "@/pages/Services";
import AuditLogs from "@/pages/AuditLogs";
import SystemHealth from "@/pages/SystemHealth";
import Users from "@/pages/Users";
import PlatformDashboard from "@/pages/PlatformDashboard";
import Preferences from "@/pages/Preferences";
import Tools from "@/pages/Tools";
import Organization from "@/pages/Organization";
import ToolPolicies from "@/pages/ToolPolicies";
import Compliance from "@/pages/Compliance";
import RoleSetup from "@/pages/RoleSetup";
import NotFound from "@/pages/not-found";
import AuthModalPage from "@/pages/AuthModal";
import { AdminRoleDebug } from "@/components/AdminRoleDebug";
import EnforcerMonitoring from "@/pages/EnforcerMonitoring";
import Snapshots from "@/pages/Snapshots";
import UsageAnalytics from "@/pages/UsageAnalytics";
import ViolationsSummary from "@/pages/ViolationsSummary";
import Presets from "@/pages/Presets";
import Billing from "@/pages/Billing";
import Invoices from "@/pages/Invoices";
import Experiments from "@/pages/Experiments";
import AgentManagement from "@/pages/AgentManagement";
import AdminSystem from "@/pages/AdminSystem";
import License from "@/pages/License";
import DebugDiagnostics from "@/pages/DebugDiagnostics";
import LiveEvents from "@/pages/LiveEvents";
import AccountSecurity from "@/pages/AccountSecurity";
import { TenantRequiredBanner } from "@/components/TenantRequiredBanner";
import { Privacy } from "@/pages/Privacy";
import FeaturesOverview from "@/pages/FeaturesOverview";
import SolutionsCompliance from "@/pages/SolutionsCompliance";
import SolutionsSecurity from "@/pages/SolutionsSecurity";
import Trust from "@/pages/Trust";

function useRoles(): string[] {
  try {
    const raw = localStorage.getItem('ps_roles');
    if (raw) return JSON.parse(raw);
  } catch {}
  // Derive basic role from existing storage as fallback
  const basic = localStorage.getItem('user_system_role') === 'admin' ? ['platform_admin'] : ['developer'];
  return basic;
}

function RequireRoles({ roles, children }: { roles: string[]; children: any }) {
  const my = useRoles();
  const allow = roles.some(r => my.includes(r));
  return allow ? children : <NotFound />;
}

function Router() {
  const { isAuthenticated, isLoading } = useAuth();
  const { tenantId, isLoading: isTenantLoading } = useTenant();
  const [loc, setLocation] = useLocation();
  const clerk = useSafeClerk();
  const clerkAuth = useSafeClerkAuth();

  // If Clerk reports a lastActiveSessionId but no active session (common after refresh),
  // promote it to active so isSignedIn flips true without manual re-login.
  useEffect(() => {
    const autoActivate = (import.meta.env.VITE_DISABLE_AUTO_SESSION_ACTIVATION !== 'true');
    if (!autoActivate) return;
    let timer: any;
    const tryActivate = async () => {
      try {
        const anyClerk: any = clerk as any;
        const loaded = !!anyClerk?.loaded;
        const client = anyClerk?.client;
        const hasActive = !!anyClerk?.session;
        const last = client?.lastActiveSessionId;
        if (loaded && !hasActive && last) {
          await anyClerk.setActive({ session: last });
          return true;
        }
      } catch { /* ignore */ }
      return false;
    };
    const tick = async () => {
      const done = await tryActivate();
      if (!done) timer = setTimeout(tick, 100);
    };
    tick();
    return () => clearTimeout(timer);
  }, [clerk]);

  // After authentication (including OAuth), push users away from auth pages
  useEffect(() => {
    const autoActivate = (import.meta.env.VITE_DISABLE_AUTO_SESSION_ACTIVATION !== 'true');
    if (isAuthenticated && (loc === '/sign-in' || loc === '/sign-up')) {
      // If roles not set yet, go to onboarding role selection
      try {
        const raw = localStorage.getItem('ps_roles');
        if (!raw) {
          setLocation('/onboarding/role');
          return;
        }
      } catch {}
      setLocation('/');
      return;
    }
    // If Clerk reports a pending session, optionally try to activate and route home
    if (!autoActivate) return;
    try {
      const anyClerk: any = clerk as any;
      const client = anyClerk?.client;
      const hasPending = !!client?.lastActiveSessionId;
      const hasActive = !!anyClerk?.session;
      if (!isAuthenticated && hasPending && !hasActive) {
        // Trigger activation and move off auth pages immediately for better UX
        anyClerk.setActive({ session: client.lastActiveSessionId }).catch(() => {});
        if (loc === '/sign-in' || loc === '/sign-up' || loc === '/landing' || loc === '/') {
          setLocation('/');
        }
      }
    } catch { /* ignore */ }
  }, [isAuthenticated, loc, setLocation, clerk]);

  // On auth, fetch minimal user info to set role hint for routing (admin vs user)
  useEffect(() => {
    if (!isAuthenticated) {
      localStorage.removeItem('user_system_role');
      return;
    }
    (async () => {
      try {
        let headers: Record<string,string> = {};
        try {
          // Prefer Clerk React useAuth().getToken (works with any templates)
          let tok: string | null | undefined = undefined;
          try {
            // Try default template first, then fallback to unnamed
            tok = await clerkAuth.getToken({ template: 'default' });
            if (!tok) tok = await clerkAuth.getToken();
          } catch {}
          // Final fallback to window.Clerk.session.getToken()
          if (!tok) {
            const anyClerk: any = (clerk as any);
            tok = await anyClerk?.session?.getToken?.().catch(() => undefined);
          }
          if (tok) headers['Authorization'] = `Bearer ${tok}`;
        } catch {}
        const r = await fetch('/api/auth/user', { credentials: 'include', headers });
        if (r.ok) {
          const data = await r.json();
          if (data?.systemRole) {
            localStorage.setItem('user_system_role', data.systemRole);
          } else {
            localStorage.setItem('user_system_role', 'user');
          }
          if (Array.isArray(data?.roles)) {
            localStorage.setItem('ps_roles', JSON.stringify(data.roles));
          }
        }
      } catch {}
    })();
  }, [isAuthenticated]);

  // Clean up debug logging

  // Show loading state while checking authentication
  if (isLoading || isTenantLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  // Show tenant selector if authenticated but no tenant selected (except for platform admins)
  const userRole = localStorage.getItem('user_system_role');
  if (isAuthenticated && !tenantId && userRole !== 'admin') {
    return <Redirect to="/auth?mode=signin" />;
  }


  return (
    <>
      <TenantRequiredBanner />
      <AdminRoleDebug />
      <Switch>
        {!isAuthenticated ? (
          <>
            <Route path="/" component={WaitlistLanding} />
            <Route path="/landing" component={WaitlistLanding} />
            <Route path="/waitlist" component={WaitlistLanding} />
            {/* Marketing pages */}
            <Route path="/features" component={FeaturesOverview} />
            <Route path="/solutions/compliance" component={SolutionsCompliance} />
            <Route path="/solutions/security" component={SolutionsSecurity} />
            <Route path="/trust" component={Trust} />
            <Route path="/onboard/role">{() => <Redirect to="/onboarding/role" />}</Route>
            {/* Friendly aliases */}
            <Route path="/signin">{() => <Redirect to="/sign-in" />}</Route>
            <Route path="/signup">{() => <Redirect to="/sign-up" />}</Route>
            {/* Brand-only auth: redirect to modal-styled auth page */}
            <Route path="/sign-in">{() => <Redirect to="/auth?mode=signin" />}</Route>
            <Route path="/sign-up">{() => <Redirect to="/auth?mode=signup" />}</Route>
            <Route path="/orgs">
              {() => (
                <div className="min-h-screen flex items-center justify-center bg-background">
                  <TenantSelector />
                </div>
              )}
            </Route>
            <Route path="/privacy" component={Privacy} />
            <Route path="/auth" component={AuthModalPage} />
          </>
        ) : (
          <>
            {/* Optional dedicated page for choosing a tenant */}
            <Route path="/choose-tenant">
              {() => (
                <div className="min-h-screen flex items-center justify-center bg-background">
                  <TenantSelector />
                </div>
              )}
            </Route>
            {/* Onboarding role selection */}
            <Route path="/onboarding/role" component={RoleSetup} />
            {/* Allow auth modal even when authenticated, to handle org selection/request */}
            <Route path="/auth" component={AuthModalPage} />
            <Route path="/onboard/role">{() => <Redirect to="/onboarding/role" />}</Route>
            {/* Route based on user role */}
            {localStorage.getItem('user_system_role') === 'admin' ? (
              <>
                {/* Platform Owner routes only */}
                <Route path="/landing" component={WaitlistLanding} />
                <Route path="/waitlist" component={WaitlistLanding} />
                {/* Marketing pages (accessible while signed in) */}
                <Route path="/features" component={FeaturesOverview} />
                <Route path="/solutions/compliance" component={SolutionsCompliance} />
                <Route path="/solutions/security" component={SolutionsSecurity} />
                <Route path="/trust" component={Trust} />
                <RequireRoles roles={["platform_admin"]}><Route path="/" component={PlatformDashboard} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/platform" component={PlatformDashboard} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/users" component={Users} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/tenants" component={Tenants} /></RequireRoles>
                <RequireRoles roles={["platform_admin","tenant_admin","security_engineer","auditor"]}><Route path="/compliance" component={Compliance} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/preferences" component={Preferences} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/health" component={SystemHealth} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/admin/snapshots" component={Snapshots} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/admin/system" component={AdminSystem} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/admin/license" component={License} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/admin/debug" component={DebugDiagnostics} /></RequireRoles>
                <RequireRoles roles={["platform_admin"]}><Route path="/account/security" component={AccountSecurity} /></RequireRoles>
                <Route component={NotFound} />
              </>
            ) : (
              <>
                {/* Tenant views with granular RBAC */}
                <Route path="/landing" component={WaitlistLanding} />
                <Route path="/waitlist" component={WaitlistLanding} />
                {/* Marketing pages (accessible while signed in) */}
                <Route path="/features" component={FeaturesOverview} />
                <Route path="/solutions/compliance" component={SolutionsCompliance} />
                <Route path="/solutions/security" component={SolutionsSecurity} />
                <Route path="/trust" component={Trust} />
                <RequireRoles roles={["tenant_admin","security_engineer","developer","auditor"]}><Route path="/" component={Dashboard} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","developer","auditor"]}><Route path="/dashboard" component={Dashboard} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","developer","auditor"]}><Route path="/monitoring/enforcer" component={EnforcerMonitoring} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer"]}><Route path="/rulepacks" component={RulePacks} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer"]}><Route path="/tools" component={Tools} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer"]}><Route path="/tool-policies" component={ToolPolicies} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer"]}><Route path="/presets" component={Presets} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer"]}><Route path="/agent" component={AgentManagement} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer"]}><Route path="/policies" component={PolicyAssignments} /></RequireRoles>
                {/* Back-compat redirects */}
                <Route path="/policy-assignment">{() => <Redirect to="/policies" />}</Route>
                <Route path="/policy-assignments">{() => <Redirect to="/policies" />}</Route>
                <RequireRoles roles={["tenant_admin","security_engineer","developer","auditor"]}><Route path="/violations" component={Violations} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","auditor"]}><Route path="/analytics/violations" component={ViolationsSummary} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","auditor"]}><Route path="/analytics/usage" component={UsageAnalytics} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer"]}><Route path="/services" component={Services} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","developer","auditor"]}><Route path="/monitoring/live" component={LiveEvents} /></RequireRoles>
                <RequireRoles roles={["tenant_admin"]}><Route path="/organization" component={Organization} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","auditor"]}><Route path="/audit" component={AuditLogs} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","auditor","platform_admin"]}><Route path="/compliance" component={Compliance} /></RequireRoles>
                <RequireRoles roles={["tenant_admin"]}><Route path="/preferences" component={Preferences} /></RequireRoles>
                <RequireRoles roles={["tenant_admin"]}><Route path="/billing" component={Billing} /></RequireRoles>
                <RequireRoles roles={["tenant_admin"]}><Route path="/invoices" component={Invoices} /></RequireRoles>
                <RequireRoles roles={["tenant_admin"]}><Route path="/experiments" component={Experiments} /></RequireRoles>
                <RequireRoles roles={["tenant_admin","security_engineer","developer","auditor"]}><Route path="/account/security" component={AccountSecurity} /></RequireRoles>
                <Route component={NotFound} />
              </>
            )}
          </>
        )}
      </Switch>
    </>
  );
}

function App() {
  const devBypass = isDevBypassClient();
  const routerPush = (to: string) => { try { window.history.pushState({}, '', to); window.dispatchEvent(new PopStateEvent('popstate')); } catch {} };
  const routerReplace = (to: string) => { try { window.history.replaceState({}, '', to); window.dispatchEvent(new PopStateEvent('popstate')); } catch {} };
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="light" storageKey="promptshield-ui-theme">
        <TooltipProvider>
          {devBypass ? (
            <TenantProvider>
              <Toaster />
              <Router />
            </TenantProvider>
          ) : (
            <ClerkProvider 
              publishableKey={import.meta.env.VITE_CLERK_PUBLISHABLE_KEY}
              routerPush={routerPush}
              routerReplace={routerReplace}
            >
              <TenantProvider>
                <Toaster />
                <Router />
              </TenantProvider>
            </ClerkProvider>
          )}
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;
