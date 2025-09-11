import { Button } from '@/components/ui/button';
import { Menu } from 'lucide-react';

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  onMenuClick?: () => void;
  actions?: React.ReactNode;
}

export function PageHeader({ title, subtitle, onMenuClick, actions }: PageHeaderProps) {

  return (
    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 sm:gap-2">
      <div className="flex items-center gap-3 sm:gap-4 min-w-0 flex-1">
        {onMenuClick && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onMenuClick}
            className="hover:bg-slate-200 dark:hover:bg-slate-700 flex-shrink-0"
            data-testid="button-menu"
          >
            <Menu className="h-5 w-5" />
          </Button>
        )}
        <div className="min-w-0">
          <h1 className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white truncate">
            {title}
          </h1>
          {subtitle && (
            <p className="text-slate-600 dark:text-slate-400 text-sm sm:text-base">
              {subtitle}
            </p>
          )}
        </div>
      </div>
      <div className="flex gap-2 w-full sm:w-auto items-center">
        {actions}
      </div>
    </div>
  );
}