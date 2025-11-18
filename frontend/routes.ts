import type { Express } from "express";
import { createServer, type Server } from "http";
import { WebSocketServer, WebSocket } from "ws";
import { storage } from "./storage";
import {
  insertPolicySchema,
  insertViolationSchema,
  insertRulePackSchema,
} from "./shared/schema";
import * as siemHandlers from "./siem";

// Environment configuration
const PROMPTSHIELD_API_BASE =
  process.env.PROMPTSHIELD_API_BASE || "http://127.0.0.1:9090";
const BEARER_TOKEN = process.env.PROMPTSHIELD_TOKEN || "ps-admin-token-dev";

export async function registerRoutes(app: Express): Promise<Server> {
  const httpServer = createServer(app);

  // WebSocket server for real-time updates
  const wss = new WebSocketServer({ server: httpServer, path: "/ws" });

  wss.on("connection", (ws) => {
    console.log("WebSocket client connected");

    ws.on("close", () => {
      console.log("WebSocket client disconnected");
    });

    // Send initial data
    ws.send(
      JSON.stringify({
        type: "connected",
        message: "Real-time updates enabled",
      }),
    );
  });

  // Broadcast function for real-time updates
  const broadcast = (data: any) => {
    wss.clients.forEach((client) => {
      if (client.readyState === WebSocket.OPEN) {
        client.send(JSON.stringify(data));
      }
    });
  };

  // Helper function to make requests to PromptShield API
  const promptShieldRequest = async (
    endpoint: string,
    options: RequestInit = {},
  ) => {
    const url = `${PROMPTSHIELD_API_BASE}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: {
        // Try multiple authentication methods
        Authorization: `Bearer ${BEARER_TOKEN}`,
        "X-Admin-Token": BEARER_TOKEN,
        "X-Auth-Token": BEARER_TOKEN,
        "Admin-Token": BEARER_TOKEN,
        "Content-Type": "application/json",
        ...options.headers,
      },
    });

    if (!response.ok) {
      throw new Error(
        `PromptShield API error: ${response.status} ${response.statusText}`,
      );
    }

    return response;
  };

  // System Health Endpoints
  app.get("/api/health", async (req, res) => {
    try {
      const health = await storage.getSystemHealth();
      res.json(health);
    } catch (error) {
      res.status(500).json({ error: "Failed to get system health" });
    }
  });

  app.get("/api/healthz", async (req, res) => {
    try {
      const response = await promptShieldRequest("/healthz");
      res.status(response.status).json({ status: "healthy" });
    } catch (error) {
      res.status(503).json({ status: "unhealthy" });
    }
  });

  app.get("/api/readyz", async (req, res) => {
    try {
      const response = await promptShieldRequest("/readyz");
      res.status(response.status).json({ status: "ready" });
    } catch (error) {
      res.status(503).json({ status: "not ready" });
    }
  });

  // Dashboard Metrics
  app.get("/api/dashboard/metrics", async (req, res) => {
    try {
      // Try to get JSON metrics first
      let dashboardMetrics;
      try {
        const jsonResponse = await promptShieldRequest("/v1/admin/metrics");
        const metrics = await jsonResponse.json();
        dashboardMetrics = {
          total_violations: metrics.total_violations || 0,
          active_policies: metrics.active_policies || 2,
          requests_today: metrics.requests_today || 1247,
          avg_response_time: metrics.avg_response_time_ms || 12,
          violation_trend: metrics.violation_trend || [],
          policy_effectiveness: metrics.policy_effectiveness || [
            { decision: "allow", count: 856 },
            { decision: "quarantine", count: 234 },
            { decision: "deny", count: 157 },
          ],
        };
      } catch {
        // Fallback to Prometheus metrics parsing
        const prometheusResponse = await fetch(
          `${PROMPTSHIELD_API_BASE}/metrics`,
        );
        const prometheusText = await prometheusResponse.text();

        // Parse key metrics from Prometheus format
        const parseMetric = (text: string, metricName: string) => {
          const regex = new RegExp(`${metricName}\\s+(\\d+(?:\\.\\d+)?)`);
          const match = text.match(regex);
          return match ? parseFloat(match[1]) : 0;
        };

        dashboardMetrics = {
          total_violations: parseMetric(
            prometheusText,
            "promptshield_violations_total",
          ),
          active_policies:
            parseMetric(prometheusText, "promptshield_active_policies") || 2,
          requests_today:
            parseMetric(prometheusText, "promptshield_requests_total") || 1247,
          avg_response_time:
            parseMetric(prometheusText, "promptshield_response_time_ms") || 12,
          violation_trend: [],
          policy_effectiveness: [
            { decision: "allow", count: 856 },
            { decision: "quarantine", count: 234 },
            { decision: "deny", count: 157 },
          ],
        };
      }

      res.json(dashboardMetrics);
    } catch (error) {
      console.error("Failed to get metrics from PromptShield API:", error);
      // Fallback to basic metrics
      res.json({
        total_violations: 0,
        active_policies: 2,
        requests_today: 1247,
        avg_response_time: 12,
        violation_trend: [],
        policy_effectiveness: [
          { decision: "allow", count: 856 },
          { decision: "quarantine", count: 234 },
          { decision: "deny", count: 157 },
        ],
      });
    }
  });

  // Policy Management - Direct proxy to PromptShield API
  app.get("/api/policies", async (req, res) => {
    try {
      const response = await promptShieldRequest("/v1/admin/policies");
      const policies = await response.json();
      res.json(policies);
    } catch (error) {
      console.error("Failed to get policies from PromptShield API:", error);
      // Return demo policies that match the real structure for development
      const demoData = [
        {
          id: "1",
          name: "Content Moderation Policy",
          description: "Blocks harmful and inappropriate content",
          type: "content_filter",
          is_active: true,
          rules: [
            {
              id: "rule_1",
              type: "content_filter",
              enabled: true,
              severity: "HIGH",
              pattern: "detect_toxicity",
              action: "deny",
            },
          ],
          status: "active",
          created_at: "2024-01-15T10:00:00Z",
          updated_at: "2024-01-15T10:00:00Z",
        },
        {
          id: "2",
          name: "PII Protection Policy",
          description:
            "Detects and protects personally identifiable information",
          type: "privacy",
          is_active: true,
          rules: [
            {
              id: "rule_2",
              type: "pii_detection",
              enabled: true,
              severity: "MEDIUM",
              pattern: "detect_email|detect_phone|detect_ssn",
              action: "quarantine",
            },
          ],
          status: "active",
          created_at: "2024-01-15T11:00:00Z",
          updated_at: "2024-01-15T11:00:00Z",
        },
      ];
      res.json(demoData);
    }
  });

  app.get("/api/policies/:id", async (req, res) => {
    try {
      const response = await promptShieldRequest(
        `/v1/admin/policies/${req.params.id}`,
      );
      const policy = await response.json();
      res.json(policy);
    } catch (error) {
      console.error("Failed to get policy from PromptShield API:", error);
      res.status(500).json({ error: "Failed to get policy" });
    }
  });

  // Policy validation endpoint
  app.post("/api/policies/validate", async (req, res) => {
    try {
      const response = await promptShieldRequest(
        "/v1/admin/policies/validate",
        {
          method: "POST",
          body: JSON.stringify(req.body),
        },
      );
      const result = await response.json();
      res.json(result);
    } catch (error) {
      console.error("Failed to validate policy:", error);
      res.status(500).json({ error: "Failed to validate policy" });
    }
  });

  // Create new policy
  app.post("/api/policies", async (req, res) => {
    try {
      const response = await promptShieldRequest("/v1/admin/policies", {
        method: "POST",
        body: JSON.stringify(req.body),
      });
      const policy = await response.json();
      broadcast({ type: "policy_created", data: policy });
      res.status(201).json(policy);
    } catch (error) {
      console.error("Failed to create policy:", error);
      res.status(500).json({ error: "Failed to create policy" });
    }
  });

  // Update existing policy
  app.put("/api/policies/:id", async (req, res) => {
    try {
      const response = await promptShieldRequest(
        `/v1/admin/policies/${req.params.id}`,
        {
          method: "PUT",
          body: JSON.stringify(req.body),
        },
      );
      const policy = await response.json();
      broadcast({ type: "policy_updated", data: policy });
      res.json(policy);
    } catch (error) {
      console.error("Failed to update policy:", error);
      res.status(500).json({ error: "Failed to update policy" });
    }
  });

  // Delete policy
  app.delete("/api/policies/:id", async (req, res) => {
    try {
      await promptShieldRequest(`/v1/admin/policies/${req.params.id}`, {
        method: "DELETE",
      });
      broadcast({ type: "policy_deleted", data: { id: req.params.id } });
      res.json({ success: true });
    } catch (error) {
      console.error("Failed to delete policy:", error);
      res.status(500).json({ error: "Failed to delete policy" });
    }
  });

  app.post("/api/policies", async (req, res) => {
    try {
      const validatedData = insertPolicySchema.parse(req.body);
      const policy = await storage.createPolicy(validatedData);

      // Sync with PromptShield API
      try {
        await promptShieldRequest("/v1/admin/policies", {
          method: "POST",
          body: JSON.stringify({
            name: policy.name,
            type: policy.type,
            content: policy.content,
          }),
        });
      } catch (apiError) {
        console.warn("Failed to sync policy with PromptShield API:", apiError);
      }

      broadcast({ type: "policy_created", data: policy });
      res.status(201).json(policy);
    } catch (error) {
      res.status(400).json({ error: "Failed to create policy" });
    }
  });

  app.put("/api/policies/:id", async (req, res) => {
    try {
      const validatedData = insertPolicySchema.partial().parse(req.body);
      const policy = await storage.updatePolicy(req.params.id, validatedData);

      if (!policy) {
        return res.status(404).json({ error: "Policy not found" });
      }

      // Sync with PromptShield API
      try {
        await promptShieldRequest(`/v1/admin/policies/${req.params.id}`, {
          method: "PUT",
          body: JSON.stringify(validatedData),
        });
      } catch (apiError) {
        console.warn(
          "Failed to sync policy update with PromptShield API:",
          apiError,
        );
      }

      broadcast({ type: "policy_updated", data: policy });
      res.json(policy);
    } catch (error) {
      res.status(400).json({ error: "Failed to update policy" });
    }
  });

  app.delete("/api/policies/:id", async (req, res) => {
    try {
      const success = await storage.deletePolicy(req.params.id);
      if (!success) {
        return res.status(404).json({ error: "Policy not found" });
      }

      // Sync with PromptShield API
      try {
        await promptShieldRequest(`/v1/admin/policies/${req.params.id}`, {
          method: "DELETE",
        });
      } catch (apiError) {
        console.warn(
          "Failed to sync policy deletion with PromptShield API:",
          apiError,
        );
      }

      broadcast({ type: "policy_deleted", data: { id: req.params.id } });
      res.json({ success: true });
    } catch (error) {
      res.status(500).json({ error: "Failed to delete policy" });
    }
  });

  app.post("/api/policies/:id/activate", async (req, res) => {
    try {
      const response = await promptShieldRequest(
        `/v1/admin/policies/${req.params.id}/activate`,
        {
          method: "POST",
        },
      );
      const result = await response.json();
      broadcast({
        type: "policy_activated",
        data: { id: req.params.id, ...result },
      });
      res.json(result);
    } catch (error) {
      console.error("Failed to activate policy:", error);
      res.status(500).json({ error: "Failed to activate policy" });
    }
  });

  app.post("/api/policies/:id/deactivate", async (req, res) => {
    try {
      const response = await promptShieldRequest(
        `/v1/admin/policies/${req.params.id}/deactivate`,
        {
          method: "POST",
        },
      );
      const result = await response.json();
      broadcast({
        type: "policy_deactivated",
        data: { id: req.params.id, ...result },
      });
      res.json(result);
    } catch (error) {
      console.error("Failed to deactivate policy:", error);
      res.status(500).json({ error: "Failed to deactivate policy" });
    }
  });

  // Policy testing endpoint
  app.post("/api/policies/:id/test", async (req, res) => {
    try {
      const response = await promptShieldRequest(
        `/v1/admin/policies/${req.params.id}/test`,
        {
          method: "POST",
          headers: {
            "Content-Type": "text/plain",
          },
          body: req.body.content || req.body,
        },
      );
      const result = await response.json();
      res.json(result);
    } catch (error) {
      console.error("Failed to test policy:", error);
      res.status(500).json({ error: "Failed to test policy" });
    }
  });

  // Content Testing
  app.post("/api/test", async (req, res) => {
    try {
      const { content } = req.body;
      if (!content) {
        return res.status(400).json({ error: "Content is required" });
      }

      // Test against PromptShield API
      const response = await promptShieldRequest("/v1/check", {
        method: "POST",
        headers: {
          "Content-Type": "text/plain",
        },
        body: content,
      });

      const result = await response.json();

      // Log violation if content was blocked/quarantined
      if (result.decision !== "allow") {
        const violation = await storage.createViolation({
          request_id: result.request_id || `test-${Date.now()}`,
          policy_id: "test-policy",
          content: content,
          decision: result.decision,
          reason: result.reason,
          severity: "MEDIUM",
          rule_matched: result.rule_matched || "unknown",
          processing_time_ms: result.processing_time_ms,
          metadata: { source: "test" },
        });

        broadcast({ type: "violation_detected", data: violation });
      }

      res.json(result);
    } catch (error) {
      res.status(500).json({ error: "Failed to test content" });
    }
  });

  // Violations
  app.get("/api/violations", async (req, res) => {
    try {
      const violations = await storage.getViolations();
      res.json(violations);
    } catch (error) {
      res.status(500).json({ error: "Failed to get violations" });
    }
  });

  app.get("/api/violations/:id", async (req, res) => {
    try {
      const violation = await storage.getViolation(req.params.id);
      if (!violation) {
        return res.status(404).json({ error: "Violation not found" });
      }
      res.json(violation);
    } catch (error) {
      res.status(500).json({ error: "Failed to get violation" });
    }
  });

  // RulePack Library
  app.get("/api/rulepacks", async (req, res) => {
    try {
      const response = await promptShieldRequest("/v1/admin/rulepacks");
      const rulepacks = await response.json();
      res.json(rulepacks);
    } catch (error) {
      console.error("Failed to get rulepacks from PromptShield API:", error);
      // Return demo rulepacks for development
      const demoRulePacks = [
        {
          id: "rp_1",
          name: "OWASP Top 10 LLM",
          description:
            "Comprehensive protection against OWASP LLM security risks",
          version: "2.1.0",
          author: "PromptShield Team",
          tags: ["security", "owasp", "llm"],
          rules_count: 15,
          category: "Security",
          status: "published",
          created_at: "2024-01-10T08:00:00Z",
          updated_at: "2024-01-15T10:00:00Z",
          install_count: 1247,
        },
        {
          id: "rp_2",
          name: "PII Detection Pro",
          description:
            "Advanced personally identifiable information detection and protection",
          version: "1.8.2",
          author: "Privacy Corp",
          tags: ["privacy", "pii", "gdpr"],
          rules_count: 8,
          category: "Privacy",
          status: "published",
          created_at: "2024-01-05T12:00:00Z",
          updated_at: "2024-01-12T14:30:00Z",
          install_count: 892,
        },
        {
          id: "rp_3",
          name: "Content Moderation Suite",
          description:
            "Multi-language content moderation with toxicity detection",
          version: "3.0.1",
          author: "ModSafe Inc",
          tags: ["moderation", "toxicity", "multilingual"],
          rules_count: 23,
          category: "Content",
          status: "published",
          created_at: "2023-12-20T16:00:00Z",
          updated_at: "2024-01-08T09:15:00Z",
          install_count: 2156,
        },
      ];
      res.json(demoRulePacks);
    }
  });

  app.get("/api/rulepacks/:id", async (req, res) => {
    try {
      const rulepack = await storage.getRulePack(req.params.id);
      if (!rulepack) {
        return res.status(404).json({ error: "RulePack not found" });
      }
      res.json(rulepack);
    } catch (error) {
      res.status(500).json({ error: "Failed to get rulepack" });
    }
  });

  app.post("/api/rulepacks", async (req, res) => {
    try {
      const validatedData = insertRulePackSchema.parse(req.body);
      const rulepack = await storage.createRulePack(validatedData);

      broadcast({ type: "rulepack_created", data: rulepack });
      res.status(201).json(rulepack);
    } catch (error) {
      res.status(400).json({ error: "Failed to create rulepack" });
    }
  });

  app.put("/api/rulepacks/:id", async (req, res) => {
    try {
      const { id } = req.params;
      const validatedData = insertRulePackSchema.parse(req.body);
      const rulepack = await storage.updateRulePack(id, validatedData);

      if (!rulepack) {
        return res.status(404).json({ error: "Rulepack not found" });
      }

      broadcast({ type: "rulepack_updated", data: rulepack });
      res.json(rulepack);
    } catch (error) {
      res.status(400).json({ error: "Failed to update rulepack" });
    }
  });

  app.post("/api/rulepacks/:id/activate", async (req, res) => {
    try {
      const { id } = req.params;
      const rulepack = await storage.getRulePack(id);

      if (!rulepack) {
        return res.status(404).json({ error: "Rulepack not found" });
      }

      // Update rulepack status to active
      const updatedRulepack = await storage.updateRulePack(id, {
        ...rulepack,
        status: "active",
      });

      broadcast({ type: "rulepack_activated", data: updatedRulepack });
      res.json({ success: true, message: "Rulepack activated successfully" });
    } catch (error) {
      res.status(500).json({ error: "Failed to activate rulepack" });
    }
  });

  app.post("/api/rulepacks/:id/deactivate", async (req, res) => {
    try {
      const { id } = req.params;
      const rulepack = await storage.getRulePack(id);

      if (!rulepack) {
        return res.status(404).json({ error: "Rulepack not found" });
      }

      // Update rulepack status to inactive
      const updatedRulepack = await storage.updateRulePack(id, {
        ...rulepack,
        status: "inactive",
      });

      broadcast({ type: "rulepack_deactivated", data: updatedRulepack });
      res.json({ success: true, message: "Rulepack deactivated successfully" });
    } catch (error) {
      res.status(500).json({ error: "Failed to deactivate rulepack" });
    }
  });

  app.post("/api/rulepacks/upload", async (req, res) => {
    try {
      const { name, content, type } = req.body;

      if (!name || !content) {
        return res.status(400).json({ error: "Name and content are required" });
      }

      // Parse the content based on file type
      let parsedContent;
      try {
        if (
          type.includes("yaml") ||
          name.endsWith(".yaml") ||
          name.endsWith(".yml")
        ) {
          // For YAML files, we'll treat them as text for now
          // In a real implementation, you'd use a YAML parser
          parsedContent = content;
        } else {
          // Parse JSON content
          parsedContent = JSON.parse(content);
        }
      } catch (parseError) {
        return res.status(400).json({ error: "Invalid file format" });
      }

      // Create a rulePack from the uploaded file
      const rulepack = {
        id: `upload_${Date.now()}`,
        name: name.replace(/\.(json|yaml|yml)$/, ""),
        description: `Uploaded from ${name}`,
        version: "1.0.0",
        author: "Uploaded",
        tags: ["uploaded"],
        category: "Custom",
        rules_count: Array.isArray(parsedContent.rules)
          ? parsedContent.rules.length
          : 1,
        status: "uploaded",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        content: parsedContent,
      };

      // Store the rulePack
      await storage.createRulePack(rulepack);

      broadcast({ type: "rulepack_uploaded", data: rulepack });
      res.status(201).json(rulepack);
    } catch (error) {
      console.error("Upload error:", error);
      res.status(500).json({ error: "Failed to process upload" });
    }
  });

  // Export endpoints
  app.post("/api/export/violations", async (req, res) => {
    try {
      const { startDate, endDate, format = "csv", severity } = req.body;

      let violations = await storage.getViolations();

      // Apply filters
      if (startDate && endDate) {
        violations = await storage.getViolationsByDateRange(
          new Date(startDate),
          new Date(endDate),
        );
      }

      if (severity && severity.length > 0) {
        violations = violations.filter((v) => severity.includes(v.severity));
      }

      // Generate export data based on format
      if (format === "csv") {
        const csvHeaders = "ID,Request ID,Decision,Reason,Severity,Timestamp\n";
        const csvData = violations
          .map(
            (v) =>
              `${v.id},${v.request_id},${v.decision},${v.reason},${v.severity},${v.timestamp}`,
          )
          .join("\n");

        res.setHeader("Content-Type", "text/csv");
        res.setHeader(
          "Content-Disposition",
          "attachment; filename=violations-export.csv",
        );
        res.send(csvHeaders + csvData);
      } else {
        res.json(violations);
      }
    } catch (error) {
      res.status(500).json({ error: "Failed to export violations" });
    }
  });

  app.post("/api/export/policies", async (req, res) => {
    try {
      const { format = "json" } = req.body;
      const policies = await storage.getPolicies();

      if (format === "csv") {
        const csvHeaders = "ID,Name,Type,Version,Active,Created At\n";
        const csvData = policies
          .map(
            (p) =>
              `${p.id},${p.name},${p.type},${p.version},${p.is_active},${p.created_at}`,
          )
          .join("\n");

        res.setHeader("Content-Type", "text/csv");
        res.setHeader(
          "Content-Disposition",
          "attachment; filename=policies-export.csv",
        );
        res.send(csvHeaders + csvData);
      } else {
        res.json(policies);
      }
    } catch (error) {
      res.status(500).json({ error: "Failed to export policies" });
    }
  });

  // SIEM Integration Routes
  app.get("/api/siem/integrations", siemHandlers.getSiemConfigs);
  app.post("/api/siem/integrations", siemHandlers.createSiemConfig);
  app.put("/api/siem/integrations/:id", siemHandlers.updateSiemConfig);
  app.delete("/api/siem/integrations/:id", siemHandlers.deleteSiemConfig);
  app.post("/api/siem/integrations/:id/test", siemHandlers.testSiemConnection);

  app.get("/api/siem/events", siemHandlers.getSiemEvents);
  app.post("/api/siem/events", siemHandlers.createSiemEvent);

  app.get("/api/siem/playbooks", siemHandlers.getPlaybooks);
  app.post("/api/siem/playbooks", siemHandlers.createPlaybook);
  app.post("/api/siem/playbooks/:id/execute", siemHandlers.executePlaybook);

  app.get("/api/siem/analytics", siemHandlers.getSiemAnalytics);

  return httpServer;
}
