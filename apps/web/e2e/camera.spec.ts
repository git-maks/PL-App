import { expect, test } from '@playwright/test';
import { onboard } from './demo';

test('scans a station QR code with the camera and rates the visit', async ({ page }) => {
	await onboard(page);
	await page.goto('/scan');
	// The fake camera shows the QR code of "Programowanie robotów i IoT".
	await page.waitForURL(/\/scan\/rate\?id=/, { timeout: 30_000 });
	await expect(page.getByRole('heading', { name: 'Programowanie robotów i IoT' })).toBeVisible();
	await page.getByRole('button', { name: 'Zatwierdź ocenę' }).click();
	await expect(page.getByText('+100 XP')).toBeVisible();
	await page.getByRole('link', { name: 'Panel bohatera' }).click();
	await expect(page.getByText('100 XP', { exact: true })).toBeVisible();
});
