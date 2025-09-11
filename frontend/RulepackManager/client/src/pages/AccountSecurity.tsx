import { useState } from 'react';
import { useSafeUser } from '@/clerk/shim';
import { Layout } from '@/components/Layout';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useToast } from '@/hooks/use-toast';

export default function AccountSecurity() {
  const { user, isLoaded } = useSafeUser();
  const { toast } = useToast();
  const [currentPassword, setCurrent] = useState('');
  const [newPassword, setNew] = useState('');
  const [confirmPassword, setConfirm] = useState('');
  const [saving, setSaving] = useState(false);

  const hasPassword = !!user && (user as any).passwordEnabled === true;

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isLoaded || !user) return;
    if (!newPassword || newPassword.length < 8) {
      toast({ title: 'Invalid password', description: 'Password must be at least 8 characters.', variant: 'destructive' });
      return;
    }
    if (newPassword !== confirmPassword) {
      toast({ title: 'Passwords do not match', description: 'Please confirm your new password.', variant: 'destructive' });
      return;
    }
    setSaving(true);
    try {
      if (hasPassword) {
        await (user as any).update({ password: newPassword, currentPassword });
      } else {
        await (user as any).update({ password: newPassword });
      }
      toast({ title: 'Password updated', description: 'You can now sign in with your password.' });
      setCurrent(''); setNew(''); setConfirm('');
    } catch (err: any) {
      const msg = err?.errors?.[0]?.message || 'Failed to update password';
      toast({ title: 'Error', description: msg, variant: 'destructive' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Layout title="Account Security" description="Manage your account security settings and sessions.">
      <div className="max-w-xl mx-auto p-4">
        <Card>
          <CardHeader>
            <CardTitle>Account Security</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSave} className="space-y-4">
              {hasPassword && (
                <div>
                  <Label htmlFor="current">Current password</Label>
                  <Input id="current" type="password" value={currentPassword} onChange={e => setCurrent(e.target.value)} />
                </div>
              )}
              <div>
                <Label htmlFor="new">New password</Label>
                <Input id="new" type="password" value={newPassword} onChange={e => setNew(e.target.value)} required />
              </div>
              <div>
                <Label htmlFor="confirm">Confirm new password</Label>
                <Input id="confirm" type="password" value={confirmPassword} onChange={e => setConfirm(e.target.value)} required />
              </div>
              <div className="flex justify-end">
                <Button type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save password'}</Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
