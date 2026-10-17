import type { IconName } from '#lib/icons/paths';

/** Team emblems. Keep in sync with allowedEmblems in apps/api/internal/app/teams.go. */
export const EMBLEMS: { icon: IconName; label: string }[] = [
	{ icon: 'shield', label: 'Tarcza' },
	{ icon: 'star', label: 'Gwiazda' },
	{ icon: 'rocket_launch', label: 'Rakieta' },
	{ icon: 'bolt', label: 'Błyskawica' },
	{ icon: 'science', label: 'Laboratorium' },
	{ icon: 'psychology', label: 'Umysł' },
	{ icon: 'memory', label: 'Procesor' },
	{ icon: 'smart_toy', label: 'Robot' },
	{ icon: 'code', label: 'Kod' },
	{ icon: 'satellite_alt', label: 'Satelita' },
	{ icon: 'lightbulb', label: 'Żarówka' },
	{ icon: 'eco', label: 'Liść' },
	{ icon: 'forest', label: 'Las' },
	{ icon: 'public', label: 'Glob' },
	{ icon: 'pets', label: 'Łapa' },
	{ icon: 'owl', label: 'Sowa' },
	{ icon: 'palette', label: 'Paleta' },
	{ icon: 'music_note', label: 'Nuta' },
	{ icon: 'sports_soccer', label: 'Piłka' },
	{ icon: 'sports_esports', label: 'Pad' },
	{ icon: 'casino', label: 'Kość' },
	{ icon: 'chess_knight', label: 'Skoczek' },
	{ icon: 'crown', label: 'Korona' },
	{ icon: 'military_tech', label: 'Medal' }
];
