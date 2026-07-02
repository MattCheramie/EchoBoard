// EchoBoard runs as a single-page app: the Go backend serves the API and, in
// production, the embedded static build. Disable SSR and prerendering so
// everything renders client-side against the API.
export const ssr = false;
export const prerender = false;
