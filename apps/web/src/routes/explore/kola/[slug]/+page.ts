import { error } from '@sveltejs/kit';
import { clubs, clubBySlug } from '#lib/content';

export const entries = () => clubs.map((c) => ({ slug: c.slug }));

export function load({ params }) {
	const club = clubBySlug[params.slug];
	if (!club) error(404, 'Nie ma takiego koła');
	return { club };
}
