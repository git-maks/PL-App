export interface Toast {
	id: number;
	message: string;
	action?: { label: string; run: () => void };
}

export const toasts = $state<{ items: Toast[] }>({ items: [] });
let nextId = 1;

export function toast(message: string, opts: { action?: Toast['action']; duration?: number } = {}): number {
	const id = nextId++;
	toasts.items = [...toasts.items.slice(-2), { id, message, action: opts.action }];
	const duration = opts.duration ?? (opts.action ? 8000 : 4000);
	if (duration > 0) setTimeout(() => dismissToast(id), duration);
	return id;
}

export function dismissToast(id: number) {
	toasts.items = toasts.items.filter((t) => t.id !== id);
}
