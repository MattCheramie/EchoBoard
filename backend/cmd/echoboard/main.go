// Command echoboard is the EchoBoard server and CLI entry point.
//
// Tier 0: this is a skeleton. It wires nothing yet. Real startup (config load,
// database connection, --setup admin bootstrap, HTTP + WebSocket server) lands
// in Tier 1. See ROADMAP.md.
package main

import (
	"fmt"
	"os"
)

// version is stamped at build time in Tier 6; a placeholder for now.
const version = "0.0.0-dev"

func main() {
	// TODO(tier1): parse flags (--setup), load config, open the database,
	// run migrations, and start the API + WebSocket server.
	fmt.Printf("EchoBoard %s — skeleton build. See ROADMAP.md for what's next.\n", version)
	os.Exit(0)
}
