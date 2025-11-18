import {
  type Policy,
  type InsertPolicy,
  type Violation,
  type InsertViolation,
  type SystemMetric,
  type InsertSystemMetric,
  type RulePack,
  type InsertRulePack,
  type DashboardMetrics,
  type SystemHealth,
  type UpsertUser,
  type User,
} from "./shared/schema";

const generateId = (): string => {
  if (typeof globalThis !== "undefined") {
    const cryptoObj = (globalThis as any).crypto;
    if (cryptoObj && typeof cryptoObj.randomUUID === "function") {
      return cryptoObj.randomUUID();
    }
  }
  // Fallback: not cryptographically secure but sufficient for demo/storage
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
};

export interface IStorage {
  // Policy management
  getPolicies(): Promise<Policy[]>;
  getPolicy(id: string): Promise<Policy | undefined>;
  createPolicy(policy: InsertPolicy): Promise<Policy>;
  updatePolicy(
    id: string,
    policy: Partial<InsertPolicy>
  ): Promise<Policy | undefined>;
  deletePolicy(id: string): Promise<boolean>;
  activatePolicy(id: string): Promise<boolean>;
  deactivatePolicy(id: string): Promise<boolean>;

  // Violation management
  getViolations(): Promise<Violation[]>;
  getViolation(id: string): Promise<Violation | undefined>;
  createViolation(violation: InsertViolation): Promise<Violation>;
  getViolationsByDateRange(
    startDate: Date,
    endDate: Date
  ): Promise<Violation[]>;

  // System metrics
  getSystemMetrics(): Promise<SystemMetric[]>;
  createSystemMetric(metric: InsertSystemMetric): Promise<SystemMetric>;
  getDashboardMetrics(): Promise<DashboardMetrics>;
  getSystemHealth(): Promise<SystemHealth>;

  // RulePack library
  getRulePacks(): Promise<RulePack[]>;
  getRulePack(id: string): Promise<RulePack | undefined>;
  createRulePack(rulepack: InsertRulePack): Promise<RulePack>;
  updateRulePack(
    id: string,
    rulepack: Partial<InsertRulePack>
  ): Promise<RulePack | undefined>;

  // User operations (mandatory for Replit Auth)
  getUser(id: string): Promise<User | undefined>;
  upsertUser(user: UpsertUser): Promise<User>;
}

export class MemStorage implements IStorage {
  private policies: Map<string, Policy>;
  private violations: Map<string, Violation>;
  private systemMetrics: Map<string, SystemMetric>;
  private rulePacks: Map<string, RulePack>;

  constructor() {
    this.policies = new Map();
    this.violations = new Map();
    this.systemMetrics = new Map();
    this.rulePacks = new Map();

    // Initialize with some sample data
    this.initializeSampleData();
  }

  private initializeSampleData() {
    // Sample policies
    const samplePolicies: Policy[] = [
      {
        id: "policy-1",
        name: "PII Protection",
        type: "security",
        content: `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: pii-protection
  version: "1.0"
  description: "Detect and prevent PII leakage"
rules:
  - id: "pii-email"
    name: "Email Detection"
    level: 2
    severity: "HIGH"
    patterns:
      - regex: "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b"
    response:
      action: "quarantine"
      message: "Potential email address detected"`,
        version: 1,
        created_at: new Date("2024-01-15"),
        updated_at: new Date("2024-01-15"),
        created_by: "admin",
        is_active: true,
      },
      {
        id: "policy-2",
        name: "Toxic Content Filter",
        type: "compliance",
        content: `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: toxic-content-filter
  version: "1.0"
  description: "Filter toxic and harmful content"
rules:
  - id: "toxic-keywords"
    name: "Toxic Keywords"
    level: 1
    severity: "MEDIUM"
    keywords: ["hate", "toxic", "harmful"]
    response:
      action: "deny"
      message: "Toxic content detected"`,
        version: 1,
        created_at: new Date("2024-01-16"),
        updated_at: new Date("2024-01-16"),
        created_by: "admin",
        is_active: true,
      },
      {
        id: "policy-3",
        name: "Injection Prevention",
        type: "security",
        content: `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: injection-prevention
  version: "1.0"
  description: "Prevent prompt injection attacks"
rules:
  - id: "injection-patterns"
    name: "Injection Patterns"
    level: 2
    severity: "CRITICAL"
    patterns:
      - regex: "ignore previous instructions"
      - regex: "system prompt"
    response:
      action: "deny"
      message: "Potential injection attempt detected"`,
        version: 1,
        created_at: new Date("2024-01-17"),
        updated_at: new Date("2024-01-17"),
        created_by: "admin",
        is_active: false,
      },
    ];

    samplePolicies.forEach((policy) => this.policies.set(policy.id, policy));
  }

  async getPolicies(): Promise<Policy[]> {
    return Array.from(this.policies.values()).sort(
      (a, b) =>
        new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    );
  }

  async getPolicy(id: string): Promise<Policy | undefined> {
    return this.policies.get(id);
  }

  async createPolicy(policy: InsertPolicy): Promise<Policy> {
    const id = generateId();
    const now = new Date();
    const newPolicy: Policy = {
      ...policy,
      id,
      created_at: now,
      updated_at: now,
    };
    this.policies.set(id, newPolicy);
    return newPolicy;
  }

  async updatePolicy(
    id: string,
    policy: Partial<InsertPolicy>
  ): Promise<Policy | undefined> {
    const existing = this.policies.get(id);
    if (!existing) return undefined;

    const updated: Policy = {
      ...existing,
      ...policy,
      updated_at: new Date(),
    };
    this.policies.set(id, updated);
    return updated;
  }

  async deletePolicy(id: string): Promise<boolean> {
    return this.policies.delete(id);
  }

  async activatePolicy(id: string): Promise<boolean> {
    const policy = this.policies.get(id);
    if (!policy) return false;

    policy.is_active = true;
    policy.updated_at = new Date();
    this.policies.set(id, policy);
    return true;
  }

  async deactivatePolicy(id: string): Promise<boolean> {
    const policy = this.policies.get(id);
    if (!policy) return false;

    policy.is_active = false;
    policy.updated_at = new Date();
    this.policies.set(id, policy);
    return true;
  }

  async getViolations(): Promise<Violation[]> {
    return Array.from(this.violations.values()).sort(
      (a, b) =>
        new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );
  }

  async getViolation(id: string): Promise<Violation | undefined> {
    return this.violations.get(id);
  }

  async createViolation(violation: InsertViolation): Promise<Violation> {
    const id = generateId();
    const newViolation: Violation = {
      ...violation,
      id,
      timestamp: new Date(),
    };
    this.violations.set(id, newViolation);
    return newViolation;
  }

  async getViolationsByDateRange(
    startDate: Date,
    endDate: Date
  ): Promise<Violation[]> {
    return Array.from(this.violations.values()).filter((violation) => {
      const timestamp = new Date(violation.timestamp);
      return timestamp >= startDate && timestamp <= endDate;
    });
  }

  async getSystemMetrics(): Promise<SystemMetric[]> {
    return Array.from(this.systemMetrics.values());
  }

  async createSystemMetric(metric: InsertSystemMetric): Promise<SystemMetric> {
    const id = generateId();
    const newMetric: SystemMetric = {
      ...metric,
      id,
      timestamp: new Date(),
    };
    this.systemMetrics.set(id, newMetric);
    return newMetric;
  }

  async getDashboardMetrics(): Promise<DashboardMetrics> {
    const violations = Array.from(this.violations.values());
    const policies = Array.from(this.policies.values());

    // Calculate metrics
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const todayViolations = violations.filter(
      (v) => new Date(v.timestamp) >= today
    );

    const last24Hours = new Date(Date.now() - 24 * 60 * 60 * 1000);
    const recent24HourViolations = violations.filter(
      (v) => new Date(v.timestamp) >= last24Hours
    );

    // Generate trend data for last 7 days
    const violationTrend = [];
    for (let i = 6; i >= 0; i--) {
      const date = new Date();
      date.setDate(date.getDate() - i);
      date.setHours(0, 0, 0, 0);
      const nextDay = new Date(date);
      nextDay.setDate(nextDay.getDate() + 1);

      const dayViolations = violations.filter((v) => {
        const vDate = new Date(v.timestamp);
        return vDate >= date && vDate < nextDay;
      });

      violationTrend.push({
        timestamp: date.toISOString(),
        count: dayViolations.length,
      });
    }

    // Policy effectiveness data
    const decisionCounts = violations.reduce((acc, v) => {
      acc[v.decision] = (acc[v.decision] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const policyEffectiveness = Object.entries(decisionCounts).map(
      ([decision, count]) => ({
        decision,
        count,
      })
    );

    return {
      total_violations: violations.length,
      active_policies: policies.filter((p) => p.is_active).length,
      requests_today: recent24HourViolations.length * 10, // Simulate higher traffic
      avg_response_time: 12, // Mock average response time in ms
      violation_trend: violationTrend,
      policy_effectiveness: policyEffectiveness,
    };
  }

  async getSystemHealth(): Promise<SystemHealth> {
    return {
      api_gateway: "healthy",
      policy_engine: "active",
      ml_models: "ready",
      database: "connected",
      cpu_usage: 34,
      memory_usage: 67,
    };
  }

  async getRulePacks(): Promise<RulePack[]> {
    return Array.from(this.rulePacks.values());
  }

  async getRulePack(id: string): Promise<RulePack | undefined> {
    return this.rulePacks.get(id);
  }

  async createRulePack(rulepack: InsertRulePack): Promise<RulePack> {
    const id = generateId();
    const now = new Date();
    const newRulePack: RulePack = {
      ...rulepack,
      id,
      created_at: now,
      updated_at: now,
    };
    this.rulePacks.set(id, newRulePack);
    return newRulePack;
  }

  async updateRulePack(
    id: string,
    rulepack: Partial<InsertRulePack>
  ): Promise<RulePack | undefined> {
    const existing = this.rulePacks.get(id);
    if (!existing) return undefined;

    const updated: RulePack = {
      ...existing,
      ...rulepack,
      updated_at: new Date(),
    };
    this.rulePacks.set(id, updated);
    return updated;
  }

  // User operations for Replit Auth
  async getUser(id: string): Promise<User | undefined> {
    // In MemStorage, we'll create a mock user for development
    return {
      id,
      email: "demo@example.com",
      firstName: "Demo",
      lastName: "User",
      profileImageUrl: null,
      createdAt: new Date(),
      updatedAt: new Date(),
    };
  }

  async upsertUser(userData: UpsertUser): Promise<User> {
    // In MemStorage, we'll return a mock user based on the input
    const user: User = {
      id: userData.id || generateId(),
      email: userData.email || null,
      firstName: userData.firstName || null,
      lastName: userData.lastName || null,
      profileImageUrl: userData.profileImageUrl || null,
      createdAt: new Date(),
      updatedAt: new Date(),
    };
    return user;
  }
}

// Database storage implementation for production using Drizzle ORM
// This follows the Repository pattern with dependency injection for testability
export class DatabaseStorage implements IStorage {
  // Drizzle DB instance would be injected via constructor
  // private db: DrizzleDB;
  // private schema: typeof import('./db/schema');

  // constructor(db: DrizzleDB, schema: typeof import('./db/schema')) {
  //   this.db = db;
  //   this.schema = schema;
  // }

  async getPolicies(): Promise<Policy[]> {
    // Example Drizzle implementation:
    // return await this.db.select().from(this.schema.policies).orderBy(desc(this.schema.policies.createdAt));
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getPolicy(id: string): Promise<Policy | undefined> {
    // Example: return await this.db.select().from(this.schema.policies).where(eq(this.schema.policies.id, id)).limit(1).then(r => r[0]);
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async createPolicy(policy: InsertPolicy): Promise<Policy> {
    // Example: const [created] = await this.db.insert(this.schema.policies).values(policy).returning();
    // return created;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async updatePolicy(
    id: string,
    policy: Partial<InsertPolicy>
  ): Promise<Policy | undefined> {
    // Example: const [updated] = await this.db.update(this.schema.policies).set(policy).where(eq(this.schema.policies.id, id)).returning();
    // return updated;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async deletePolicy(id: string): Promise<boolean> {
    // Example: const result = await this.db.delete(this.schema.policies).where(eq(this.schema.policies.id, id));
    // return result.rowCount > 0;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async activatePolicy(id: string): Promise<boolean> {
    // Example: const [updated] = await this.db.update(this.schema.policies).set({ isActive: true }).where(eq(this.schema.policies.id, id)).returning();
    // return !!updated;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async deactivatePolicy(id: string): Promise<boolean> {
    // Example: const [updated] = await this.db.update(this.schema.policies).set({ isActive: false }).where(eq(this.schema.policies.id, id)).returning();
    // return !!updated;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getViolations(): Promise<Violation[]> {
    // Example: return await this.db.select().from(this.schema.violations).orderBy(desc(this.schema.violations.timestamp));
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getViolation(id: string): Promise<Violation | undefined> {
    // Example: return await this.db.select().from(this.schema.violations).where(eq(this.schema.violations.id, id)).limit(1).then(r => r[0]);
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async createViolation(violation: InsertViolation): Promise<Violation> {
    // Example: const [created] = await this.db.insert(this.schema.violations).values(violation).returning();
    // return created;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getViolationsByDateRange(
    startDate: Date,
    endDate: Date
  ): Promise<Violation[]> {
    // Example: return await this.db.select().from(this.schema.violations)
    //   .where(and(gte(this.schema.violations.timestamp, startDate), lte(this.schema.violations.timestamp, endDate)));
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getSystemMetrics(): Promise<SystemMetric[]> {
    // Example: return await this.db.select().from(this.schema.systemMetrics).orderBy(desc(this.schema.systemMetrics.timestamp));
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async createSystemMetric(metric: InsertSystemMetric): Promise<SystemMetric> {
    // Example: const [created] = await this.db.insert(this.schema.systemMetrics).values(metric).returning();
    // return created;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getDashboardMetrics(): Promise<DashboardMetrics> {
    // Example: Aggregate queries using Drizzle's sql`` template or count() functions
    // const totalPolicies = await this.db.select({ count: count() }).from(this.schema.policies);
    // const activePolicies = await this.db.select({ count: count() }).from(this.schema.policies).where(eq(this.schema.policies.isActive, true));
    // ... combine results
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getSystemHealth(): Promise<SystemHealth> {
    // Example: Query latest metrics and compute health status
    // const latestMetrics = await this.db.select().from(this.schema.systemMetrics).orderBy(desc(this.schema.systemMetrics.timestamp)).limit(1);
    // return { status: 'healthy', uptime: ..., ... };
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getRulePacks(): Promise<RulePack[]> {
    // Example: return await this.db.select().from(this.schema.rulePacks).orderBy(desc(this.schema.rulePacks.createdAt));
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getRulePack(id: string): Promise<RulePack | undefined> {
    // Example: return await this.db.select().from(this.schema.rulePacks).where(eq(this.schema.rulePacks.id, id)).limit(1).then(r => r[0]);
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async createRulePack(rulepack: InsertRulePack): Promise<RulePack> {
    // Example: const [created] = await this.db.insert(this.schema.rulePacks).values(rulepack).returning();
    // return created;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async updateRulePack(
    id: string,
    rulepack: Partial<InsertRulePack>
  ): Promise<RulePack | undefined> {
    // Example: const [updated] = await this.db.update(this.schema.rulePacks).set(rulepack).where(eq(this.schema.rulePacks.id, id)).returning();
    // return updated;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async getUser(id: string): Promise<User | undefined> {
    // Example: return await this.db.select().from(this.schema.users).where(eq(this.schema.users.id, id)).limit(1).then(r => r[0]);
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }

  async upsertUser(userData: UpsertUser): Promise<User> {
    // Example using Drizzle's onConflictDoUpdate:
    // const [user] = await this.db.insert(this.schema.users).values(userData)
    //   .onConflictDoUpdate({ target: this.schema.users.id, set: userData }).returning();
    // return user;
    throw new Error(
      "DatabaseStorage requires Drizzle DB instance - inject via constructor"
    );
  }
}

export const storage = new MemStorage();
