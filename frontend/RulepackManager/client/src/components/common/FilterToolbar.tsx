import { ReactNode } from 'react';

interface FilterToolbarProps {
  search?: ReactNode;
  children?: ReactNode;
  actions?: ReactNode;
  className?: string;
}

// Responsive filter toolbar used on list pages. Layout matches existing pages.
export function FilterToolbar({ search, children, actions, className }: FilterToolbarProps) {
  return (
    <div className={"p-4 sm:p-6 " + (className || '')}>
      <div className="flex flex-col space-y-3 sm:space-y-4">
        {search ? <div className="w-full">{search}</div> : null}
        <div className="flex flex-col sm:flex-row gap-2 sm:gap-4 items-stretch sm:items-center">
          <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2 sm:gap-3">
            {children}
          </div>
          {actions ? <div className="flex gap-2 sm:ml-auto">{actions}</div> : null}
        </div>
      </div>
    </div>
  );
}

