declare module "@shared/schema" {
  // Lightweight ambient types for the shared schema used by the frontend.
  // These are intentionally broad so the frontend can compile in this repo
  // without depending on the original shared package.

  export type Policy = any;
  export type InsertPolicy = any;
  export type Violation = any;
  export type InsertViolation = any;
  export type SystemMetric = any;
  export type InsertSystemMetric = any;
  export type RulePack = any;
  export type InsertRulePack = any;
  export type DashboardMetrics = any;
  export type SystemHealth = any;
  export type UpsertUser = any;
  export type User = any;

  export const insertPolicySchema: any;
  export const insertViolationSchema: any;
  export const insertRulePackSchema: any;
}
