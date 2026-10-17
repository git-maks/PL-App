import type { IconName } from '#lib/icons/paths';

/**
 * App-wide confirmation modal. Every question the app asks (update, permissions…) goes through
 * this instead of the browser's own alert/confirm, so it matches the rest of the UI.
 * Requests queue up; only one modal is shown at a time.
 */
export interface ModalRequest {
	title: string;
	body: string;
	icon?: IconName;
	confirm: string;
	/** Omit for a single-button notice. */
	cancel?: string;
}

interface Pending extends ModalRequest {
	id: number;
	resolve: (ok: boolean) => void;
}

let seq = 0;
const queue: Pending[] = [];
export const modal = $state<{ current: Pending | null }>({ current: null });

function next() {
	if (!modal.current && queue.length) modal.current = queue.shift()!;
}

/** Shows the modal and resolves true for the confirm button, false for cancel/Escape/backdrop. */
export function ask(req: ModalRequest): Promise<boolean> {
	return new Promise((resolve) => {
		queue.push({ ...req, id: ++seq, resolve });
		next();
	});
}

/** Answers the modal with the given id; stale answers (e.g. a late close event) are ignored. */
export function answer(id: number, ok: boolean) {
	const c = modal.current;
	if (!c || c.id !== id) return;
	modal.current = null;
	c.resolve(ok);
	queueMicrotask(next);
}
