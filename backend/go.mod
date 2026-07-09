module github.com/MattCheramie/echoboard

go 1.25.0

// Pin the toolchain to a patched Go release. go 1.25.0 shipped with stdlib
// CVEs (crypto/tls, crypto/x509, net/url, net/http, …) that govulncheck flags;
// 1.25.12 is the first patch clearing every one of them. actions/setup-go reads
// this directive via go-version-file, so CI, the release build, and local
// builds all use the patched toolchain. Bump as new stdlib CVEs are disclosed.
toolchain go1.25.12

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.10.0
	golang.org/x/crypto v0.53.0
	golang.org/x/term v0.44.0
	modernc.org/sqlite v1.53.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	modernc.org/libc v1.73.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
