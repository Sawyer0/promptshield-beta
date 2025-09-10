import React, { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/toaster';
import { useLocation } from 'wouter';
import { TenantProvider } from '@/contexts/TenantContext';

// Mock wouter
vi.mock('wouter', () => ({
  useLocation: vi.fn(() => ['/', vi.fn()]),
  Route: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Switch: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

// Mock Clerk
vi.mock('@clerk/clerk-react', () => ({
  useClerk: () => ({
    signOut: vi.fn(),
  }),
  useUser: () => ({
    user: {
      id: 'dev-user',
      emailAddresses: [{ emailAddress: 'dev@example.com' }],
      firstName: 'Dev',
      lastName: 'User',
    },
    isLoaded: true,
  }),
  useAuth: () => ({
    isSignedIn: true,
    isLoaded: true,
  }),
  // Added for components that import safe shims which delegate to real hooks
  useSignIn: () => ({ isLoaded: false, signIn: vi.fn() }),
  useSignUp: () => ({ isLoaded: false, signUp: vi.fn() }),
  ClerkProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

// Create a custom render function that includes providers
const AllTheProviders = ({ children }: { children: React.ReactNode }) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return (
    <QueryClientProvider client={queryClient}>
      <TenantProvider>
        {children}
        <Toaster />
      </TenantProvider>
    </QueryClientProvider>
  );
};

const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => render(ui, { wrapper: AllTheProviders, ...options });

// Mock auth context
export const mockAuthContext = {
  userId: 'dev-user',
  userName: 'Dev User',
  email: 'dev@example.com',
  tenantId: '6f4d338d-f0c0-4091-b54e-f71752c8f568',
};

// Setup auth in localStorage
export const setupAuth = () => {
  localStorage.setItem('user_id', mockAuthContext.userId);
  localStorage.setItem('user_name', mockAuthContext.userName);
  localStorage.setItem('user_email', mockAuthContext.email);
  localStorage.setItem('tenant_id', mockAuthContext.tenantId);
  localStorage.setItem('user_system_role', 'user');
  localStorage.setItem('promptshield_tenant_id', mockAuthContext.tenantId);
  localStorage.setItem('promptshield_tenant_name', 'Dev Tenant');
};

// Clear auth from localStorage
export const clearAuth = () => {
  localStorage.removeItem('user_id');
  localStorage.removeItem('user_name');
  localStorage.removeItem('user_email');
  localStorage.removeItem('tenant_id');
  localStorage.removeItem('user_system_role');
  localStorage.removeItem('promptshield_tenant_id');
  localStorage.removeItem('promptshield_tenant_name');
};

// Mock location
export const mockLocation = (path: string) => {
  (useLocation as any).mockReturnValue([path, vi.fn()]);
};

// Helpers to mock fetch easily
export const mockFetchOnce = (payload: any, opts: Partial<Response> = { ok: true }) => {
  const ok = (opts as any).ok ?? true;
  const status = (opts as any).status ?? (ok ? 200 : 500);
  const statusText = (opts as any).statusText ?? (ok ? 'OK' : 'Internal Server Error');
  const bodyText = typeof payload === 'string' ? payload : JSON.stringify(payload);
  (global.fetch as any) = vi.fn().mockResolvedValueOnce({
    ok,
    status,
    statusText,
    headers: new Headers({ 'Content-Type': 'application/json' }),
    json: () => Promise.resolve(payload),
    text: () => Promise.resolve(ok ? bodyText : statusText),
  });
};

export const mockFetchRejectOnce = (error: Error) => {
  (global.fetch as any) = vi.fn().mockRejectedValueOnce(error);
};

// Re-export everything
export * from '@testing-library/react';
export { customRender as render };
