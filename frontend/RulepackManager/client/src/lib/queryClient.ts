import { QueryClient, QueryFunction } from "@tanstack/react-query";

function emitTenantRequired() {
  try {
    // Clear any stale local selection so Router shows TenantSelector
    localStorage.removeItem('promptshield_tenant_id');
  } catch {}
  try {
    const ev = new CustomEvent('ps:tenant-required');
    window.dispatchEvent(ev);
  } catch {}
}

async function throwIfResNotOk(res: Response) {
  if (!res.ok) {
    // Special-case tenant context requirement (BFF returns 428 status)
    if (res.status === 428) {
      emitTenantRequired();
      const text = (await res.text()) || 'TENANT_REQUIRED';
      throw new Error(`${res.status}: ${text}`);
    }
    const text = (await res.text()) || res.statusText;
    // Try to detect TENANT_REQUIRED even if status was 200-range with error payloads
    if (/TENANT_REQUIRED/i.test(text)) {
      emitTenantRequired();
    }
    throw new Error(`${res.status}: ${text}`);
  }
}

// Get current user ID for X-PS-User-ID header
const getUserId = (): string | null => {
  return localStorage.getItem('user_id');
};

// Get current user name for X-PS-User-Name header
const getUserName = (): string | null => {
  return localStorage.getItem('user_name');
};

// Get current tenant ID for X-Tenant-ID header
const getTenantId = (): string | null => {
  return localStorage.getItem('promptshield_tenant_id');
};

// Dev fallback values (used when user/tenant not yet selected)
const DEV_FALLBACK_TENANT = '6f4d338d-f0c0-4091-b54e-f71752c8f568';
const DEV_FALLBACK_USER_ID = 'frontend-dev-user';
const DEV_FALLBACK_USER_NAME = 'frontend-dev';

// Get headers for your backend with frontend bypass authentication
const getApiHeaders = () => {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-PS-Frontend-Auth': 'verified'  // Backend trusts frontend auth
  };
  
  // User context headers (required by your middleware)
  const userId = getUserId() || DEV_FALLBACK_USER_ID;
  if (userId) {
    headers['X-PS-User-ID'] = userId;
  }
  
  const userName = getUserName() || DEV_FALLBACK_USER_NAME;
  if (userName) {
    headers['X-PS-User-Name'] = userName;
  }
  
  // Tenant context header (required for RLS)
  const tenantId = getTenantId() || DEV_FALLBACK_TENANT;
  headers['X-Tenant-ID'] = tenantId;
  
  return headers;
};

export async function apiRequest(
  method: string,
  url: string,
  data?: unknown | undefined,
): Promise<Response> {
  const res = await fetch(url, {
    method,
    headers: getApiHeaders(),
    body: data ? JSON.stringify(data) : undefined,
    credentials: "include",
  });

  await throwIfResNotOk(res);
  return res;
}

type UnauthorizedBehavior = "returnNull" | "throw";
export const getQueryFn: <T>(options: {
  on401: UnauthorizedBehavior;
}) => QueryFunction<T> =
  ({ on401: unauthorizedBehavior }) =>
  async ({ queryKey }) => {
    const res = await fetch(queryKey.join("/") as string, {
      headers: getApiHeaders(),
      credentials: "include",
    });

    if (unauthorizedBehavior === "returnNull" && res.status === 401) {
      return null;
    }

    await throwIfResNotOk(res);
    return await res.json();
  };

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      queryFn: getQueryFn({ on401: "throw" }),
      refetchInterval: false,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000, // 5 minutes - cache data for better performance
      gcTime: 10 * 60 * 1000, // 10 minutes - keep in cache longer
      retry: false,
    },
    mutations: {
      retry: false,
    },
  },
});
