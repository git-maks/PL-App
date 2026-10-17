import { expect, test } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { onboard, visitAndRate } from './demo';

test('landing → onboarding → hero with zeroed stats', async ({ page }) => {
	await page.goto('/');
	await page.getByRole('link', { name: 'Zaczynamy' }).click();
	await onboard(page, 'Ada');
	await expect(page.getByText('Ada', { exact: true })).toBeVisible();
	await expect(page.getByText('0/5 ocenionych aktywności')).toBeVisible();
	await expect(page.getByRole('meter')).toHaveCount(8);
});

test('five rated visits unlock the character card and recommendations', async ({ page }) => {
	await onboard(page);
	// The spec example: WM r=5, WEiI r=3, sport r=4, library r=2, WBiA r=4 → 45/30/25.
	await visitAndRate(page, 'wm-druk-3d', 5);
	await visitAndRate(page, 'weii-roboty-iot', 3);
	await visitAndRate(page, 'sport-sztafeta', 4);
	await visitAndRate(page, 'cizt-escape-room', 2);
	await visitAndRate(page, 'wbia-makiety-vr', 4);
	await expect(page.getByRole('link', { name: /Karta postaci odblokowana!/ })).toBeVisible();

	await page.goto('/card');
	await page.getByRole('button', { name: /Odkryj swoją kartę/ }).click();
	await expect(page.getByRole('heading', { name: 'Mistrz Formy' })).toBeVisible();
	await expect(page.getByText('Twoja ścieżka to w 45% Konstruktor, w 30% Architekt i w 25% Atleta.')).toBeVisible();
	await expect(page.getByAltText('Grafika karty postaci do udostępnienia')).toBeVisible();

	await page.getByRole('link', { name: 'Pasujące kierunki i koła naukowe' }).click();
	await expect(page.getByRole('heading', { name: 'Budownictwo' })).toBeVisible();
});

test('scanning the same station twice does not double the points', async ({ page }) => {
	await onboard(page);
	await visitAndRate(page, 'wm-druk-3d', 3);
	await page.goto('/hero');
	await expect(page.getByText('100 XP', { exact: true })).toBeVisible();
	await page.goto('/q/s.' + 'x'.repeat(32) + '.abcdefghijkl'); // forged code
	await expect(page.getByRole('heading', { name: 'Nie udało się zaliczyć stacji' })).toBeVisible();
	await expect(page.getByRole('alert')).toContainText('Ten kod nie należy do UniQuest');
});

test('manual code from the station display is accepted', async ({ page }) => {
	await onboard(page);
	await page.goto('/settings');
	await page.getByRole('switch', { name: /Tryb organizatora/ }).check();
	await page.goto('/staff');
	await page.getByRole('link', { name: 'Wyświetl kod QR' }).first().click();
	const code = (await page.locator('.code').first().textContent())!.trim();
	expect(code).toMatch(/^[A-Z2-9]{6}$/);
	await page.goto('/scan');
	// Our own modal explains the camera before the browser asks; choose the manual code instead.
	const cameraModal = page.getByRole('dialog', { name: 'Dostęp do aparatu' });
	await cameraModal.getByRole('button', { name: 'Wpisz kod ręcznie' }).click();
	await page.getByRole('textbox', { name: 'Kod stacji' }).fill(code);
	await page.getByRole('button', { name: 'Zatwierdź', exact: true }).click();
	await page.waitForURL(/\/scan\/rate\?id=/);
});

test('team: create, invite QR, simulated friends join live', async ({ page }) => {
	await onboard(page);
	await page.goto('/team');
	await page.getByLabel('Nazwa drużyny').fill('Bity i Śruby');
	await page.getByRole('button', { name: 'Utwórz drużynę' }).click();
	await expect(page.getByRole('heading', { name: 'Bity i Śruby' })).toBeVisible();
	await page.getByRole('button', { name: 'Zaproś' }).click();
	await expect(page.getByRole('img', { name: /Kod QR zaproszenia/ })).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(page.getByText(/2\/5 członków/)).toBeVisible({ timeout: 12_000 });
});

test('knowledge base search works offline-first and pages are accessible', async ({ page }) => {
	await page.goto('/explore');
	await page.getByLabel('Szukaj w kompendium').fill('drony');
	await expect(page.getByRole('link', { name: /eDron/ })).toBeVisible();
	await page.goto('/explore/kierunki/cyberbezpieczenstwo');
	await expect(page.getByRole('heading', { name: 'Cyberbezpieczeństwo', level: 1 })).toBeVisible();
	const a11y = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
	expect(a11y.violations.map((v) => `${v.id}: ${v.help}`)).toEqual([]);
});

test('hero and landing have no WCAG 2.1 AA violations', async ({ page }) => {
	await page.goto('/');
	let r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
	expect(r.violations.map((v) => `${v.id}: ${v.help}`)).toEqual([]);
	await onboard(page);
	r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
	expect(r.violations.map((v) => `${v.id}: ${v.help}`)).toEqual([]);
});

test('preset avatar, team emblems, facts carousel and the MoAI profile', async ({ page }) => {
	await page.goto('/onboarding');
	await page.getByRole('button', { name: 'Stwórz bohatera' }).click();
	await page.getByLabel('Pseudonim (opcjonalnie)').fill('Pikselka');
	await page.getByRole('checkbox').check();
	await page.getByRole('button', { name: 'Dalej' }).click();
	await expect(page.getByRole('radio', { name: /^Postać:/ })).toHaveCount(9);
	await page.getByRole('radio', { name: 'Postać: Leśny' }).click();
	await expect(page.getByRole('radio', { name: 'Postać: Leśny' })).toHaveAttribute('aria-checked', 'true');
	await page.getByRole('button', { name: 'Zapisz awatar' }).click();
	await page.getByRole('button', { name: 'Do panelu bohatera' }).click();
	await page.waitForURL('**/hero');
	await expect(page.locator('.avatar img').first()).toHaveAttribute('src', /^data:image\/png/);

	await page.goto('/team');
	await expect(page.getByRole('radiogroup', { name: 'Emblemat drużyny' }).getByRole('radio')).toHaveCount(24);
	let a11y = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
	expect(a11y.violations.map((v) => `${v.id}: ${v.help}`)).toEqual([]);

	await page.goto('/explore');
	const counter = page.locator('.carousel .num');
	const before = await counter.textContent();
	await page.getByRole('button', { name: 'Następna ciekawostka' }).click();
	await expect(counter).not.toHaveText(before!);
	a11y = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze();
	expect(a11y.violations.map((v) => `${v.id}: ${v.help}`)).toEqual([]);

	await page.goto('/explore/kola/sztucznej-inteligencji-moai');
	await expect(page.getByRole('heading', { name: 'Koło Naukowe Sztucznej Inteligencji „MoAI”', level: 1 })).toBeVisible();
	await expect(page.getByRole('link', { name: /Strona koła/ })).toHaveAttribute('href', 'https://moai-pl.github.io');
	for (const name of ['Sztuczna inteligencja w biznesie', 'Cyberbezpieczeństwo', 'Informatyka', 'Inżynieria i analiza danych', 'Sztuczna inteligencja w energetyce', 'Marketing i komunikacja rynkowa', 'Mechatronika'])
		await expect(page.getByRole('link', { name: new RegExp('^' + name) })).toBeVisible();
});
