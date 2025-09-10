import { ReactNode } from 'react';
import { Control, FieldValues } from 'react-hook-form';
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

type Option = { value: string; label: ReactNode };

type BaseProps<T extends FieldValues = any> = {
  control: Control<T>;
  name: string;
  label: ReactNode;
  options: Option[];
  placeholder?: ReactNode;
  className?: string;
  'data-testid'?: string;
};

export function SelectField<T extends FieldValues = any>({ control, name, label, options, placeholder, className, ...rest }: BaseProps<T>) {
  return (
    <FormField
      control={control}
      name={name as any}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Select value={field.value} onValueChange={field.onChange}>
              <SelectTrigger data-testid={rest['data-testid']}>
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
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

