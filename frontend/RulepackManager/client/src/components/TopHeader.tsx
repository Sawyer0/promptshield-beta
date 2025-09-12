import { Menu, Moon, Sun, Settings, Bell, Crown, X, User, LogOut, LogIn, Shield } from "lucide-react";
import { useTheme } from "@/components/ThemeProvider";
import { Button } from "@/components/ui/button";
import { TenantSelector } from "@/components/TenantSelector";
import { getUserSystemRole } from "@/utils/frontendAuth";
import { useState } from "react";
import { useToast } from "@/hooks/use-toast";
import { useAuth } from "@/hooks/useAuth";
import { useSafeClerk, useSafeClerkAuth } from '@/clerk/shim';
import { SignInModal } from '@/components/auth/SignInModal';
import { SignUpModal } from '@/components/auth/SignUpModal';
import { useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Link } from "wouter";

interface TopHeaderProps {
  title: string;
  description: string;
  onMenuClick: () => void;
}

export function TopHeader({ title, description, onMenuClick }: TopHeaderProps) {
  const { theme, setTheme } = useTheme();
  const { user, isAuthenticated } = useAuth();
  const queryClient = useQueryClient();
  const userRole = getUserSystemRole();
  const isAdmin = userRole === 'admin';
  let roles: string[] = [];
  try { const raw = localStorage.getItem('ps_roles'); roles = raw ? JSON.parse(raw) : []; } catch {}

  const { signOut } = useSafeClerk();
  const { isSignedIn: clerkSignedIn } = useSafeClerkAuth();
  const [signInOpen, setSignInOpen] = useState(false);
  const [signUpOpen, setSignUpOpen] = useState(false);
  const { toast } = useToast();
  const handleLogout = async () => {
    try {
      // Clear local auth hints
      localStorage.removeItem('user_system_role');
      localStorage.removeItem('selected_tenant_id'); // legacy key
      localStorage.removeItem('promptshield_tenant_id');
      localStorage.removeItem('promptshield_tenant_name');
      queryClient.clear();
      try {
        // Clear server-side cookies/session
        await fetch('/api/auth/signout', { method: 'POST', credentials: 'include' });
        await fetch('/api/session/clear', { method: 'POST', credentials: 'include' });
      } catch {}
      // End Clerk session fully
      await signOut();
      // Hard redirect to landing
      window.location.href = '/landing';
    } catch (error) {
      window.location.href = '/landing';
    }
  };

  const handleLogin = () => setSignInOpen(true);

  const toggleTheme = () => {
    setTheme(theme === "dark" ? "light" : "dark");
  };

  const notifications = isAdmin ? [
    { id: 1, type: 'info', title: 'System Health Check', message: 'All systems operational', time: '2 minutes ago' },
    { id: 2, type: 'success', title: 'New Tenant Registration', message: 'TechStart Inc has joined the platform', time: '1 hour ago' },
    { id: 3, type: 'warning', title: 'High Usage Alert', message: 'Platform usage: 8.7M requests today (+15%)', time: '3 hours ago' }
  ] : [
    { id: 1, type: 'info', title: 'Policy Update', message: 'RulePack "Security v2.1" has been updated', time: '30 minutes ago' },
    { id: 2, type: 'success', title: 'Scan Complete', message: 'Daily security scan completed successfully', time: '2 hours ago' }
  ];

  return (
    <header className="bg-card border-b border-border">
      <div className="px-4 sm:px-6 py-3 sm:py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3 sm:space-x-4 min-w-0 flex-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={onMenuClick}
              data-testid="menu-toggle"
              className="flex-shrink-0"
            >
              <Menu className="h-5 w-5" />
            </Button>
            <div className="min-w-0 flex-1">
              <h1 className="text-lg sm:text-2xl font-bold text-foreground flex items-center gap-1 sm:gap-2 truncate" data-testid="page-title">
                <span className="truncate">Prompt Shield</span>
                <Shield className="h-4 w-4 sm:h-5 sm:w-5 text-primary flex-shrink-0" />
                {isAdmin && <Crown className="h-4 w-4 sm:h-5 sm:w-5 text-warning flex-shrink-0" />}
              </h1>
              
            </div>
          </div>
          <div className="flex items-center space-x-2 sm:space-x-4 md:space-x-6 flex-shrink-0">
            {/* Tenant Selector - Only for regular users, not platform admins */}
            {!isAdmin && (
              <div className="hidden sm:block">
                <TenantSelector />
              </div>
            )}
            
            <div className="flex items-center space-x-2 sm:space-x-4">
              {/* Dark Mode Toggle */}
              <Button
                variant="ghost"
                size="sm"
                onClick={toggleTheme}
                data-testid="theme-toggle"
              >
                {theme === "dark" ? (
                  <Sun className="h-5 w-5" />
                ) : (
                  <Moon className="h-5 w-5" />
                )}
              </Button>
              
              {/* Notifications */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button 
                    variant="ghost" 
                    size="sm" 
                    className="relative" 
                    data-testid="notifications"
                  >
                    <Bell className="h-5 w-5" />
                    <span className="absolute top-1 right-1 h-2 w-2 bg-destructive rounded-full"></span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-72 sm:w-80 max-w-[90vw]">
                  <DropdownMenuLabel>Notifications</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <div className="max-h-80 sm:max-h-96 overflow-y-auto">
                    {notifications.map((notification) => (
                      <div key={notification.id} className="flex items-start space-x-2 sm:space-x-3 p-2 sm:p-3 hover:bg-accent rounded-sm">
                        <div className={`w-2 h-2 rounded-full mt-2 flex-shrink-0 ${
                          notification.type === 'success' ? 'bg-green-500' : 
                          notification.type === 'warning' ? 'bg-yellow-500' : 
                          'bg-blue-500'
                        }`} />
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-sm truncate">{notification.title}</div>
                          <div className="text-muted-foreground text-sm line-clamp-2">{notification.message}</div>
                          <div className="text-xs text-muted-foreground mt-1">{notification.time}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                </DropdownMenuContent>
              </DropdownMenu>

              {/* User Menu */}
              {(isAuthenticated || clerkSignedIn) ? (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="flex items-center gap-2" data-testid="user-menu">
                      <User className="h-4 w-4" />
                      <span className="hidden sm:inline text-sm">{user?.email}</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-56">
                    <DropdownMenuLabel className="flex items-center gap-2">
                      <User className="h-4 w-4" />
                      <div>
                        <div className="font-medium">{user?.email}</div>
                        <div className="text-xs text-muted-foreground">
                          {isAdmin ? 'Platform Administrator' : 'User'}
                        </div>
                        {Array.isArray(roles) && roles.length > 0 && (
                          <div className="mt-1 text-[10px] text-muted-foreground">Roles: {roles.join(', ')}</div>
                        )}
                      </div>
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem asChild>
                      <Link href="/account/security">Account security</Link>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={handleLogout} className="text-destructive">
                      <LogOut className="mr-2 h-4 w-4" />
                      Sign out
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : (
                <>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    data-testid="login-button"
                    className="flex items-center gap-2"
                    onClick={() => setSignInOpen(true)}
                  >
                    <LogIn className="h-4 w-4" />
                    Sign in
                  </Button>
                  <Button 
                    variant="ghost" 
                    size="sm"
                    onClick={() => setSignUpOpen(true)}
                  >
                    Sign up
                  </Button>
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Auth Modals */}
      <SignInModal
        open={signInOpen}
        onOpenChange={setSignInOpen}
        onSuccess={() => { /* stay on current route */ }}
      />
      <SignUpModal
        open={signUpOpen}
        onOpenChange={setSignUpOpen}
        onSuccess={async (orgName?: string) => {
          if (orgName && orgName.trim()) {
            try {
              const headers: Record<string,string> = { 'Content-Type': 'application/json' };
              try { const tok = await (window as any)?.Clerk?.session?.getToken?.(); if (tok) headers['Authorization'] = `Bearer ${tok}`; } catch {}
              const resp = await fetch('/api/orgs/create', {
                method: 'POST',
                headers,
                credentials: 'include',
                body: JSON.stringify({ name: orgName.trim() })
              });
              if (resp.ok) {
                toast({ title: 'Organization created', description: 'You are now admin of this organization.' });
              } else if (resp.status === 403) {
                toast({ title: 'Signup disabled', description: 'Self-serve tenant signup is disabled. Contact your administrator.', variant: 'destructive' });
              }
            } catch (_) {}
          }
          // stay on current route
        }}
      />
    </header>
  );
}
