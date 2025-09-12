import { Shield, AlertTriangle, BrainCircuit, Lock, Zap, BarChart3 } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";

export function FeatureGrid() {
  const items = [
    { icon: Shield, title: "Injection & jailbreak prevention", desc: "Detect and stop malicious prompts before they reach your model." },
    { icon: AlertTriangle, title: "PII & secrets protection", desc: "Real-time detection and redaction to prevent data leakage." },
    { icon: BrainCircuit, title: "Policy-as-code guardrails", desc: "Versioned policies with CI checks, environments, and exceptions." },
    { icon: Lock, title: "Compliance evidence", desc: "Immutable logs, control mappings, and exportable reports." },
    { icon: BarChart3, title: "Observability", desc: "Incident dashboards, risk trends, and posture overview." },
    { icon: Zap, title: "Developer friendly", desc: "Works with your stack; minimal code changes." },
  ];
  return (
    <section className="mx-auto max-w-6xl px-4 sm:px-6 py-12">
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {items.map(({ icon: Icon, title, desc }) => (
          <Card key={title} className="card-hover">
            <CardContent className="p-6">
              <Icon className="h-6 w-6 mb-4" style={{ color: "var(--brand-accent)" }} />
              <h3 className="font-semibold mb-1">{title}</h3>
              <p className="text-sm text-muted-foreground">{desc}</p>
            </CardContent>
          </Card>
        ))}
      </div>
    </section>
  );
}

