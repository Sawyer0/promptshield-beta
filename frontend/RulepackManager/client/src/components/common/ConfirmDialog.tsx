import { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Modal } from './Modal';

interface ConfirmDialogProps {
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: () => void;
  title?: ReactNode;
  description?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  isLoading?: boolean;
}

export function ConfirmDialog({
  isOpen,
  onCancel,
  onConfirm,
  title = 'Confirm Action',
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  isLoading = false,
}: ConfirmDialogProps) {
  return (
    <Modal
      isOpen={isOpen}
      onClose={onCancel}
      title={title}
      description={description}
      size="sm"
      footer={
        <div className="flex items-center justify-end gap-2">
          <Button variant="outline" type="button" onClick={onCancel} disabled={isLoading}>
            {cancelLabel}
          </Button>
          <Button type="button" onClick={onConfirm} disabled={isLoading}>
            {isLoading ? 'Working…' : confirmLabel}
          </Button>
        </div>
      }
    >
      {/* empty body by default; description covers most use cases */}
    </Modal>
  );
}

