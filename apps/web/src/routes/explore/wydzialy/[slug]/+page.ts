import { error } from '@sveltejs/kit';
import { faculties, facultyBySlug } from '#lib/content';

export const entries = () => faculties.map((f) => ({ slug: f.slug }));

export function load({ params }) {
	const faculty = facultyBySlug[params.slug];
	if (!faculty) error(404, 'Nie ma takiego wydziału');
	return { faculty };
}
