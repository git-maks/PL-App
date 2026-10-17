/// <reference lib="webworker" />
/*
 * UniQuest service worker (spec §5.7):
 *  - precache: app shell, hashed build files, static assets, prerendered pages (knowledge base works offline)
 *  - navigation: cached page → network → cached shell (SPA fallback)
 *  - API GETs: network-first with cache fallback; mutations always go to the network
 *  - Web Push: show notification, focus/open the app on click
 */
import { self } from '$app/service-worker';
import { immutable, assets, prerendered } from '$app/manifest';
import { version } from '$app/env';
import { resolve } from '$app/paths';

const BASE = (resolve as (p: string) => string)('/').replace(/\/$/, '');
const CACHE = `uq-static-${version}`;
const API_CACHE = 'uq-api';
const SHELL = BASE + '/';

const PRECACHE = [
	...immutable.map((f) => BASE + '/' + f.path.replace(/^\//, '')),
	...assets.map((f) => BASE + '/' + f.path.replace(/^\//, '')),
	...prerendered.map((p) => BASE + (p.path.startsWith('/') ? p.path : '/' + p.path))
];
const PRECACHE_SET = new Set(PRECACHE);

self.addEventListener('install', (event) => {
	event.waitUntil(
		caches.open(CACHE).then(async (cache) => {
			// Add one by one so a single failing asset does not abort the install.
			await Promise.all(PRECACHE.map((url) => cache.add(url).catch(() => undefined)));
		})
	);
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			for (const key of await caches.keys()) {
				if (key !== CACHE && key !== API_CACHE) await caches.delete(key);
			}
			await self.clients.claim();
		})()
	);
});

self.addEventListener('message', (event) => {
	if ((event.data as { type?: string })?.type === 'SKIP_WAITING') void self.skipWaiting();
});

async function networkFirst(request: Request, cacheName: string): Promise<Response> {
	const cache = await caches.open(cacheName);
	try {
		const res = await fetch(request);
		if (res.ok) await cache.put(request, res.clone());
		return res;
	} catch (e) {
		const hit = await cache.match(request);
		if (hit) return hit;
		throw e;
	}
}

self.addEventListener('fetch', (event) => {
	const req = event.request;
	if (req.method !== 'GET') return;
	const url = new URL(req.url);
	if (url.origin !== self.location.origin) return;

	if (url.pathname.startsWith(BASE + '/api/')) {
		// Never cache the event stream; other reads fall back to the last answer when offline.
		if (url.pathname.endsWith('/stream')) return;
		event.respondWith(networkFirst(req, API_CACHE));
		return;
	}

	if (req.mode === 'navigate') {
		event.respondWith(
			(async () => {
				const cache = await caches.open(CACHE);
				const path = url.pathname.replace(/\/$/, '') || BASE + '/';
				const cached = (await cache.match(path)) ?? (await cache.match(path + '.html'));
				if (cached) return cached;
				try {
					return await fetch(req);
				} catch {
					return (await cache.match(SHELL)) ?? Response.error();
				}
			})()
		);
		return;
	}

	if (PRECACHE_SET.has(url.pathname)) {
		event.respondWith(caches.match(url.pathname).then((hit) => hit ?? fetch(req)));
		return;
	}
	// Lazily loaded files (e.g. the QR decoder WASM) are cached on first use.
	if (url.pathname.startsWith(BASE + '/_app/')) {
		event.respondWith(
			caches.open(CACHE).then(async (cache) => {
				const hit = await cache.match(req);
				if (hit) return hit;
				const res = await fetch(req);
				if (res.ok) await cache.put(req, res.clone());
				return res;
			})
		);
	}
});

self.addEventListener('push', (event) => {
	let data: { title?: string; body?: string; url?: string; tag?: string } = {};
	try {
		data = event.data?.json() ?? {};
	} catch {
		data = { body: event.data?.text() };
	}
	event.waitUntil(
		self.registration.showNotification(data.title ?? 'UniQuest', {
			body: data.body ?? '',
			icon: BASE + '/icons/icon-192.png',
			badge: BASE + '/icons/badge-72.png',
			tag: data.tag,
			data: { url: data.url ?? BASE + '/hero' }
		})
	);
});

self.addEventListener('notificationclick', (event) => {
	event.notification.close();
	const target = new URL((event.notification.data as { url?: string })?.url ?? BASE + '/', self.location.origin).href;
	event.waitUntil(
		(async () => {
			const all = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
			for (const c of all) {
				if ('focus' in c) {
					await c.focus();
					if ('navigate' in c) await (c as WindowClient).navigate(target);
					return;
				}
			}
			await self.clients.openWindow(target);
		})()
	);
});
