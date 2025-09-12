import { ArrowRight } from "lucide-react";

export function Privacy() {
  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-3xl px-4 sm:px-6 py-12">
        <h1 className="text-3xl font-semibold serif-display">Privacy Policy</h1>
        <p className="mt-3 text-muted-foreground">
          We respect your privacy. This page outlines how we handle data you submit via the waitlist form and while using this site.
        </p>

        <div className="mt-8 space-y-6 text-sm leading-6">
          <section>
            <h2 className="text-lg font-semibold">Waitlist data</h2>
            <p>
              If you join the waitlist, we collect the fields you submit (email, company, role, company size, LLM provider,
              frameworks, and top risk concern). We use this information to prioritize access and to contact you about
              onboarding and relevant product updates.
            </p>
          </section>
          <section>
            <h2 className="text-lg font-semibold">Storage and retention</h2>
            <p>
              During early access, submissions are stored securely within our infrastructure and retained only as long as necessary
              for onboarding and product communication. You can request deletion at any time.
            </p>
          </section>
          <section>
            <h2 className="text-lg font-semibold">Prompt demo</h2>
            <p>
              The on-page Prompt Risk Demo runs client-side only and does not store inputs on our servers.
              Do not paste real secrets or personal data.
            </p>
          </section>
          <section>
            <h2 className="text-lg font-semibold">Contact</h2>
            <p>
              Questions or requests (including data deletion): contact us at privacy@promptshield.dev.
            </p>
          </section>
        </div>

        <a href="/" className="inline-flex items-center gap-2 mt-10 text-primary hover:underline">
          <ArrowRight className="h-4 w-4" /> Back to home
        </a>
      </div>
    </div>
  );
}

