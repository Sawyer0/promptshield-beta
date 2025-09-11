import { ReactNode } from 'react';
import { Control, FieldValues } from 'react-hook-form';
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Checkbox } from '@/components/ui/checkbox'
import { Slider } from '@/components/ui/slider'
import { Button } from '@/components/ui/button'

type BaseProps<T extends FieldValues = any> = {
  control: Control<T>;
  name: string;
  label: ReactNode;
  description?: ReactNode;
  className?: string;
  'data-testid'?: string;
};

export function TextField<T extends FieldValues = any>({ control, name, label, description, className, ...rest }: BaseProps<T> & { placeholder?: string; type?: string }) {
  return (
    <FormField
      control={control}
      name={name as any}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Input {...field} {...rest} />
          </FormControl>
          {description ? <div className="text-xs text-muted-foreground">{description}</div> : null}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

export function NumberField<T extends FieldValues = any>({ control, name, label, description, className, ...rest }: BaseProps<T> & { min?: number; max?: number; step?: number }) {
  return (
    <FormField
      control={control}
      name={name as any}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Input
              type="number"
              {...rest}
              value={field.value}
              onChange={(e) => {
                const v = e.target.value;
                const n = v === '' ? '' : Number(v);
                // Pass number when valid, otherwise empty string to keep RHF happy
                field.onChange(Number.isFinite(n as number) ? n : '');
              }}
            />
          </FormControl>
          {description ? <div className="text-xs text-muted-foreground">{description}</div> : null}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

export function TextAreaField<T extends FieldValues = any>({ control, name, label, description, className, ...rest }: BaseProps<T> & { rows?: number; placeholder?: string }) {
  return (
    <FormField
      control={control}
      name={name as any}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Textarea {...field} {...rest} />
          </FormControl>
          {description ? <div className="text-xs text-muted-foreground">{description}</div> : null}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

export function ToggleField<T extends FieldValues = any>({ control, name, label, description, className }: BaseProps<T>) {
  return (
    <FormField
      control={control}
      name={name as any}
      render={({ field }) => (
        <FormItem className={className + ' flex flex-row items-center justify-between rounded-lg border p-4'}>
          <div className="space-y-0.5">
            <FormLabel className="text-base">{label}</FormLabel>
            {description ? <div className="text-sm text-muted-foreground">{description}</div> : null}
          </div>
          <FormControl>
            <Switch checked={!!field.value} onCheckedChange={field.onChange} />
          </FormControl>
        </FormItem>
      )}
    />
  );
}

export function MultiSelectChips<T extends FieldValues = any>({ control, name, label, description, className, options }: BaseProps<T> & { options: { value: string; label: ReactNode }[] }) {
  return (
    <FormField
      control={control}
      name={name as any}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>{label}</FormLabel>
          {description ? <div className="text-xs text-muted-foreground mb-2">{description}</div> : null}
          <div className="flex flex-wrap gap-2">
            {options.map((opt) => {
              const selected: string[] = Array.isArray(field.value) ? field.value : []
              const checked = selected.includes(opt.value)
              return (
                <Button
                  key={opt.value}
                  type="button"
                  variant={checked ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => {
                    const next = checked ? selected.filter(v => v !== opt.value) : [...selected, opt.value]
                    field.onChange(next)
                  }}
                >
                  {opt.label}
                </Button>
              )
            })}
          </div>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

export function SliderField<T extends FieldValues = any>({ control, name, label, description, className, min = 0, max = 1, step = 0.01 }: BaseProps<T> & { min?: number; max?: number; step?: number }) {
  return (
    <FormField
      control={control}
      name={name as any}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <div className="flex items-center gap-3">
              <Slider value={[Number(field.value ?? (min + max) / 2)]} min={min} max={max} step={step} onValueChange={(vals) => field.onChange(vals[0])} className="w-full" />
              <span className="text-xs text-muted-foreground w-10 text-right">{(Number(field.value ?? 0)).toFixed(2)}</span>
            </div>
          </FormControl>
          {description ? <div className="text-xs text-muted-foreground">{description}</div> : null}
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

