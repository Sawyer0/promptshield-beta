import { useState } from 'react';
import { useSafeSignIn, useSafeClerk } from '@/clerk/shim';
import { Modal } from '@/components/common/Modal';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';

interface Props {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSuccess?: () => void;
}

export function SignInModal({ open, onOpenChange, onSuccess }: Props) {
  const { signIn, isLoaded } = useSafeSignIn();
  const { setActive } = useSafeClerk();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [needsEmailCode, setNeedsEmailCode] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState<'password'|'email_code'|'oauth'>('email_code');

  // Session existence is handled gracefully in submit catch ('session_exists')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isLoaded) return;
    // If a session already exists, Clerk will return a session_exists error; handled in catch
    setLoading(true);
    setError(null);
    try {
      if (mode === 'password') {
        const res = await signIn.create({ identifier: email.trim(), password });
        if (res.status === 'complete') {
          await setActive({ session: res.createdSessionId });
          onOpenChange(false);
          onSuccess?.();
        } else {
          setError('Additional verification required. Please complete the next step.');
        }
      } else if (mode === 'email_code') {
        if (!needsEmailCode) {
          await (signIn as any).create({ identifier: email.trim(), strategy: 'email_code' });
          setNeedsEmailCode(true);
        } else {
          const res = await (signIn as any).attemptFirstFactor({ strategy: 'email_code', code });
          if (res.status === 'complete') {
            await setActive({ session: res.createdSessionId });
            onOpenChange(false);
            onSuccess?.();
          } else {
            setError('Invalid or expired code.');
          }
        }
      }
    } catch (err: any) {
      const code = err?.errors?.[0]?.code;
      const msg = err?.errors?.[0]?.message;
      // If Clerk reports an existing session, just close and continue
      if (code === 'session_exists' || /session already signed in/i.test(msg || '')) {
        onOpenChange(false);
        onSuccess?.();
      } else {
        setError(msg || 'Sign in failed');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleOAuth = async (provider: 'oauth_google'|'oauth_github'|'oauth_microsoft') => {
    if (!isLoaded) return;
    setError(null);
    try {
      await (signIn as any).authenticateWithRedirect({
        strategy: provider,
        redirectUrl: window.location.origin,
        redirectUrlComplete: window.location.origin + '/',
      });
    } catch (err: any) {
      setError(err?.errors?.[0]?.message || 'OAuth sign-in failed');
    }
  };

  return (
    <Modal isOpen={open} onClose={() => onOpenChange(false)} title="Sign In">
      <Tabs value={mode} onValueChange={(v) => setMode(v as any)} className="w-full">
        <TabsList className="grid grid-cols-3">
          <TabsTrigger value="password">Password</TabsTrigger>
          <TabsTrigger value="email_code">Email Code</TabsTrigger>
          <TabsTrigger value="oauth">OAuth</TabsTrigger>
        </TabsList>

        <TabsContent value="password">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            </div>
            <div>
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
            </div>
            {error && <div className="text-sm text-red-600">{error}</div>}
            <div className="flex justify-end">
              <Button type="submit" disabled={loading}>{loading ? 'Signing in…' : 'Sign In'}</Button>
            </div>
          </form>
        </TabsContent>

        <TabsContent value="email_code">
          {!needsEmailCode ? (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="email2">Email</Label>
                <Input id="email2" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
              </div>
              {error && <div className="text-sm text-red-600">{error}</div>}
              <div className="flex justify-end">
                <Button type="submit" disabled={loading}>{loading ? 'Sending code…' : 'Send code'}</Button>
              </div>
            </form>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="code">Verification code</Label>
                <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} required />
              </div>
              {error && <div className="text-sm text-red-600">{error}</div>}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setNeedsEmailCode(false)}>Back</Button>
                <Button type="submit" disabled={loading}>{loading ? 'Verifying…' : 'Verify & Sign In'}</Button>
              </div>
            </form>
          )}
        </TabsContent>

        <TabsContent value="oauth">
          <div className="space-y-3">
            {error && <div className="text-sm text-red-600">{error}</div>}
            <Button className="w-full" variant="outline" onClick={() => handleOAuth('oauth_google')}>Continue with Google</Button>
            <Button className="w-full" variant="outline" onClick={() => handleOAuth('oauth_github')}>Continue with GitHub</Button>
            <Button className="w-full" variant="outline" onClick={() => handleOAuth('oauth_microsoft')}>Continue with Microsoft</Button>
          </div>
        </TabsContent>
      </Tabs>
    </Modal>
  );
}
