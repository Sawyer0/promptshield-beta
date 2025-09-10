import { isDevBypassClient } from "@/lib/dev";
import {
  useClerk as realUseClerk,
  useAuth as realUseClerkAuth,
  useSignIn as realUseSignIn,
  useSignUp as realUseSignUp,
  useUser as realUseUser,
} from "@clerk/clerk-react";

export function useSafeClerk(): any {
  if (isDevBypassClient()) {
    return {
      signOut: async () => {},
      setActive: async () => {},
    } as any;
  }
  return realUseClerk();
}

export function useSafeClerkAuth(): any {
  if (isDevBypassClient()) {
    return { isSignedIn: true } as any;
  }
  return realUseClerkAuth();
}

export function useSafeSignIn(): any {
  if (isDevBypassClient()) {
    return { isLoaded: false } as any;
  }
  return realUseSignIn();
}

export function useSafeSignUp(): any {
  if (isDevBypassClient()) {
    return { isLoaded: false } as any;
  }
  return realUseSignUp();
}

export function useSafeUser(): any {
  if (isDevBypassClient()) {
    return { user: undefined, isLoaded: true } as any;
  }
  return realUseUser();
}

