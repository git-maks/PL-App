import { dev } from '$app/env';
import { href } from '#lib/paths';
import { toast } from '#lib/state/toast.svelte';

/**
 * Registers the service worker. A new version waits until the user agrees to reload
 * (never in the middle of scanning), as described in spec §5.7.
 */
let updateAccepted = false;

export function registerServiceWorker() {
	if (dev || !('serviceWorker' in navigator)) return;
	navigator.serviceWorker
		.register(href('/service-worker.js'), { type: 'module', scope: href('/') })
		.then((reg) => {
			const offer = (worker: ServiceWorker) => {
				if (location.pathname.includes('/scan')) return;
				toast('Dostępna nowa wersja UniQuest', {
					duration: 0,
					action: {
						label: 'Odśwież',
						run: () => {
							updateAccepted = true;
							worker.postMessage({ type: 'SKIP_WAITING' });
						}
					}
				});
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
