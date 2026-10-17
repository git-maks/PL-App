import { expect, test, type Page } from '@playwright/test';
import { onboard } from '../e2e/demo';

// Requires: `uniquest seed -admin admin@pollub.test` and the server running with
// UQ_DEV_LOGIN_CODES=true (login codes are shown in the UI instead of e-mailed).

async function loginAsAdmin(page: Page) {
	await onboard(page, 'Organizator testowy');
	await page.goto('/settings');
	await page.getByRole('button', { name: 'Dodaj e-mail' }).click();
	await page.getByRole('textbox', { name: 'Adres e-mail' }).fill('admin@pollub.test');
	await page.getByRole('checkbox', { name: /16 lat/ }).check();
	await page.getByRole('button', { name: 'Wyślij kod' }).click();
	const code = (await page.locator('.dev strong').textContent())!.trim();
	await page.getByRole('textbox', { name: 'Kod' }).fill(code);
	await page.getByRole('button', { name: 'Potwierdź' }).click();
	await expect(page.getByText('admin@pollub.test', { exact: true })).toBeVisible();
}

test('staff display → manual code → rating → XP, against the Go backend', async ({ page }) => {
	await loginAsAdmin(page);
	await page.goto('/staff');
	await page.getByRole('link', { name: 'Wyświetl kod QR' }).first().click();
	const code = (await page.locator('.code').first().textContent())!.trim();
	expect(code).toMatch(/^[A-Z2-9]{6}$/);

	await page.goto('/scan');
	await page.getByRole('button', { name: 'Wpisz kod ręcznie' }).click();
	await page.getByRole('textbox', { name: 'Kod stacji' }).fill(code);
	await page.getByRole('button', { name: 'Zatwierdź', exact: true }).click();
	await page.waitForURL(/\/scan\/rate\?id=/);
	await page.getByRole('button', { name: 'Zatwierdź ocenę' }).click();
	await expect(page.getByText(/^\+\d+ XP$/)).toBeVisible();
	await page.getByRole('link', { name: 'Panel bohatera' }).click();
	await expect(page.getByText('0/5 ocenionych aktywności')).toHaveCount(0);
	await expect(page.getByText('1/5 ocenionych aktywności')).toBeVisible();
});

test('team invite: a second player joins and the captain sees it live (SSE)', async ({ browser }) => {
	const captain = await (await browser.newContext()).newPage();
	await onboard(captain, 'Kapitan Nemo');
	await captain.goto('/team');
	await captain.getByLabel('Nazwa drużyny').fill('Nautilus');
	await captain.getByRole('button', { name: 'Utwórz drużynę' }).click();
	await expect(captain.getByText('1/5 członków', { exact: false })).toBeVisible();
	await captain.getByRole('button', { name: 'Zaproś' }).click();
	const code = (await captain.locator('.invite .code').textContent())!.trim();
	await captain.keyboard.press('Escape');

	const sailor = await (await browser.newContext()).newPage();
	await onboard(sailor, 'Marynarz');
	await sailor.goto('/t/' + code);
	await sailor.getByRole('button', { name: 'Dołącz do drużyny' }).click();
	await sailor.waitForURL('**/team');
	await expect(sailor.getByRole('heading', { name: 'Nautilus' })).toBeVisible();

	// No reload: the captain's page updates from the event stream.
	await expect(captain.getByText('2/5 członków', { exact: false })).toBeVisible();
	await expect(captain.getByText('Marynarz', { exact: true })).toBeVisible();
});
