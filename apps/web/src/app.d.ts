declare global {
	/** 'http' (Go backend) or 'demo' (in-browser backend for GitHub Pages). Injected by Vite. */
	const __API_MODE__: 'http' | 'demo';
	namespace App {}
}
export {};
