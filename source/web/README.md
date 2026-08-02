# Bonghos Web UI

The frontend is an intentionally dependency-free vanilla JavaScript single-page
application: no framework, no bundler, no npm install. This keeps the single
binary reproducible offline and eliminates the entire JS supply chain.

Source files live in `web/src/`. "Building" the frontend is a copy:

    cp web/src/* cmd/bonghos/webdist/

The Go binary embeds `cmd/bonghos/webdist` via go:embed.
