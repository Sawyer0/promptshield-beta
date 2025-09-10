import { useEffect, useState } from 'react';
import { useSafeSignUp, useSafeClerk } from '@/clerk/shim';
import { Modal } from '@/components/common/Modal';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';

interface Props {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSuccess?: (organizationName?: string) => void;
}

export function SignUpModal({ open, onOpenChange, onSuccess }: Props) {
  const { signUp, isLoaded } = useSafeSignUp();
  const { setActive } = useSafeClerk();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [firstName, setFirst] = useState('');
  const [lastName, setLast] = useState('');
  const [orgName, setOrg] = useState('');
  const [code, setCode] = useState('');
  const [needsCode, setNeedsCode] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open) {
      setEmail(''); setPassword(''); setFirst(''); setLast(''); setOrg(''); setCode(''); setNeedsCode(false); setError(null);
    }
  }, [open]);

  const startSignUp = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isLoaded) return;
    setLoading(true);
    setError(null);
    try {
      const created = await signUp.create({ emailAddress: email.trim(), password, firstName, lastName });
      // If email verification is not required, Clerk might complete immediately
      if ((created as any)?.status === 'complete') {
        await setActive({ session: (created as any).createdSessionId });
        onOpenChange(false);
        onSuccess?.(orgName);
        return;
      }
      // Otherwise, try to start email code verification if enabled in Clerk
      try {
        await signUp.prepareEmailAddressVerification({ strategy: 'email_code' });
        setNeedsCode(true);
      } catch (prepErr: any) {
        // Email verification not enabled; surface a helpful message
        setError('Email verification is not enabled in Clerk. Enable Email Code or disable verification for sign up.');
      }
    } catch (err: any) {
      setError(err?.errors?.[0]?.message || 'Sign up failed');
    } finally {
      setLoading(false);
    }
  };

  const verifyCode = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isLoaded) return;
    setLoading(true);
    setError(null);
    try {
      const res = await signUp.attemptEmailAddressVerification({ code });
      if (res.status === 'complete') {
        await setActive({ session: res.createdSessionId });
        onOpenChange(false);
        onSuccess?.(orgName);
      } else {
        setError('Additional verification required.');
      }
    } catch (err: any) {
      setError(err?.errors?.[0]?.message || 'Verification failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={open} onClose={() => onOpenChange(false)} title="Create Account">
      {!needsCode ? (
        <form onSubmit={startSignUp} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label htmlFor="first">First name</Label>
              <Input id="first" value={firstName} onChange={(e) => setFirst(e.target.value)} required />
            </div>
            <div>
              <Label htmlFor="last">Last name</Label>
              <Input id="last" value={lastName} onChange={(e) => setLast(e.target.value)} />
            </div>
          </div>
          <div>
            <Label htmlFor="org">Organization name</Label>
            <Input id="org" value={orgName} onChange={(e) => setOrg(e.target.value)} required />
          </div>
          <div>
            <Label htmlFor="email">Email</Label>
            <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </div>
          <div>
            <Label htmlFor="pw">Password</Label>
            <Input id="pw" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
          </div>
          {error && <div className="text-sm text-red-600">{error}</div>}
          <div className="flex justify-end">
            <Button type="submit" disabled={loading}>{loading ? 'Creating…' : 'Create account'}</Button>
          </div>
        </form>
      ) : (
        <form onSubmit={verifyCode} className="space-y-4">
          <div>
            <Label htmlFor="code">Verification code</Label>
            <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="Enter the code sent to your email" required />
          </div>
          {error && <div className="text-sm text-red-600">{error}</div>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => setNeedsCode(false)}>Back</Button>
            <Button type="submit" disabled={loading}>{loading ? 'Verifying…' : 'Verify & Continue'}</Button>
          </div>
        </form>
      )}
    </Modal>
  );
}
  // Session guard handled by Clerk errors; proceed with flow
