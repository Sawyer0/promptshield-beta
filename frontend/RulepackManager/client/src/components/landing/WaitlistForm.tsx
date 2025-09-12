import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";
import { useTrack } from "@/hooks/useTrack";

export function WaitlistForm({ inline }: { inline?: boolean }) {
  const { toast } = useToast();
  const track = useTrack();
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);

  const [form, setForm] = useState({
    email: "",
    role: "",
    company: "",
    size: "",
    provider: "",
    frameworks: [] as string[],
    risk: "",
    agree: false,
  });

  async function submit() {
    if (!form.email || !form.company || !form.agree) {
      toast({ title: "Missing required fields", description: "Email, company, and agreement are required." });
      return;
    }
    setSubmitting(true);
    try {
      const res = await fetch('/api/waitlist', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, ts: new Date().toISOString() }),
      });
      if (!res.ok) throw new Error(await res.text());
      setSuccess(true);
      track("FormCompleted");
    } catch (e: any) {
      toast({ title: "Submission failed", description: e?.message || "Please try again." });
    } finally {
      setSubmitting(false);
    }
  }

  const toggleFramework = (v: string) => {
    setForm((f) => {
      const has = f.frameworks.includes(v);
      return { ...f, frameworks: has ? f.frameworks.filter((x) => x !== v) : [...f.frameworks, v] };
    });
  };

  if (success) {
    return (
      <div className="rounded-xl border bg-card p-6 shadow-sm">
        <h3 className="text-lg font-semibold">You’re on the list.</h3>
        <p className="text-sm text-muted-foreground mt-1">We’ll email early access and our LLM Security & Compliance Checklist.</p>
        <div className="mt-4">
          <Button variant="outline" className="gap-2" onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}>
            Back to top
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className={inline ? "" : "rounded-xl border bg-card p-6 shadow-sm"}>
      {!inline && <h3 className="text-lg font-semibold mb-4">Join the waitlist</h3>}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <Input placeholder="Work email" type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} aria-label="Work email" />
        <Input placeholder="Company" value={form.company} onChange={(e) => setForm({ ...form, company: e.target.value })} aria-label="Company" />
        <Select value={form.role} onValueChange={(v) => setForm({ ...form, role: v })}>
          <SelectTrigger><SelectValue placeholder="Role" /></SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="engineering">Engineering</SelectItem>
              <SelectItem value="appsec">AppSec/SecOps</SelectItem>
              <SelectItem value="compliance">Compliance/Privacy</SelectItem>
              <SelectItem value="product">Product</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
        <Select value={form.size} onValueChange={(v) => setForm({ ...form, size: v })}>
          <SelectTrigger><SelectValue placeholder="Company size" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="1-10">1–10</SelectItem>
            <SelectItem value="11-50">11–50</SelectItem>
            <SelectItem value="51-200">51–200</SelectItem>
            <SelectItem value="201-1000">201–1k</SelectItem>
            <SelectItem value="1000+">1k+</SelectItem>
          </SelectContent>
        </Select>
        <Select value={form.provider} onValueChange={(v) => setForm({ ...form, provider: v })}>
          <SelectTrigger><SelectValue placeholder="Primary LLM provider" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="openai">OpenAI/Azure</SelectItem>
            <SelectItem value="bedrock">AWS Bedrock</SelectItem>
            <SelectItem value="vertex">Google Vertex</SelectItem>
            <SelectItem value="anthropic">Anthropic</SelectItem>
            <SelectItem value="self-hosted">Self‑hosted</SelectItem>
          </SelectContent>
        </Select>
        <div className="sm:col-span-2">
          <div className="text-sm font-medium mb-2">Frameworks</div>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
            {[
              { id: "langchain", label: "LangChain" },
              { id: "llamaindex", label: "LlamaIndex" },
              { id: "openai-sdk", label: "OpenAI SDK" },
              { id: "others", label: "Others" },
            ].map((fw) => (
              <label key={fw.id} className="flex items-center gap-2 text-sm">
                <Checkbox checked={form.frameworks.includes(fw.id)} onCheckedChange={() => toggleFramework(fw.id)} />
                {fw.label}
              </label>
            ))}
          </div>
        </div>
        <Select value={form.risk} onValueChange={(v) => setForm({ ...form, risk: v })}>
          <SelectTrigger className="sm:col-span-2"><SelectValue placeholder="Top risk concern" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="injection">Injection</SelectItem>
            <SelectItem value="pii">PII/Secrets</SelectItem>
            <SelectItem value="tool-abuse">Tool/Agent Abuse</SelectItem>
            <SelectItem value="evidence">Compliance Evidence</SelectItem>
          </SelectContent>
        </Select>
        <label className="flex items-center gap-2 text-sm sm:col-span-2">
          <Checkbox checked={form.agree} onCheckedChange={(v) => setForm({ ...form, agree: Boolean(v) })} />
          I agree to the <a className="underline" href="/privacy" target="_self" rel="noopener noreferrer">Privacy Policy</a>
        </label>
        <div className="sm:col-span-2 flex gap-2">
          <Button onClick={() => { track("HeroWaitlistSubmit"); submit(); }} disabled={submitting} className="gap-2" style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }}>
            {submitting ? "Submitting…" : "Join the waitlist"}
          </Button>
        </div>
      </div>
    </div>
  );
}

