// Every route is prerendered as a static shell; game screens render on the client (ssr = false),
// knowledge-base screens under /explore opt back into SSR so their content is in the HTML.
export const prerender = true;
export const ssr = false;
export const trailingSlash = 'never';
