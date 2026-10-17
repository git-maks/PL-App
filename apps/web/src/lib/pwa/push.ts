import type { ApiClient } from '#lib/api';

export type PushResult = 'enabled' | 'denied' | 'unsupported' | 'unavailable';

function urlBase64ToUint8Array(base64: string): Uint8Array {
	const padding = '='.repeat((4 - (base64.length % 4)) % 4);
	const raw = atob((base64 + padding).replace(/-/g, '+').replace(/_/g, '/'));
	return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}

/**
 * Asks for notification permission (only ever after a user action, spec §7.7) and registers the
 * Web Push subscription with the backend. On iOS this works only in the installed PWA.
 */
export async function enablePush(api: ApiClient): Promise<PushResult> {
	if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) return 'unsupported';
	const key = await api.pushPublicKey();
	if (!key) return 'unavailable';
	const permission = await Notification.requestPermission();
	if (permission !== 'granted') return 'denied';
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

export function pushPermission(): NotificationPermission | 'unsupported' {
	return 'Notification' in window ? Notification.permission : 'unsupported';
}
