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
    <section className="mx-auto max-w-[1600px] 2xl:max-w-[1760px] px-8 sm:px-12 xl:px-16 py-24">
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-10">
        {items.map(({ icon: Icon, title, desc }) => (
          <Card key={title} className="card-hover">
            <CardContent className="p-10">
              <Icon className="h-8 w-8 mb-6" style={{ color: "var(--brand-accent)" }} />
              <h3 className="font-semibold mb-3 text-lg">{title}</h3>
              <p className="text-base text-muted-foreground leading-relaxed">{desc}</p>
            </CardContent>
          </Card>
        ))}
      </div>
    </section>
  );
}

