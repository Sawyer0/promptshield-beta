import { useState } from 'react';
import { Button } from '@/components/ui/button';

interface JSONViewerProps {
  data: any;
  initialCollapsed?: boolean;
  className?: string;
}

export function JSONViewer({ data, initialCollapsed = false, className }: JSONViewerProps) {
  const [collapsed, setCollapsed] = useState(initialCollapsed);

  const pretty = JSON.stringify(data, null, 2);
  const compact = JSON.stringify(data);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(pretty);
    } catch {}
  };

  return (
    <div className={className}>
      <div className="flex items-center justify-end gap-2 mb-2">
        <Button variant="outline" size="sm" onClick={() => setCollapsed(!collapsed)}>
          {collapsed ? 'Pretty' : 'Compact'}
        </Button>
        <Button variant="outline" size="sm" onClick={handleCopy}>Copy JSON</Button>
      </div>
      <pre className="bg-muted rounded-md p-3 overflow-auto text-xs max-h-[320px]"><code>{collapsed ? compact : pretty}</code></pre>
    </div>
  );
}

