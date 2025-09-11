import { ReactNode } from 'react';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

export interface Column<T> {
  key: string;
  header: ReactNode;
  className?: string;
  cell: (row: T, index: number) => ReactNode;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  getRowKey: (row: T, index: number) => string | number;
  isLoading?: boolean;
  loadingRows?: number;
  emptyMessage?: ReactNode;
  rowClassName?: (row: T, index: number) => string | undefined;
}

export function DataTable<T>({
  columns,
  data,
  getRowKey,
  isLoading = false,
  loadingRows = 10,
  emptyMessage = 'No results',
  rowClassName,
}: DataTableProps<T>) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((c) => (
              <TableHead key={c.key} className={c.className}>{c.header}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            Array.from({ length: loadingRows }).map((_, i) => (
              <TableRow key={`loading-${i}`}>
                {columns.map((c) => (
                  <TableCell key={c.key} className="align-middle">
                    <Skeleton className="h-4 w-full" />
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : data.length === 0 ? (
            <TableRow>
              <TableCell colSpan={columns.length} className="text-center py-10 text-muted-foreground">
                {emptyMessage}
              </TableCell>
            </TableRow>
          ) : (
            data.map((row, i) => (
              <TableRow key={getRowKey(row, i)} className={cn('hover:bg-muted/50', rowClassName?.(row, i))}>
                {columns.map((c) => (
                  <TableCell key={c.key}>{c.cell(row, i)}</TableCell>
                ))}
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

