import { expect, type APIRequestContext, type Page } from '@playwright/test';
import { ADMIN_TOKEN } from './env';

export interface SentCheck {
  status: number;
  requestId: string;
}

/** Sends a prompt straight to the gateway, as an application would. */
export async function sendCheck(request: APIRequestContext, prompt: string): Promise<SentCheck> {
  const res = await request.post('/check', { data: prompt, headers: { 'Content-Type': 'text/plain' } });
  return { status: res.status(), requestId: res.headers()['x-ps-request-id'] ?? '' };
}

export async function signIn(page: Page): Promise<void> {
  await page.goto('/dashboard/');
  await page.getByLabel('Admin token').fill(ADMIN_TOKEN);
  await page.getByRole('button', { name: 'Open dashboard' }).click();
  await expect(page.getByRole('heading', { name: 'Live decisions' })).toBeVisible();
  await expect(page.getByRole('status').filter({ hasText: 'Event stream: Live' })).toBeVisible();
}

/** Reads a numeric stat card value (e.g. "1,204" -> 1204). */
export async function statValue(page: Page, id: 'total' | 'blocked' | 'allowed'): Promise<number> {
  const text = await page.getByTestId(`stat-${id}`).locator('.stat__value').innerText();
  return Number(text.replace(/[^0-9]/g, ''));
}

export function adminHeaders(): Record<string, string> {
  return { Authorization: `Bearer ${ADMIN_TOKEN}` };
}
