import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';

interface Kpi {
  label: string;
  value?: string | number;
  hint?: string;
  icon?: React.ReactNode;
  loading?: boolean;
}

export function AdminKpis({ items }: { items: Kpi[] }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      {items.map((k, i) => (
        <Card key={i} className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground mb-1">{k.label}</p>
              {k.loading ? (
                <Skeleton className="h-6 w-16" />
              ) : (
                <p className="text-xl font-bold">{k.value ?? '—'}</p>
              )}
              {k.hint && <p className="text-xs text-muted-foreground">{k.hint}</p>}
            </div>
            {k.icon}
          </div>
        </Card>
      ))}
    </div>
  );
}
