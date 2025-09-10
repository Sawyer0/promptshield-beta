import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';

interface TenantContextType {
  tenantId: string | null;
  tenantName: string | null;
  setTenant: (tenantId: string, tenantName: string) => void;
  clearTenant: () => void;
  isLoading: boolean;
}

const TenantContext = createContext<TenantContextType | undefined>(undefined);

export function useTenant() {
  const context = useContext(TenantContext);
  if (!context) {
    throw new Error('useTenant must be used within TenantProvider');
  }
  return context;
}

interface TenantProviderProps {
  children: ReactNode;
}

export function TenantProvider({ children }: TenantProviderProps) {
  const [tenantId, setTenantId] = useState<string | null>(null);
  const [tenantName, setTenantName] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const bootstrap = async () => {
      // Try localStorage first
      const savedTenantId = localStorage.getItem('promptshield_tenant_id');
      const savedTenantName = localStorage.getItem('promptshield_tenant_name');
      if (savedTenantId && savedTenantName) {
        setTenantId(savedTenantId);
        setTenantName(savedTenantName);
        setIsLoading(false);
        return;
      }

      // Try signed cookie (server authoritative)
      try {
        const m = document.cookie.match(/(?:^|; )ps_tenant_id=([^;]+)/);
        if (m && m[1]) {
          const id = decodeURIComponent(m[1]);
          // Ask server for current session tenant to get display name
          const resp = await fetch('/api/session/tenant', { credentials: 'include' });
          if (resp.ok) {
            const data = await resp.json();
            const name = data?.tenantName || data?.name || '';
            if (id) {
              setTenantId(id);
              setTenantName(name || '');
              if (id) localStorage.setItem('promptshield_tenant_id', id);
              if (name) localStorage.setItem('promptshield_tenant_name', name);
              setIsLoading(false);
              return;
            }
          }
        }
      } catch { /* ignore */ }

      // Fallback to server session (no cookie path)
      try {
        const resp = await fetch('/api/session/tenant', { credentials: 'include' });
        if (resp.ok) {
          const data = await resp.json();
          const id = data?.tenantId || data?.tenant_id || '';
          const name = data?.tenantName || data?.name || '';
          if (id && name) {
            setTenantId(id);
            setTenantName(name);
            localStorage.setItem('promptshield_tenant_id', id);
            localStorage.setItem('promptshield_tenant_name', name);
          }
        }
      } catch { /* ignore */ }

      setIsLoading(false);
    };

    bootstrap();
  }, []);

  const setTenant = useCallback((newTenantId: string, newTenantName: string) => {
    setTenantId(newTenantId);
    setTenantName(newTenantName);
    localStorage.setItem('promptshield_tenant_id', newTenantId);
    localStorage.setItem('promptshield_tenant_name', newTenantName);
  }, []);

  const clearTenant = useCallback(() => {
    setTenantId(null);
    setTenantName(null);
    localStorage.removeItem('promptshield_tenant_id');
    localStorage.removeItem('promptshield_tenant_name');
  }, []);

  return (
    <TenantContext.Provider value={{
      tenantId,
      tenantName,
      setTenant,
      clearTenant,
      isLoading
    }}>
      {children}
    </TenantContext.Provider>
  );
}
