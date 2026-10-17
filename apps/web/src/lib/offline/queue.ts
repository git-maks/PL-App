import { openKV } from './idb';

/** A scan captured while offline, replayed to the server when connectivity returns (spec §5.7). */
export interface QueuedScan {
	id: string;
	payload: string;
	scannedAt: string;
	rating: number | null;
}

const kv = openKV('uniquest-queue');

export async function listQueue(): Promise<QueuedScan[]> {
	const items = await Promise.all((await kv.keys()).map((k) => kv.get<QueuedScan>(k)));
	return items.filter((x): x is QueuedScan => !!x).sort((a, b) => a.scannedAt.localeCompare(b.scannedAt));
}

export async function enqueue(payload: string): Promise<QueuedScan> {
	const item: QueuedScan = { id: crypto.randomUUID(), payload, scannedAt: new Date().toISOString(), rating: null };
	await kv.set(item.id, item);
	return item;
}

export async function updateQueued(item: QueuedScan): Promise<void> {
	await kv.set(item.id, item);
}

export async function removeQueued(id: string): Promise<void> {
	await kv.del(id);
}
