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

## Local notification debugging without WSL

The repository includes a dependency-free Node development relay for testing
one Telegram bot and one Discord bot from Windows. Provider tokens remain on
the Node side and are never returned to browser JavaScript.

From the repository root, copy the tracked placeholder file and fill in only
the providers you want to test:

```powershell
Copy-Item .env.development.example .env.development
```

`.env.development` is ignored by Git. Configure the Telegram token and/or the
Discord bot token and numeric channel ID, then run:

```powershell
node scripts/dev-web.js
```

Open:

```text
http://127.0.0.1:8000/?demo&debug-bots
```

The Settings page shows sanitized provider entries sourced from the environment
file. For Telegram, add the bot to a group as an administrator and run `/bonghos here`
inside the topic that should receive broadcasts. Repeat this in up to three
groups. Use `/bonghos where` to check a group, `/bonghos disconnect` to remove
it, and `/bonghos help` to list commands. **Send
test** sends to every connected destination. Enable, destination, and notification toggles are temporary
relay-process debugging state; edit credentials in `.env.development` and
restart the relay to restore its configured values.

Never put real tokens in `.env.development.example`, frontend source, demo
fixtures, or a URL. The normal `?demo` mode remains fully mocked and never
sends messages.
