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

// Get headers for your backend with frontend bypass authentication
const getApiHeaders = () => {
  // Minimal headers; rely on same-origin cookies (Clerk) and server-issued JWT to gateway
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
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
