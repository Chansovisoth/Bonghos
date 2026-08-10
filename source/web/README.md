# Bonghos Web UI

The frontend is an intentionally dependency-free vanilla JavaScript single-page
application: no framework, no bundler, no npm install. This keeps the single
binary reproducible offline and eliminates the entire JS supply chain.

Source files live in `web/src/`. "Building" the frontend is a copy:

    cp web/src/* cmd/bonghos/webdist/

The Go binary embeds `cmd/bonghos/webdist` via go:embed.

## Local visual demo

You can preview Web UI changes on a laptop without running Bonghos or a
Minecraft server. Serve the static frontend directory and opt into mock data:

```bash
cd source/web/src
python3 -m http.server 8000
```

Then open:

```text
http://127.0.0.1:8000/?demo
```

`?demo` uses an in-browser mock API and sample server data. It is only for
visual review of the dependency-free Web UI. Production builds and real
installs keep using the Go REST API and WebSocket API.
