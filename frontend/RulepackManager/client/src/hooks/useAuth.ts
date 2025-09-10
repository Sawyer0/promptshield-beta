import { useEffect, useState } from "react";
import { useUser, useAuth as useClerkAuth } from "@clerk/clerk-react";
import { isDevBypassClient } from "@/lib/dev";

export interface User {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  name?: string;
}

export function useAuth() {
  const devBypass = isDevBypassClient();
  if (devBypass) {
    return {
      user: {
        id: import.meta.env.VITE_DEV_USER_ID || 'dev-user',
        email: import.meta.env.VITE_DEV_USER_EMAIL || 'dev@example.com',
        firstName: undefined,
        lastName: undefined,
        systemRole: (localStorage.getItem('user_system_role') as any) || 'user',
      },
      isLoading: false,
      isAuthenticated: true,
      getToken: async () => undefined,
    } as any;
  }
  const { isLoaded: authLoaded, isSignedIn, getToken } = useClerkAuth();
  const { isLoaded: userLoaded, user } = useUser();

  // Proactively activate a pending Clerk session (common after refresh)
  const [activated, setActivated] = useState(false);
  useEffect(() => {
    // Only attempt if not already signed in
    if (isSignedIn || activated) return;
    try {
      const anyClerk: any = (globalThis as any).Clerk;
      const loaded = !!anyClerk?.loaded;
      const session = anyClerk?.session;
      const last = anyClerk?.client?.lastActiveSessionId;
      if (loaded && !session && last) {
        anyClerk.setActive({ session: last })
          .then(() => setActivated(true))
          .catch(() => {});
      }
    } catch { /* ignore */ }
  }, [isSignedIn, activated]);

  const hasPending = (() => {
    try { return !!(globalThis as any).Clerk?.client?.lastActiveSessionId; } catch { return false; }
  })();
  const isAuthenticated = !!isSignedIn || (hasPending && activated);
  // Consider loaded when auth state is known and, if signed in, user has loaded
  const isLoading = !(authLoaded && (isAuthenticated ? userLoaded : true));

  return {
    user: user
      ? {
          id: user.id,
          email: user.primaryEmailAddress?.emailAddress || user.emailAddresses[0]?.emailAddress,
          firstName: user.firstName || undefined,
          lastName: user.lastName || undefined,
          systemRole: 'user',
        }
      : undefined,
    isLoading,
    isAuthenticated,
    getToken,
  };
}
