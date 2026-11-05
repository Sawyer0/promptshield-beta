import { useState, type FormEvent } from 'react';
import { UnauthorizedError } from '../../api/gatewayClient';
import { useSession } from '../../session/SessionContext';

/** Sign-in with the gateway admin token (PS_ENFORCER_ADMIN_TOKEN). */
export function TokenForm() {
  const { signIn, expired } = useSession();
  const [token, setToken] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token.trim()) {
      setError('Enter the admin token.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await signIn(token);
    } catch (err) {
      setError(err instanceof UnauthorizedError ? 'The gateway rejected that token.' : `Could not reach the gateway: ${String(err instanceof Error ? err.message : err)}`);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="signin">
      <form className="signin__card" onSubmit={submit} noValidate>
        <h1 className="signin__title">PromptShield Gateway</h1>
        <p className="muted">
          Enter the admin token configured as <code>PS_ENFORCER_ADMIN_TOKEN</code>. It is kept only for this browser tab.
        </p>
        {expired && !error && (
          <p className="notice" role="status">
            Your session ended because the gateway rejected the token. Sign in again.
          </p>
        )}
        <label htmlFor="admin-token">Admin token</label>
        <input
          id="admin-token"
          type="password"
          autoComplete="current-password"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'admin-token-error' : undefined}
        />
        {error && (
          <p id="admin-token-error" className="error" role="alert">
            {error}
          </p>
        )}
        <button type="submit" className="button button--primary" disabled={busy}>
          {busy ? 'Checking…' : 'Open dashboard'}
        </button>
      </form>
    </main>
  );
}
