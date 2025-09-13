export function MarketingFooter() {
  return (
    <footer className="border-t mt-16">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 py-12 text-sm text-muted-foreground">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          <div>© {new Date().getFullYear()} PoliSync Guard</div>
          <div className="flex items-center gap-4">
            <a href="/trust" className="hover:underline">Security</a>
            <a href="/solutions/compliance" className="hover:underline">Compliance</a>
            <a href="/privacy" className="hover:underline">Privacy</a>
            <a href="#" className="hover:underline">Terms</a>
            <a href="#" className="hover:underline">Contact</a>
          </div>
        </div>
      </div>
    </footer>
  );
}

