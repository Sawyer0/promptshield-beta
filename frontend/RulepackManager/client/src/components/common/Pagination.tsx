import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Button } from '@/components/ui/button';

interface PaginationProps {
  page: number; // zero-based
  totalPages: number;
  totalCount?: number;
  pageSize: number;
  pageSizeOptions?: number[];
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  disabled?: boolean;
}

export function Pagination({
  page,
  totalPages,
  totalCount,
  pageSize,
  pageSizeOptions = [25, 50, 100, 250],
  onPageChange,
  onPageSizeChange,
  disabled,
}: PaginationProps) {
  if (totalPages <= 1) return null;

  return (
    <div className="flex flex-col sm:flex-row items-center justify-between mt-4 sm:mt-6 gap-4">
      <div className="flex items-center space-x-2 order-2 sm:order-1">
        <span className="text-sm text-muted-foreground">Rows per page:</span>
        <Select value={String(pageSize)} onValueChange={(v) => onPageSizeChange(parseInt(v))}>
          <SelectTrigger className="w-20">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {pageSizeOptions.map((n) => (
              <SelectItem key={n} value={String(n)}>{n}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="text-sm text-muted-foreground order-3 sm:order-2">
        Page {page + 1} of {totalPages}{typeof totalCount === 'number' ? ` (${totalCount} total)` : ''}
      </div>
      <div className="flex items-center space-x-2 order-1 sm:order-3">
        <Button
          variant="outline"
          onClick={() => onPageChange(Math.max(0, page - 1))}
          disabled={page === 0 || disabled}
        >
          Previous
        </Button>
        <Button
          variant="outline"
          onClick={() => onPageChange(Math.min(totalPages - 1, page + 1))}
          disabled={page >= totalPages - 1 || disabled}
        >
          Next
        </Button>
      </div>
    </div>
  );
}

