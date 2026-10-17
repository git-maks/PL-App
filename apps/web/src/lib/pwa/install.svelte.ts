import { browser } from '$app/env';

interface BeforeInstallPromptEvent extends Event {
	prompt(): Promise<void>;
	userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

/** Install state: Android/desktop Chrome offer a prompt; iOS needs manual "Add to Home Screen". */
export const install = $state({
	deferred: null as BeforeInstallPromptEvent | null,
	standalone: false,
	ios: false
});

if (browser) {
	install.standalone =
		matchMedia('(display-mode: standalone)').matches || (navigator as Navigator & { standalone?: boolean }).standalone === true;
	install.ios = /iphone|ipad|ipod/i.test(navigator.userAgent) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
	addEventListener('beforeinstallprompt', (e) => {
		e.preventDefault();
		install.deferred = e as BeforeInstallPromptEvent;
	});
	addEventListener('appinstalled', () => {
		install.deferred = null;
		install.standalone = true;
	});
}

export async function promptInstall(): Promise<boolean> {
	const d = install.deferred;
	if (!d) return false;
	await d.prompt();
	const { outcome } = await d.userChoice;
	install.deferred = null;
	return outcome === 'accepted';
}
