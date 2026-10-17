// Applies the saved theme and motion preference before first paint (no flash). Kept as a file so the
// Content-Security-Policy does not need to allow arbitrary inline scripts.
try {
	var t = localStorage.getItem('uq.theme');
	if (t === 'light' || t === 'dark') document.documentElement.dataset.theme = t;
	if (localStorage.getItem('uq.motion') === 'reduced') document.documentElement.dataset.motion = 'reduced';
} catch (e) {}
