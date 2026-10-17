// Mirrors the demo backend's deterministic ids and secrets (src/lib/api/demo/model.ts) so tests
// can mint valid station QR payloads without a server.
import { createHash, createHmac } from 'node:crypto';
import type { Page } from '@playwright/test';

export const ORIGIN = 'http://localhost:4173';

export function demoUuid(name: string): string {
	const b = createHash('sha256').update('uniquest-demo|' + name).digest().subarray(0, 16);
	b[6] = (b[6]! & 0x0f) | 0x50;
	b[8] = (b[8]! & 0x3f) | 0x80;
	const h = b.toString('hex');
	return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

export function staticPayload(slug: string, origin = ORIGIN): string {
	const id = demoUuid('station|' + slug).replaceAll('-', '');
	const secret = createHash('sha256').update('uniquest-demo-secret|' + slug).digest();
	const mac = createHmac('sha256', secret).update('s|' + id).digest('base64url').slice(0, 12);
	return `${origin}/q/s.${id}.${mac}`;
}

/** Creates a guest hero through the real onboarding UI. */
export async function onboard(page: Page, nickname = 'Testowy Bohater') {
	await page.goto('/onboarding');
	await page.getByRole('button', { name: 'Stwórz bohatera' }).click();
	await page.getByLabel('Pseudonim (opcjonalnie)').fill(nickname);
	await page.getByRole('checkbox').check();
	await page.getByRole('button', { name: 'Dalej' }).click();
	await page.getByRole('button', { name: /Pomiń/ }).click();
	await page.getByRole('button', { name: 'Do panelu bohatera' }).click();
	await page.waitForURL('**/hero');
}

/** Opens a station QR link (as the phone camera would) and rates the visit. */
export async function visitAndRate(page: Page, slug: string, rating: number) {
	await page.goto(staticPayload(slug).replace(ORIGIN, ''));
	await page.waitForURL(/\/scan\/rate\?id=/);
	const slider = page.getByRole('slider');
	await slider.focus();
	await slider.press('Home');
	for (let i = 1; i < rating; i++) await slider.press('ArrowRight');
	await page.getByRole('button', { name: 'Zatwierdź ocenę' }).click();
	await page.getByText(/\+\d+ XP/).first().waitFor();
}
