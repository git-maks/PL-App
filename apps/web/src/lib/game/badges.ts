import type { IconName } from '#lib/icons/paths';

export const BADGES: Record<string, { title: string; description: string; icon: IconName }> = {
	'first-scan': { title: 'Pierwszy krok', description: 'Zeskanuj pierwszą stację.', icon: 'flag' },
	'faculties-all': { title: 'Odkrywca wydziałów', description: 'Odwiedź stacje wszystkich 6 wydziałów.', icon: 'apartment' },
	bookworm: { title: 'Mól książkowy', description: 'Odwiedź Bibliotekę w CIZT.', icon: 'menu_book' },
	'team-spirit': { title: 'Duch drużyny', description: 'Zalicz 3 zadania drużynowe.', icon: 'groups' },
	'early-bird': { title: 'Ranny ptaszek', description: 'Zeskanuj stację przed 11:00.', icon: 'schedule' },
	'card-unlocked': { title: 'Karta postaci', description: 'Odblokuj swoją kartę postaci.', icon: 'workspace_premium' }
};
