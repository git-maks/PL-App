import { resolve } from '$app/paths';

/** App base path ('' or e.g. '/PL-App' on GitHub Pages), without a trailing slash. */
export const BASE: string = (resolve as (p: string) => string)('/').replace(/\/$/, '');

/** Prefix an app-absolute path ("/hero") with the base path. */
export function href(path: string): string {
	return BASE + path;
}

/** Strip the base path from a pathname, for matching the current route. */
export function routePath(pathname: string): string {
	return BASE && pathname.startsWith(BASE) ? pathname.slice(BASE.length) || '/' : pathname;
}
