import { ReactNode } from 'react';
import { Button } from '@/components/ui/button';

export interface QuickFilterOption {
  key: string;
  label: ReactNode;
}

interface QuickFilterChipsProps {
  value: string | null;
  onChange: (value: string | null) => void;
  options: QuickFilterOption[];
}

export function QuickFilterChips({ value, onChange, options }: QuickFilterChipsProps) {
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((opt) => (
        <Button
          key={opt.key}
          variant={value === opt.key ? 'default' : 'outline'}
          size="sm"
          onClick={() => onChange(value === opt.key ? null : opt.key)}
        >
          {opt.label}
        </Button>
      ))}
    </div>
  );
}

