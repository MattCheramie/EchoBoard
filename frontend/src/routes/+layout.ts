// EchoBoard is a client-side single-page app: rendering happens in the browser
// against the Go REST API, and adapter-static emits an index.html fallback that
// the backend serves for every non-API path. Disabling SSR/prerender keeps auth
// state and data-fetching entirely on the client.
export const ssr = false;
export const prerender = false;
