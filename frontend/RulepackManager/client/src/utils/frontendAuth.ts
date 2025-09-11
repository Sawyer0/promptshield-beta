/**
 * Frontend authentication utilities for PromptShield
 * Handles user authentication, session management, and tenant context
 */

export interface User {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  name?: string;
  systemRole?: 'admin' | 'user';
  tenants?: UserTenant[];
}

export interface UserTenant {
  tenant_id: string;
  tenant_name: string;
  role: string;
}

/**
 * Authenticates a user with email and password
 * @param email User's email address
 * @param password User's password
 * @returns User object if authentication successful, null otherwise
 */
export async function authenticateUser(email: string, password: string): Promise<User | null> {
  // Mock authentication - replace with your actual auth system
  // (Supabase Auth, Auth0, Firebase Auth, etc.)
  
  if (email && password) {
    // Determine system role based on email (you can replace with your actual logic)
    const systemRole = email.includes('admin') ? 'admin' : 'user';
    
    return {
      id: 'user_123',
      email: email,
      firstName: 'John',
      lastName: 'Doe',
      name: 'John Doe',
      systemRole,
      tenants: [
        { tenant_id: '550e8400-e29b-41d4-a716-446655440001', tenant_name: 'Acme Corporation', role: 'admin' },
        { tenant_id: '550e8400-e29b-41d4-a716-446655440002', tenant_name: 'TechStartup Inc', role: 'user' }
      ]
    };
  }
  
  return null;
}

/**
 * Retrieves all tenants associated with a user
 * @param userId The user's unique identifier
 * @returns Array of tenants the user has access to
 */
export async function getUserTenants(userId: string): Promise<UserTenant[]> {
  try {
    // This should call your actual backend endpoint
    // For now, return mock data
    return [
      { tenant_id: '550e8400-e29b-41d4-a716-446655440001', tenant_name: 'Acme Corporation', role: 'admin' },
      { tenant_id: '550e8400-e29b-41d4-a716-446655440002', tenant_name: 'TechStartup Inc', role: 'user' },
      { tenant_id: '550e8400-e29b-41d4-a716-446655440003', tenant_name: 'Enterprise Solutions', role: 'user' }
    ];
  } catch (error) {
    // Error handled by returning empty array
    return [];
  }
}

/**
 * Stores user authentication context in localStorage
 * @param user The authenticated user object
 */
export function setUserAuth(user: User): void {
  localStorage.setItem('user_id', user.id);
  localStorage.setItem('user_name', user.name || `${user.firstName} ${user.lastName}`.trim());
  localStorage.setItem('user_email', user.email);
  localStorage.setItem('user_system_role', user.systemRole || 'user');
  
  if (user.tenants && user.tenants.length > 0) {
    // Auto-select first tenant if user has tenants
    const firstTenant = user.tenants[0];
    setUserTenant(firstTenant.tenant_id, firstTenant.tenant_name);
  }
}

/**
 * Sets the user's currently selected tenant for multi-tenant context
 * @param tenantId The tenant's unique identifier
 * @param tenantName The tenant's display name
 */
export function setUserTenant(tenantId: string, tenantName: string): void {
  localStorage.setItem('promptshield_tenant_id', tenantId);
  localStorage.setItem('promptshield_tenant_name', tenantName);
}

/**
 * Clears all authentication data from localStorage
 * Should be called on logout
 */
export function clearUserAuth(): void {
  localStorage.removeItem('user_id');
  localStorage.removeItem('user_name');
  localStorage.removeItem('user_email');
  localStorage.removeItem('user_system_role');
  localStorage.removeItem('promptshield_tenant_id');
  localStorage.removeItem('promptshield_tenant_name');
}

// Check if user is authenticated
export function isUserAuthenticated(): boolean {
  const userId = localStorage.getItem('user_id');
  
  // Don't auto-authenticate - let the backend handle real authentication
  return !!userId;
}

// Auto-authenticate a default user for development/demo
function autoAuthenticateUser(): void {
  const defaultUser: User = {
    id: 'user_123',
    email: 'admin@promptshield.com',
    firstName: 'John',
    lastName: 'Doe', 
    name: 'John Doe',
    systemRole: 'admin',
    tenants: [
      { tenant_id: '550e8400-e29b-41d4-a716-446655440001', tenant_name: 'Acme Corporation', role: 'admin' },
      { tenant_id: '550e8400-e29b-41d4-a716-446655440002', tenant_name: 'TechStartup Inc', role: 'user' }
    ]
  };
  
  setUserAuth(defaultUser);
}

// Get current authenticated user info
export function getCurrentUser(): User | null {
  const userId = localStorage.getItem('user_id');
  const userEmail = localStorage.getItem('user_email');
  const userName = localStorage.getItem('user_name');
  const systemRole = localStorage.getItem('user_system_role');
  
  if (!userId || !userEmail) {
    return null;
  }
  
  const nameParts = userName?.split(' ') || [];
  return {
    id: userId,
    email: userEmail,
    firstName: nameParts[0] || '',
    lastName: nameParts.slice(1).join(' ') || '',
    name: userName || '',
    systemRole: (systemRole as 'admin' | 'user') || 'user'
  };
}

// Get current tenant info
export function getCurrentTenant(): { id: string; name: string } | null {
  const tenantId = localStorage.getItem('promptshield_tenant_id');
  const tenantName = localStorage.getItem('promptshield_tenant_name');
  
  if (!tenantId || !tenantName) {
    return null;
  }
  
  return { id: tenantId, name: tenantName };
}

// Get current user's system role
export function getUserSystemRole(): 'admin' | 'user' {
  return (localStorage.getItem('user_system_role') as 'admin' | 'user') || 'user';
}

// Check if user is the SaaS platform owner/admin
export function isSystemAdmin(): boolean {
  const role = getUserSystemRole();
  return role === 'admin';
}