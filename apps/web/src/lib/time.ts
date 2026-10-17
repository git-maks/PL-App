const fmtTime = new Intl.DateTimeFormat('pl-PL', { hour: '2-digit', minute: '2-digit' });
const fmtDate = new Intl.DateTimeFormat('pl-PL', { day: 'numeric', month: 'long', year: 'numeric' });
const fmtRel = new Intl.RelativeTimeFormat('pl-PL', { numeric: 'auto' });

export function time(iso: string): string {
	return fmtTime.format(new Date(iso));
}
export function date(iso: string | Date): string {
	return fmtDate.format(typeof iso === 'string' ? new Date(iso) : iso);
}
export function relative(iso: string, now = Date.now()): string {
	const diff = (new Date(iso).getTime() - now) / 1000;
	const abs = Math.abs(diff);
	if (abs < 60) return fmtRel.format(Math.round(diff), 'second');
	if (abs < 3600) return fmtRel.format(Math.round(diff / 60), 'minute');
	if (abs < 86400) return fmtRel.format(Math.round(diff / 3600), 'hour');
	return fmtRel.format(Math.round(diff / 86400), 'day');
}
