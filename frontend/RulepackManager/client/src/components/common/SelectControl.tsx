import { ReactNode } from 'react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

type Option = { value: string; label: ReactNode };

interface SelectControlProps {
  value: string;
  onChange: (value: string) => void;
  options: Option[];
  placeholder?: ReactNode;
  className?: string;
  triggerClassName?: string;
  'data-testid'?: string;
}

export function SelectControl({ value, onChange, options, placeholder, className, triggerClassName, ...rest }: SelectControlProps) {
  return (
    <div className={className}>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className={triggerClassName} data-testid={rest['data-testid']}>
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {options.map((opt) => (
            <SelectItem key={String(opt.value)} value={String(opt.value)}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

