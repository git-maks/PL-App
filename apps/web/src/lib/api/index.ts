import { browser } from '$app/env';
import { BASE } from '#lib/paths';
import { HttpApi } from './http';
import type { ApiClient } from './types';

export * from './types';

let client: ApiClient | null = null;

/** The API client for this build: the Go backend, or the in-browser demo backend (GitHub Pages). */
export async function api(): Promise<ApiClient> {
	if (client) return client;
	if (__API_MODE__ === 'demo') {
		const { DemoApi } = await import('./demo');
		client = new DemoApi(browser ? location.origin + BASE : BASE);
	} else {
		client = new HttpApi(BASE + '/api/v1');
	}
	return client;
}

export const isDemo = __API_MODE__ === 'demo';
