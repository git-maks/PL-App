import { describe, expect, it } from 'vitest';
import { answer, ask, modal } from './modal.svelte';

describe('modal queue', () => {
	it('shows one request at a time and resolves each with its own answer', async () => {
		const first = ask({ title: 'A', body: '', confirm: 'OK', cancel: 'Nie' });
		const second = ask({ title: 'B', body: '', confirm: 'OK' });
		const a = modal.current!;
		expect(a.title).toBe('A');
		answer(a.id, true);
		answer(a.id, false); // a late close event for the same modal is ignored
		await Promise.resolve();
		const b = modal.current!;
		expect(b.title).toBe('B');
		answer(b.id, false);
		expect(await first).toBe(true);
		expect(await second).toBe(false);
		expect(modal.current).toBeNull();
	});
});
