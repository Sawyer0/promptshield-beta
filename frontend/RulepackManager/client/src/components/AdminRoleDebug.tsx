import React, { useState } from 'react';

export function AdminRoleDebug() {
  try {
    const raw = localStorage.getItem('ps_roles');
    const roles = raw ? JSON.parse(raw) : [];
    const tenant = localStorage.getItem('promptshield_tenant_name') || localStorage.getItem('promptshield_tenant_id');
    const devBypass = (import.meta as any).env?.VITE_ALLOW_DEV_BYPASS === 'true';
    const [_, setTick] = useState(0);
    if (!roles || roles.length === 0) return null;
    return (
      <div className="fixed bottom-2 right-2 text-[10px] px-2 py-1 rounded bg-black/60 text-white z-50">
        <div>roles: {Array.isArray(roles) ? roles.join(',') : String(roles)}</div>
        {tenant ? <div>tenant: {tenant}</div> : null}
        {devBypass && (
          <div className="mt-1 flex items-center gap-1">
            <span>switch:</span>
            {['platform_admin','tenant_admin','security_engineer','developer','auditor'].map(r => (
              <button
                key={r}
                className="px-1 py-0.5 bg-white/10 hover:bg-white/20 rounded"
                onClick={() => {
                  try {
                    const currRaw = localStorage.getItem('ps_roles');
                    let curr = currRaw ? JSON.parse(currRaw) : [];
                    if (!Array.isArray(curr)) curr = [];
                    if (curr.includes(r)) curr = curr.filter((x: string) => x !== r); else curr.push(r);
                    localStorage.setItem('ps_roles', JSON.stringify(curr));
                    setTick(t => t + 1);
                  } catch {}
                }}
              >{r}</button>
            ))}
          </div>
        )}
      </div>
    );
  } catch {
    return null;
  }
}
