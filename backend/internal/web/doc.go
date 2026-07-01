// Package web exposes the compiled SvelteKit frontend to the Go server so
// EchoBoard can ship as a single binary (see README "single executable" goal).
//
// The real assets are embedded only when the binary is built with the `embed`
// build tag (Tier 6, PR 6.1), after the frontend has been compiled into
// backend/internal/web/build. Without that tag the default build compiles
// cleanly with an empty asset set, which keeps Tier-0 CI green while no
// frontend build output exists yet.
package web
