import { ReactNode } from 'react';
import { cn } from '@/lib/utils';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';

type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | '2xl' | 'full';

const sizeClass: Record<ModalSize, string> = {
  sm: 'max-w-md',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
  '2xl': 'max-w-6xl',
  full: 'max-w-[95vw]'
};

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
  size?: ModalSize;
  contentClassName?: string;
  allowClose?: boolean; // when false, prevent overlay/esc close
}

export function Modal({
  isOpen,
  onClose,
  title,
  description,
  children,
  footer,
  size = 'lg',
  contentClassName,
  allowClose = true,
}: ModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (allowClose && !open) onClose(); }}>
      <DialogContent className={cn(sizeClass[size], 'max-h-[90vh] overflow-y-auto', contentClassName)}>
        {(title || description) && (
          <DialogHeader>
            {title ? <DialogTitle>{title}</DialogTitle> : null}
            {description ? <DialogDescription>{description}</DialogDescription> : null}
          </DialogHeader>
        )}
        {children}
        {footer ? (
          <div className="pt-6 mt-2 border-t">{footer}</div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
