import { Switch, Route, Redirect, useLocation } from "wouter";
import { useEffect } from "react";
import { useClerk, useAuth as useClerkAuth } from '@clerk/clerk-react';
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
import SimpleLanding from "@/pages/SimpleLanding";
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
import NotFound from "@/pages/not-found";
import { AdminRoleDebug } from "@/components/AdminRoleDebug";
import AccountSecurity from "@/pages/AccountSecurity";
import { TenantRequiredBanner } from "@/components/TenantRequiredBanner";

function Router() {
  const { isAuthenticated, isLoading } = useAuth();
  const { tenantId, isLoading: isTenantLoading } = useTenant();
  const [loc, setLocation] = useLocation();
  const clerk = useClerk();
  const clerkAuth = useClerkAuth();

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
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <TenantSelector />
      </div>
    );
  }


  return (
    <>
      <TenantRequiredBanner />
      <AdminRoleDebug />
      <Switch>
        {!isAuthenticated ? (
          <>
            <Route path="/" component={SimpleLanding} />
            <Route path="/landing" component={SimpleLanding} />
            {/* Friendly aliases */}
            <Route path="/signin">{() => <Redirect to="/sign-in" />}</Route>
            <Route path="/signup">{() => <Redirect to="/sign-up" />}</Route>
            {/* Clerk auth routes */}
            <Route path="/sign-in">
              {() => (
                <SignIn 
                  onBack={() => setLocation("/landing")} 
                  onSignUp={() => setLocation("/sign-up")} 
                />
              )}
            </Route>
            <Route path="/sign-up">
              {() => (
                <SignUp 
                  onBack={() => setLocation("/sign-in")} 
                  onSignIn={() => setLocation("/sign-in")} 
                />
              )}
            </Route>
            <Route path="/orgs">
              {() => (
                <div className="min-h-screen flex items-center justify-center bg-background">
                  <TenantSelector />
                </div>
              )}
            </Route>
            <Route component={SimpleLanding} />
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
            {/* Route based on user role */}
            {localStorage.getItem('user_system_role') === 'admin' ? (
              <>
                {/* Platform Owner routes only */}
                <Route path="/" component={PlatformDashboard} />
                <Route path="/platform" component={PlatformDashboard} />
                <Route path="/users" component={Users} />
                <Route path="/tenants" component={Tenants} />
                <Route path="/preferences" component={Preferences} />
                <Route path="/health" component={SystemHealth} />
                <Route path="/account/security" component={AccountSecurity} />
                <Route component={NotFound} />
              </>
            ) : (
              <>
                {/* Regular User routes only */}
                <Route path="/" component={Dashboard} />
                <Route path="/dashboard" component={Dashboard} />
                <Route path="/rulepacks" component={RulePacks} />
                <Route path="/tools" component={Tools} />
                <Route path="/tool-policies" component={ToolPolicies} />
                <Route path="/policies" component={PolicyAssignments} />
                {/* Back-compat redirects */}
                <Route path="/policy-assignment">{() => <Redirect to="/policies" />}</Route>
                <Route path="/policy-assignments">{() => <Redirect to="/policies" />}</Route>
                <Route path="/violations" component={Violations} />
                <Route path="/services" component={Services} />
                <Route path="/organization" component={Organization} />
                <Route path="/audit" component={AuditLogs} />
                <Route path="/preferences" component={Preferences} />
                <Route path="/account/security" component={AccountSecurity} />
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
