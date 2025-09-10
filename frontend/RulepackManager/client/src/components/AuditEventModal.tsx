import { Modal } from '@/components/common/Modal';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { JSONViewer } from '@/components/common/JSONViewer';
import { Eye, User, Shield, Activity } from 'lucide-react';

interface AuditEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  event: any | null;
}

export function AuditEventModal({ isOpen, onClose, event }: AuditEventModalProps) {
  if (!event) return null;

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      year: 'numeric', month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      size="lg"
      title={
        <div className="flex items-center gap-2">
          <Activity className="h-5 w-5" />
          <span>Audit Event</span>
          {event.decision && <Badge variant="secondary">{String(event.decision).toUpperCase()}</Badge>}
        </div>
      }
      description={event.id}
    >
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Summary</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <div className="text-muted-foreground">Action</div>
              <div className="font-medium">{String(event.action).replace('_', ' ')}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Timestamp</div>
              <div className="font-mono">{formatTimestamp(event.timestamp)}</div>
            </div>
            <div className="flex items-center gap-2">
              <User className="h-3 w-3 text-muted-foreground" />
              <div>
                <div className="text-muted-foreground">User</div>
                <div className="font-medium">{event.user_email || event.user_id || 'System'}</div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Shield className="h-3 w-3 text-muted-foreground" />
              <div>
                <div className="text-muted-foreground">Status</div>
                <div className="font-medium">{event.status_code || 'N/A'}</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Raw Event JSON</CardTitle>
          </CardHeader>
          <CardContent>
            <JSONViewer data={event} />
          </CardContent>
        </Card>
      </div>
    </Modal>
  );
}

