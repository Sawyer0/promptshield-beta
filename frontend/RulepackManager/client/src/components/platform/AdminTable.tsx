import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { ReactNode } from 'react';

export function AdminTable<T>({ columns, rows, empty }: { columns: { key: keyof T | string; label: string; render?: (row: T) => ReactNode }[]; rows: T[]; empty?: ReactNode; }) {
  return (
    <div className="rounded-xl border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((c) => (
              <TableHead key={String(c.key)}>{c.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length ? (
            rows.map((r, i) => (
              <TableRow key={i}>
                {columns.map((c) => (
                  <TableCell key={String(c.key)}>
                    {c.render ? c.render(r) : (r as any)[c.key]}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length}>{empty || <div className="p-4 text-sm text-muted-foreground">No data</div>}</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
