// Holds the admin token for this browser tab and the client built from it.
// The token lives in sessionStorage: it survives a reload but not closing
// the tab, and it is never written to the URL or a cookie.
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { GatewayClient, type FetchLike } from '../api/gatewayClient';

const STORAGE_KEY = 'promptshield.adminToken';

export interface Session {
  client: GatewayClient | null;
  /** Set when the gateway rejected the stored token. */
  expired: boolean;
  signIn: (token: string) => Promise<void>;
  signOut: () => void;
  /** Called by data hooks when an admin call answers 401. */
  expire: () => void;
}

const SessionContext = createContext<Session | null>(null);

interface SessionProviderProps {
  children: ReactNode;
  baseUrl?: string;
  fetchFn?: FetchLike;
}

export function SessionProvider({ children, baseUrl = '', fetchFn }: SessionProviderProps) {
  const build = useCallback(
    (token: string) => new GatewayClient(fetchFn ? { token, baseUrl, fetchFn } : { token, baseUrl }),
    [baseUrl, fetchFn],
  );
  const [client, setClient] = useState<GatewayClient | null>(() => {
    const stored = sessionStorage.getItem(STORAGE_KEY);
    return stored ? build(stored) : null;
  });
  const [expired, setExpired] = useState(false);

  const signIn = useCallback(
    async (token: string) => {
      const candidate = build(token.trim());
      await candidate.verifyToken();
      sessionStorage.setItem(STORAGE_KEY, candidate.token);
      setExpired(false);
      setClient(candidate);
    },
    [build],
  );

  const signOut = useCallback(() => {
    sessionStorage.removeItem(STORAGE_KEY);
    setClient(null);
  }, []);

  const expire = useCallback(() => {
    sessionStorage.removeItem(STORAGE_KEY);
    setExpired(true);
    setClient(null);
  }, []);

  const value = useMemo(() => ({ client, expired, signIn, signOut, expire }), [client, expired, signIn, signOut, expire]);
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): Session {
  const session = useContext(SessionContext);
  if (!session) throw new Error('useSession must be used inside <SessionProvider>');
  return session;
}

/** For components rendered only after sign-in. */
export function useClient(): GatewayClient {
  const { client } = useSession();
  if (!client) throw new Error('useClient requires a signed-in session');
  return client;
}
