import { error } from '@sveltejs/kit';
import { programs, programBySlug } from '#lib/content';

export const entries = () => programs.map((p) => ({ slug: p.slug }));

export function load({ params }) {
	const program = programBySlug[params.slug];
	if (!program) error(404, 'Nie ma takiego kierunku');
	return { program };
}
