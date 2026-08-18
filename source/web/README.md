# Bonghos Web UI

The frontend is an intentionally dependency-free vanilla JavaScript single-page
application: no framework, no bundler, no npm install. This keeps the single
binary reproducible offline and keeps the default UI free of runtime frontend
dependencies. When an Owner explicitly enables Cloudflare Turnstile, the login
page alone loads Cloudflare's challenge script from `challenges.cloudflare.com`.

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

`.env.development` is ignored by Git. Configure the Telegram and/or Discord bot
token, then run:

```powershell
node scripts/dev-web.js
```

Open:

```text
http://127.0.0.1:8000/?demo&debug-bots
```

The Settings page shows sanitized provider entries sourced from the environment
file. For Telegram, add the bot to a group and have a group administrator run
`/bonghos here` inside the topic that should receive broadcasts. For Discord, invite the bot
with the `bot` and `applications.commands` scopes and allow it to view the target
channel and send messages there (plus send messages in threads when applicable).
The relay registers an instant guild command, and a server administrator runs
`/bonghos here` in the target channel. Repeat this in up to three destinations. Use `/bonghos where`
to check one, `/bonghos disconnect` to remove it, and `/bonghos help` to list commands. **Send
test** sends to every connected destination. **Invite Bot** opens Telegram's
group picker or Discord's server installation screen with only the required
permissions. The edit dialog refreshes connected destinations automatically.
It also lists groups and servers detected from provider membership events as
**Not configured** until `/bonghos here` selects their broadcast destination.
Discord backfills every server when the relay connects. Telegram has no API to
enumerate historical memberships: a newly added bot is detected immediately,
while a bot that was already in a group before the relay started may need one
visible group interaction such as `/bonghos help` to be detected without
configuring a destination.
Enable, destination, and notification toggles are temporary
relay-process debugging state; edit credentials in `.env.development` and
restart the relay to restore its configured values.

Never put real tokens in `.env.development.example`, frontend source, demo
fixtures, or a URL. The normal `?demo` mode remains fully mocked and never
sends messages.
