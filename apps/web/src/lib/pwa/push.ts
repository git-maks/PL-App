import type { ApiClient } from '#lib/api';
import { ask } from '#lib/state/modal.svelte';

export type PushResult = 'enabled' | 'denied' | 'dismissed' | 'unsupported' | 'unavailable';

const DISMISSED_KEY = 'uq-push-dismissed';

function urlBase64ToUint8Array(base64: string): Uint8Array {
	const padding = '='.repeat((4 - (base64.length % 4)) % 4);
	const raw = atob((base64 + padding).replace(/-/g, '+').replace(/_/g, '/'));
	return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}

/**
 * Asks for notification permission (only ever after a user action, spec §7.7) and registers the
 * Web Push subscription with the backend. On iOS this works only in the installed PWA.
 */
export async function enablePush(api: ApiClient, { explain = true, skipIfDismissed = false } = {}): Promise<PushResult> {
	if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) return 'unsupported';
	if (Notification.permission === 'denied') return 'denied';
	// Without server keys (e.g. the static demo) there is nothing to ask permission for.
	const key = await api.pushPublicKey();
	if (!key) return 'unavailable';
	if (Notification.permission === 'default') {
		// The browser's permission prompt can't be styled, so our own modal explains it first.
		if (skipIfDismissed && dismissedBefore()) return 'dismissed';
		if (explain) {
			const ok = await ask({
				icon: 'notifications_active',
				title: 'Włączyć powiadomienia?',
				body: 'Przypomnimy Ci o zajęciach 10 minut przed startem i damy znać, gdy drużyna zaliczy zadanie. Za chwilę przeglądarka poprosi o zgodę.',
				confirm: 'Włącz',
				cancel: 'Nie teraz'
			});
			if (!ok) {
				try {
					localStorage.setItem(DISMISSED_KEY, '1');
				} catch {
					/* storage unavailable */
				}
				return 'dismissed';
			}
		}
		// Asked straight after the tap on "Włącz", while the browser still counts it as a user action.
		if ((await Notification.requestPermission()) !== 'granted') return 'denied';
	}
	const reg = await navigator.serviceWorker.ready;
	const sub =
		(await reg.pushManager.getSubscription()) ??
		(await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: urlBase64ToUint8Array(key) as BufferSource }));
	const json = sub.toJSON() as { endpoint: string; keys: { p256dh: string; auth: string } };
	await api.pushSubscribe({ endpoint: json.endpoint, keys: json.keys });
	return 'enabled';
}

export async function disablePush(api: ApiClient): Promise<void> {
	if (!('serviceWorker' in navigator)) return;
	const reg = await navigator.serviceWorker.getRegistration();
	const sub = await reg?.pushManager.getSubscription();
	if (!sub) return;
	await api.pushUnsubscribe(sub.endpoint).catch(() => {});
	await sub.unsubscribe();
}

function dismissedBefore() {
	try {
		return localStorage.getItem(DISMISSED_KEY) === '1';
	} catch {
		return false;
	}
}

export function pushPermission(): NotificationPermission | 'unsupported' {
	return 'Notification' in window ? Notification.permission : 'unsupported';
}
