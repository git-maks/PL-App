import { dev } from '$app/env';
import { href } from '#lib/paths';
import { ask } from '#lib/state/modal.svelte';

/**
 * Registers the service worker. A new version waits until the user agrees to reload
 * (never in the middle of scanning), as described in spec §5.7.
 */
let updateAccepted = false;
let waiting: ServiceWorker | null = null;
let asking = false;

/** Offers a downloaded update in a modal, unless the player is scanning (then after they leave). */
export async function offerUpdate() {
	if (!waiting || asking || updateAccepted || location.pathname.includes('/scan')) return;
	asking = true;
	const ok = await ask({
		icon: 'refresh',
		title: 'Nowa wersja UniQuest',
		body: 'Pobraliśmy aktualizację aplikacji. Odśwież teraz, aby z niej skorzystać – Twój postęp zostanie zachowany.',
		confirm: 'Odśwież teraz',
		cancel: 'Później'
	});
	asking = false;
	if (ok && waiting) {
		updateAccepted = true;
		waiting.postMessage({ type: 'SKIP_WAITING' });
	}
	// "Później": the update installs on the next start of the app.
	waiting = null;
}

export function registerServiceWorker() {
	if (dev || !('serviceWorker' in navigator)) return;
	navigator.serviceWorker
		.register(href('/service-worker.js'), { type: 'module', scope: href('/') })
		.then((reg) => {
			const offer = (worker: ServiceWorker) => {
				waiting = worker;
				void offerUpdate();
			};
			if (reg.waiting && navigator.serviceWorker.controller) offer(reg.waiting);
			reg.addEventListener('updatefound', () => {
				const w = reg.installing;
				w?.addEventListener('statechange', () => {
					if (w.state === 'installed' && navigator.serviceWorker.controller) offer(w);
				});
			});
			let reloaded = false;
			navigator.serviceWorker.addEventListener('controllerchange', () => {
				// The first install also fires controllerchange (clients.claim) – never reload then,
				// only when the user accepted an update.
				if (reloaded || !updateAccepted) return;
				reloaded = true;
				location.reload();
			});
		})
		.catch(() => {
			/* SW is an enhancement; the app works without it */
		});
}
