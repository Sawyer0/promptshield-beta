import type { Request, Response } from "express";

// Simple in-memory demo types for SIEM entities
export type SiemIntegration = {
  id: string;
  name: string;
  type: string;
  enabled: boolean;
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type SiemEvent = {
  id: string;
  source: string;
  severity: string;
  category: string;
  message: string;
  timestamp: string;
  metadata?: Record<string, unknown>;
};

export type Playbook = {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
  conditions?: Record<string, unknown>;
  actions?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

// In-memory stores (demo only)
const siemIntegrations: SiemIntegration[] = [];
const siemEvents: SiemEvent[] = [];
const playbooks: Playbook[] = [];

function generateId(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

// SIEM Integrations
export function getSiemConfigs(req: Request, res: Response): void {
  res.json(siemIntegrations);
}

export function createSiemConfig(req: Request, res: Response): void {
  const now = new Date().toISOString();
  const integration: SiemIntegration = {
    id: generateId("siem"),
    name: (req.body && req.body.name) || "New Integration",
    type: (req.body && req.body.type) || "generic",
    enabled: req.body?.enabled ?? true,
    config: req.body?.config ?? {},
    created_at: now,
    updated_at: now,
  };

  siemIntegrations.push(integration);
  res.status(201).json(integration);
}

export function updateSiemConfig(req: Request, res: Response): void {
  const { id } = req.params;
  const index = siemIntegrations.findIndex((i) => i.id === id);

  if (index === -1) {
    res.status(404).json({ error: "SIEM integration not found" });
    return;
  }

  const existing = siemIntegrations[index];
  const updated: SiemIntegration = {
    ...existing,
    ...req.body,
    updated_at: new Date().toISOString(),
  };

  siemIntegrations[index] = updated;
  res.json(updated);
}

export function deleteSiemConfig(req: Request, res: Response): void {
  const { id } = req.params;
  const index = siemIntegrations.findIndex((i) => i.id === id);

  if (index === -1) {
    res.status(404).json({ error: "SIEM integration not found" });
    return;
  }

  siemIntegrations.splice(index, 1);
  res.json({ success: true });
}

export function testSiemConnection(req: Request, res: Response): void {
  const { id } = req.params;
  const integration = siemIntegrations.find((i) => i.id === id);

  if (!integration) {
    res.status(404).json({ error: "SIEM integration not found" });
    return;
  }

  // Demo: always succeed
  res.json({
    success: true,
    message: `Tested SIEM integration '${integration.name}' successfully`,
  });
}

// SIEM Events
export function getSiemEvents(req: Request, res: Response): void {
  res.json(siemEvents);
}

export function createSiemEvent(req: Request, res: Response): void {
  const now = new Date().toISOString();
  const event: SiemEvent = {
    id: generateId("event"),
    source: req.body?.source ?? "promptshield",
    severity: req.body?.severity ?? "INFO",
    category: req.body?.category ?? "generic",
    message: req.body?.message ?? "",
    timestamp: now,
    metadata: req.body?.metadata ?? {},
  };

  siemEvents.push(event);
  res.status(201).json(event);
}

// Playbooks
export function getPlaybooks(req: Request, res: Response): void {
  res.json(playbooks);
}

export function createPlaybook(req: Request, res: Response): void {
  const now = new Date().toISOString();
  const playbook: Playbook = {
    id: generateId("playbook"),
    name: req.body?.name ?? "New Playbook",
    description: req.body?.description ?? "",
    enabled: req.body?.enabled ?? true,
    conditions: req.body?.conditions ?? {},
    actions: req.body?.actions ?? {},
    created_at: now,
    updated_at: now,
  };

  playbooks.push(playbook);
  res.status(201).json(playbook);
}

export function executePlaybook(req: Request, res: Response): void {
  const { id } = req.params;
  const playbook = playbooks.find((p) => p.id === id);

  if (!playbook) {
    res.status(404).json({ error: "Playbook not found" });
    return;
  }

  // Demo: execution is a no-op that returns a summary
  res.json({
    success: true,
    playbook_id: id,
    message: `Playbook '${playbook.name}' executed successfully`,
  });
}

// Analytics
export function getSiemAnalytics(req: Request, res: Response): void {
  const totalEvents = siemEvents.length;
  const bySeverity: Record<string, number> = {};

  for (const evt of siemEvents) {
    bySeverity[evt.severity] = (bySeverity[evt.severity] ?? 0) + 1;
  }

  res.json({
    total_events: totalEvents,
    events_by_severity: bySeverity,
  });
}
