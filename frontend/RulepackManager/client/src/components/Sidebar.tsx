import { useLocation } from "wouter";
import { cn } from "@/lib/utils";
import { useAuth } from "@/hooks/useAuth";
import { useClerk } from '@clerk/clerk-react';
import { isSystemAdmin, getUserSystemRole } from "@/utils/frontendAuth";
import { Shield, LayoutDashboard, FileText, Building, Link as LinkIcon, ClipboardList, Activity, User, Users, Crown, X, Settings, Target, AlertTriangle, Server } from "lucide-react";

const navigation = [
  // Platform Owner (SaaS Admin) views only
  { name: "Platform Overview", href: "/platform", icon: Activity, view: "platform", roles: ["admin"] },
  { name: "Users", href: "/users", icon: Users, view: "users", roles: ["admin"] },
  { name: "Tenants", href: "/tenants", icon: Building, view: "tenants", roles: ["admin"] },
  { name: "System Health", href: "/health", icon: Activity, view: "health", roles: ["admin"] },
  
  // User views (only for regular users, not platform admins)
  { name: "Dashboard", href: "/", icon: LayoutDashboard, view: "dashboard", roles: ["user"] },
  { name: "RulePacks", href: "/rulepacks", icon: FileText, view: "rulepacks", roles: ["user"] },
  { name: "RulePack Assignments", href: "/policies", icon: Target, view: "policy-assignments", roles: ["user"] },
  { name: "Violations", href: "/violations", icon: AlertTriangle, view: "violations", roles: ["user"] },
  { name: "Services", href: "/services", icon: Server, view: "services", roles: ["user"] },
  { name: "Organization", href: "/organization", icon: Building, view: "organization", roles: ["user"] },
  { name: "Tool Policies", href: "/tool-policies", icon: Settings, view: "tool-policies", roles: ["user"] },
  { name: "Audit Logs", href: "/audit", icon: ClipboardList, view: "audit", roles: ["user"] },
];

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

export function Sidebar({ isOpen, onClose }: SidebarProps) {
  const [location] = useLocation();
  const { user } = useAuth();
  const userRole = getUserSystemRole();
  const { signOut } = useClerk();

  const getUserInitials = (user: any) => {
    if (user?.firstName && user?.lastName) {
      return `${user.firstName[0]}${user.lastName[0]}`.toUpperCase();
    }
    if (user?.email) {
      return user.email.substring(0, 2).toUpperCase();
    }
    return "U";
  };

  const getUserName = (user: any) => {
    if (user?.firstName && user?.lastName) {
      return `${user.firstName} ${user.lastName}`;
    }
    if (user?.email) {
      return user.email;
    }
    return "User";
  };

  return (
    <>
      {/* Backdrop overlay */}
      {isOpen && (
        <div 
          className="fixed inset-0 bg-black bg-opacity-50 z-40"
          onClick={onClose}
        />
      )}
      
      {/* Enterprise Security Sidebar */}
      <aside 
        className={cn(
          "fixed inset-y-0 left-0 z-50 w-80 sm:w-80 transition-transform duration-300 ease-in-out",
          "bg-white dark:bg-gray-950 border-r border-sidebar-border shadow-2xl",
          isOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <div className="flex flex-col h-full relative">
          {/* Enterprise Header */}
          <div className="flex items-center justify-between h-16 sm:h-20 px-4 sm:px-6 border-b border-sidebar-border">
            <div className="flex items-center space-x-3 sm:space-x-4">
              <div className="relative">
                <div className="h-8 w-8 sm:h-10 sm:w-10 bg-gradient-to-br from-emerald-500 to-green-600 rounded-xl flex items-center justify-center shadow-lg">
                  <Shield className="h-4 w-4 sm:h-6 sm:w-6 text-white" />
                </div>
                <div className="absolute -top-1 -right-1 h-2 w-2 sm:h-3 sm:w-3 bg-emerald-400 rounded-full border-2 border-slate-900"></div>
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-lg sm:text-xl font-bold tracking-tight text-sidebar-foreground truncate" style={{ fontFamily: 'var(--font-display)' }}>
                  PromptShield
                </div>
                <div className="text-xs font-medium text-sidebar-muted tracking-wider uppercase">
                  Security Command
                </div>
              </div>
            </div>
            <button 
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-slate-800/50 transition-colors group"
              data-testid="sidebar-close"
            >
              <X className="h-5 w-5 text-sidebar-muted group-hover:text-sidebar-foreground transition-colors" />
            </button>
          </div>

          {/* Enterprise Navigation */}
          <div className="flex-1 py-4 sm:py-6 overflow-y-auto">
            <div className="px-4 sm:px-6 mb-4 sm:mb-6">
              <div className="text-xs font-bold text-sidebar-muted uppercase tracking-widest mb-3" style={{ fontFamily: 'var(--font-security)', letterSpacing: '0.1em' }}>
                Security Operations
              </div>
            </div>
            
            <nav className="px-3 sm:px-4 space-y-1 sm:space-y-2">
              {navigation
                .filter((item) => {
                  if (userRole === 'admin') {
                    return item.roles.includes('admin') && !item.roles.includes('user');
                  } else {
                    return item.roles.includes('user') && !item.roles.includes('admin');
                  }
                })
                .map((item) => {
                  const isActive = location === item.href;
                  return (
                    <a
                      key={item.name}
                      href={item.href}
                      className={cn(
                        "flex items-center px-3 sm:px-4 py-4 sm:py-3.5 text-sm font-semibold rounded-xl transition-all duration-200 group relative",
                        "border border-transparent touch-manipulation",
                        isActive
                          ? "bg-gradient-to-r from-emerald-500/20 to-green-500/20 text-sidebar-primary-foreground border-emerald-500/30 shadow-lg shadow-emerald-500/20"
                          : "text-sidebar-foreground hover:text-sidebar-primary-foreground hover:bg-sidebar-accent hover:border-sidebar-border"
                      )}
                      style={{ fontFamily: 'var(--font-security)' }}
                      data-testid={`nav-${item.view}`}
                    >
                      <div className={cn(
                        "p-2 rounded-lg mr-3 sm:mr-4 transition-all duration-200 flex-shrink-0",
                        isActive 
                          ? "bg-emerald-500/20 shadow-lg" 
                          : "bg-sidebar-accent/60 group-hover:bg-sidebar-accent"
                      )}>
                        <item.icon className="h-5 w-5 sm:h-5 sm:w-5" />
                      </div>
                      <span className="font-bold tracking-wide text-sm sm:text-sm truncate">
                        {item.name}
                      </span>
                      {isActive && (
                        <div className="absolute right-3 h-2 w-2 bg-emerald-400 rounded-full shadow-lg shadow-emerald-400/50"></div>
                      )}
                    </a>
                  );
                })
              }
            </nav>
          </div>

          {/* Enterprise Configuration */}
          <div className="px-4 sm:px-6 py-4 sm:py-6 border-t border-sidebar-border">
            <div className="text-xs font-bold text-sidebar-muted uppercase tracking-widest mb-3 sm:mb-4" style={{ fontFamily: 'var(--font-security)', letterSpacing: '0.1em' }}>
              Configuration
            </div>
            <div className="space-y-1 sm:space-y-2">
              {userRole === 'admin' ? (
                <>
                  <div className="flex items-center px-3 sm:px-4 py-3 text-sm font-medium text-sidebar-foreground hover:text-sidebar-primary-foreground hover:bg-sidebar-accent rounded-lg transition-all duration-200 cursor-pointer group touch-manipulation">
                    <div className="p-1.5 rounded-md bg-sidebar-accent/60 group-hover:bg-sidebar-accent mr-3 transition-all flex-shrink-0">
                      <Settings className="h-4 w-4" />
                    </div>
                    <span className="truncate" style={{ fontFamily: 'var(--font-security)' }}>System Control</span>
                  </div>
                  <div className="flex items-center px-3 sm:px-4 py-3 text-sm font-medium text-sidebar-foreground hover:text-sidebar-primary-foreground hover:bg-sidebar-accent rounded-lg transition-all duration-200 cursor-pointer group touch-manipulation">
                    <div className="p-1.5 rounded-md bg-sidebar-accent/60 group-hover:bg-sidebar-accent mr-3 transition-all flex-shrink-0">
                      <Shield className="h-4 w-4" />
                    </div>
                    <span className="truncate" style={{ fontFamily: 'var(--font-security)' }}>Security Policies</span>
                  </div>
                </>
              ) : (
                <>
                  <div className="flex items-center px-3 sm:px-4 py-3 text-sm font-medium text-sidebar-foreground hover:text-sidebar-primary-foreground hover:bg-sidebar-accent rounded-lg transition-all duration-200 cursor-pointer group touch-manipulation">
                    <div className="p-1.5 rounded-md bg-sidebar-accent/60 group-hover:bg-sidebar-accent mr-3 transition-all flex-shrink-0">
                      <User className="h-4 w-4" />
                    </div>
                    <span className="truncate" style={{ fontFamily: 'var(--font-security)' }}>Account Settings</span>
                  </div>
                  <a href="/preferences" className="flex items-center px-3 sm:px-4 py-3 text-sm font-medium text-sidebar-foreground hover:text-sidebar-primary-foreground hover:bg-sidebar-accent rounded-lg transition-all duration-200 group touch-manipulation">
                    <div className="p-1.5 rounded-md bg-sidebar-accent/60 group-hover:bg-sidebar-accent mr-3 transition-all flex-shrink-0">
                      <Settings className="h-4 w-4" />
                    </div>
                    <span className="truncate" style={{ fontFamily: 'var(--font-security)' }}>Preferences</span>
                  </a>
                </>
              )}
            </div>
          </div>

          {/* Enterprise User Profile */}
          <div className="px-4 sm:px-6 py-4 sm:py-6 border-t border-sidebar-border bg-sidebar-accent/30">
            <div className="flex items-center space-x-3 sm:space-x-4">
              <div className="relative flex-shrink-0">
                <div className="h-8 w-8 sm:h-10 sm:w-10 bg-gradient-to-br from-emerald-500 to-green-600 rounded-xl flex items-center justify-center shadow-lg">
                  <span className="text-xs sm:text-sm font-bold text-white" data-testid="user-initials" style={{ fontFamily: 'var(--font-display)' }}>
                    {getUserInitials(user)}
                  </span>
                </div>
                <div className="absolute -bottom-1 -right-1 h-2 w-2 sm:h-3 sm:w-3 bg-emerald-400 rounded-full border-2 border-sidebar-bg"></div>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs sm:text-sm font-bold text-sidebar-foreground truncate" data-testid="user-name" style={{ fontFamily: 'var(--font-security)' }}>
                  {getUserName(user)}
                </p>
                <div className="flex items-center mt-1">
                  {userRole === 'admin' && <Crown className="h-2 w-2 sm:h-3 sm:w-3 mr-1 sm:mr-2 text-yellow-400 flex-shrink-0" />}
                  <p className="text-xs font-medium text-sidebar-muted uppercase tracking-wider truncate" style={{ fontFamily: 'var(--font-security)' }}>
                    {userRole === 'admin' ? 'Platform Owner' : 'Security User'}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-1">
                <button className="p-1.5 sm:p-2 rounded-lg hover:bg-sidebar-accent transition-colors group flex-shrink-0 touch-manipulation" title="Settings">
                  <Settings className="h-3 w-3 sm:h-4 sm:w-4 text-sidebar-muted group-hover:text-sidebar-foreground transition-colors" />
                </button>
                <button
                  className="p-1.5 sm:p-2 rounded-lg hover:bg-sidebar-accent transition-colors group flex-shrink-0 touch-manipulation"
                  title="Sign out"
                  onClick={async () => {
                    try { await fetch('/api/auth/signout', { method: 'POST', credentials: 'include' }); } catch {}
                    try { await fetch('/api/session/clear', { method: 'POST', credentials: 'include' }); } catch {}
                    try { await signOut({ redirectUrl: '/sign-in' }); } catch { window.location.href = '/sign-in'; }
                  }}
                >
                  <X className="h-3 w-3 sm:h-4 sm:w-4 text-sidebar-muted group-hover:text-sidebar-foreground transition-colors" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </>
  );
}
