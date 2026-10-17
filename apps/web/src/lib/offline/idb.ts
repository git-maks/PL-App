/**
 * Minimal promise-based key-value store on IndexedDB (one object store per database).
 * Falls back to memory when IndexedDB is unavailable (SSR, some private modes), so callers
 * never have to special-case storage failures.
 */
export interface KV {
	get<T>(key: string): Promise<T | undefined>;
	set(key: string, value: unknown): Promise<void>;
	del(key: string): Promise<void>;
	keys(): Promise<string[]>;
	clear(): Promise<void>;
}

const STORE = 'kv';

function memoryKV(): KV {
	const m = new Map<string, unknown>();
	return {
		get: async <T>(k: string) => structuredClone(m.get(k)) as T | undefined,
		set: async (k, v) => void m.set(k, structuredClone(v)),
		del: async (k) => void m.delete(k),
		keys: async () => [...m.keys()],
		clear: async () => m.clear()
	};
}

function request<T>(r: IDBRequest<T>): Promise<T> {
	return new Promise((resolve, reject) => {
		r.onsuccess = () => resolve(r.result);
		r.onerror = () => reject(r.error);
	});
}

export function openKV(dbName: string): KV {
	if (typeof indexedDB === 'undefined') return memoryKV();
	let dbp: Promise<IDBDatabase> | null = null;
	const db = () =>
		(dbp ??= new Promise<IDBDatabase>((resolve, reject) => {
			const open = indexedDB.open(dbName, 1);
			open.onupgradeneeded = () => open.result.createObjectStore(STORE);
			open.onsuccess = () => resolve(open.result);
			open.onerror = () => reject(open.error);
		}));
	let fallback: KV | null = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	async function run<T>(mode: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest<any>, alt: (kv: KV) => Promise<T>): Promise<T> {
		if (fallback) return alt(fallback);
		try {
			const tx = (await db()).transaction(STORE, mode);
			return (await request(fn(tx.objectStore(STORE)))) as T;
		} catch {
			fallback = memoryKV();
			return alt(fallback);
		}
	}
	return {
		get: <T>(k: string) => run<T | undefined>('readonly', (s) => s.get(k), (f) => f.get<T>(k)),
		set: (k, v) => run<unknown>('readwrite', (s) => s.put(v, k), (f) => f.set(k, v)).then(() => {}),
		del: (k) => run<unknown>('readwrite', (s) => s.delete(k), (f) => f.del(k)).then(() => {}),
		keys: () => run<IDBValidKey[]>('readonly', (s) => s.getAllKeys(), (f) => f.keys()).then((ks) => ks.map(String)),
		clear: () => run<unknown>('readwrite', (s) => s.clear(), (f) => f.clear()).then(() => {})
	};
}
