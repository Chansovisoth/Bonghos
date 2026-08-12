"use strict";

// Native local Web UI server with an opt-in notification relay. Provider
// tokens remain in the ignored .env.development file and are never returned
// to browser JavaScript. This helper is development-only and binds to loopback.

const fs = require("node:fs");
const http = require("node:http");
const path = require("node:path");

const REPO_ROOT = path.resolve(__dirname, "..");
const STATIC_ROOT = path.join(REPO_ROOT, "source", "web", "src");
const DEFAULT_ENV_PATH = path.join(REPO_ROOT, ".env.development");
const RELAY_HEADER = "x-bonghos-dev-relay";

function parseEnv(text) {
  const values = {};
  for (const rawLine of String(text).split(/\r?\n/)) {
    let line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    if (line.startsWith("export ")) line = line.slice(7).trim();
    const separator = line.indexOf("=");
    if (separator < 1) continue;
    const key = line.slice(0, separator).trim();
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) continue;
    let value = line.slice(separator + 1).trim();
    if (value.length >= 2 && ((value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'")))) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }
  return values;
}

function loadDevelopmentEnv(envPath = DEFAULT_ENV_PATH) {
  let fileValues = {};
  try {
    fileValues = parseEnv(fs.readFileSync(envPath, "utf8"));
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
  return { ...fileValues, ...process.env };
}

function configuredBots(env) {
  const definitions = [
    {
      id: 1,
      provider: "telegram",
      name: env.BONGHOS_DEV_TELEGRAM_NAME || "Telegram development",
      token: String(env.BONGHOS_DEV_TELEGRAM_TOKEN || "").trim(),
      destination: String(env.BONGHOS_DEV_TELEGRAM_CHAT_ID || "").trim(),
    },
    {
      id: 2,
      provider: "discord",
      name: env.BONGHOS_DEV_DISCORD_NAME || "Discord development",
      token: String(env.BONGHOS_DEV_DISCORD_TOKEN || "").trim(),
      destination: String(env.BONGHOS_DEV_DISCORD_CHANNEL_ID || "").trim(),
    },
  ];

  const configured = [];
  for (const definition of definitions) {
    if (!definition.token && !definition.destination) continue;
	if (!definition.token || (definition.provider === "discord" && !definition.destination)) {
      throw new Error(`${definition.provider} requires both its token and destination ID`);
    }
    if (/\s/.test(definition.token)) {
      throw new Error(`${definition.provider} token contains whitespace`);
    }
    // Telegram destinations must be established by an administrator using
    // /bonghos here. Never seed the flow from a legacy environment chat ID.
    const destinationIDs = definition.provider === "telegram" ? [] : [definition.destination];
    if (new Set(destinationIDs).size !== destinationIDs.length) {
      throw new Error(`${definition.provider} destination IDs must be unique`);
    }
    for (const destinationID of destinationIDs) {
      if (definition.provider === "telegram" && !/^(?:-?\d+|@[A-Za-z0-9_]{5,})$/.test(destinationID)) {
        throw new Error("Telegram chat IDs must be numeric or public @channel usernames");
      }
      if (definition.provider === "discord" && !/^\d{10,25}$/.test(destinationID)) {
        throw new Error("Discord channel ID must be numeric");
      }
    }
    const destinations = destinationIDs.map((destinationID) => ({
      id: destinationID, name: destinationID, type: definition.provider === "telegram" ? "group" : "channel",
    }));
    configured.push({
      ...definition,
      destination: destinationIDs[0] || "",
      destinations,
      public: {
        id: definition.id,
        name: definition.name,
        provider: definition.provider,
        destination_id: destinationIDs[0] || "",
        destinations: destinations.map((destination) => ({ ...destination })),
        enabled: true,
        notify_server_started: true,
        notify_server_stopped: true,
        notify_player_joined: true,
        notify_player_left: true,
        token_configured: true,
        development_env: true,
      },
    });
  }
  return configured;
}

async function telegramAPI(bot, method, params = {}, fetchImpl = fetch) {
  const query = new URLSearchParams(params);
  let response;
  try {
    response = await fetchImpl(`https://api.telegram.org/bot${bot.token}/${method}${query.size ? `?${query}` : ""}`);
  } catch {
    throw new Error("telegram command request failed");
  }
  let payload = {};
  try { payload = JSON.parse(await response.text()); } catch { /* handled below */ }
  if (!response.ok || !payload.ok) throw new Error(`telegram command request returned HTTP ${response.status}`);
  return payload.result;
}

function telegramCommandName(text) {
  const fields = String(text || "").trim().split(/\s+/).filter(Boolean);
  const root = String(fields[0] || "").toLowerCase().replace(/^\//, "").split("@", 1)[0];
  if (root !== "bonghos") return "";
  const subcommand = String(fields[1] || "help").toLowerCase();
  return ["here", "disconnect", "where", "help"].includes(subcommand) ? subcommand : "help";
}

async function pollDevelopmentTelegramCommands(bot, fetchImpl = fetch) {
  const updates = await telegramAPI(bot, "getUpdates", {
    allowed_updates: JSON.stringify(["message"]), limit: "100", timeout: "0",
    offset: String((Number(bot.lastUpdateID) || 0) + 1),
  }, fetchImpl);
	if (!bot.commandInitialized) {
	  for (const update of Array.isArray(updates) ? updates : []) {
		bot.lastUpdateID = Math.max(Number(bot.lastUpdateID) || 0, Number(update.update_id) || 0);
	  }
	  if (updates.length) bot.commandEmptyPolls = 0;
	  else bot.commandEmptyPolls = (Number(bot.commandEmptyPolls) || 0) + 1;
	  // Drain and confirm the old backlog before accepting commands. Two empty
	  // polls prevent a delayed historical update from reconnecting a stale topic.
	  bot.commandInitialized = bot.commandEmptyPolls >= 2;
	  return;
	}
  for (const update of Array.isArray(updates) ? updates : []) {
    bot.lastUpdateID = Math.max(Number(bot.lastUpdateID) || 0, Number(update.update_id) || 0);
    const message = update.message;
    const command = telegramCommandName(message?.text);
    if (!command || !["group", "supergroup"].includes(message?.chat?.type)) continue;
    const chatID = String(message.chat.id);
    const threadID = Number(message.message_thread_id) > 1 ? Number(message.message_thread_id) : 0;
    let administrator = message.sender_chat?.id === message.chat.id;
    if (!administrator && Number(message.from?.id) > 0) {
      const member = await telegramAPI(bot, "getChatMember", { chat_id: chatID, user_id: String(message.from.id) }, fetchImpl);
      administrator = ["creator", "administrator"].includes(member?.status);
    }
    const reply = (text) => providerRequest(bot, text, fetchImpl, chatID, threadID);
    if (command === "help") {
      // await reply("Bonghos commands:\n`/bonghos here` : Send notifications to this topic\n`/bonghos where` : Check this group's destination\n`/bonghos disconnect`: Stop notifications to this group\n\nOnly group administrators can change destinations.");
      await reply("Bonghos commands:\n<code>/bonghos here</code> : Send notifications to this topic\n<code>/bonghos where</code> : Check this group's destination\n<code>/bonghos disconnect</code>: Stop notifications to this group\n\nOnly group administrators can change destinations.");
      continue;
    }
    if (!administrator) {
      await reply("Only a group administrator can configure Bonghos notifications.");
      continue;
    }
    const existingIndex = bot.destinations.findIndex((destination) => destination.id === chatID);
    if (command === "here") {
      if (existingIndex < 0 && bot.destinations.length >= 3) {
        await reply("Bonghos could not connect this destination: Telegram already has three connected groups");
        continue;
      }
      const destination = {
        id: chatID, name: String(message.chat.title || chatID), type: message.chat.type,
        forum: !!(message.chat.is_forum || threadID), thread_id: threadID,
        thread_name: threadID ? "Selected topic" : "General",
      };
      if (existingIndex >= 0) bot.destinations[existingIndex] = destination;
      else bot.destinations.push(destination);
      bot.destination = bot.destinations[0]?.id || "";
      bot.public.destination_id = bot.destination;
      bot.public.destinations = bot.destinations.map((value) => ({ ...value }));
      await reply("Bonghos notifications will be sent here.");
    } else if (command === "disconnect") {
      bot.destinations = bot.destinations.filter((destination) => destination.id !== chatID);
      bot.destination = bot.destinations[0]?.id || "";
      bot.public.destination_id = bot.destination;
      bot.public.destinations = bot.destinations.map((value) => ({ ...value }));
      await reply("Bonghos notifications are disconnected from this group.");
    } else {
      const destination = bot.destinations.find((value) => value.id === chatID);
      await reply(!destination
        ? "This group is not connected. Run /bonghos here in the topic that should receive notifications."
        : Number(destination.thread_id) === threadID
          ? "Bonghos notifications are configured for this topic."
          : "Bonghos notifications are configured for another topic in this group.");
    }
  }
}

function startDevelopmentTelegramCommands(bots, fetchImpl = fetch) {
  const telegram = bots.find((bot) => bot.provider === "telegram");
  if (!telegram) return () => {};
  let running = false;
  const poll = async () => {
    if (running) return;
    running = true;
    try { await pollDevelopmentTelegramCommands(telegram, fetchImpl); }
    catch (error) { console.warn(`Telegram command polling: ${error.message}`); }
    finally { running = false; }
  };
  void poll();
  const timer = setInterval(poll, 3000);
  return () => clearInterval(timer);
}

async function providerRequest(bot, message, fetchImpl = fetch, destination = bot.destination, threadID = 0) {
  let response;
  try {
    if (bot.provider === "telegram") {
      if (!/^[A-Za-z0-9:_-]+$/.test(bot.token)) throw new Error("Telegram token has an invalid format");
      response = await fetchImpl(`https://api.telegram.org/bot${bot.token}/sendMessage`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          chat_id: destination, text: message,
          ...(Number(threadID) > 0 ? { message_thread_id: Number(threadID) } : {}),
        }),
      });
    } else {
      response = await fetchImpl(`https://discord.com/api/v10/channels/${destination}/messages`, {
        method: "POST",
        headers: {
          "Authorization": `Bot ${bot.token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content: message, allowed_mentions: { parse: [] } }),
      });
    }
  } catch {
    throw new Error(`${bot.provider} request failed`);
  }

  const responseText = await response.text();
  if (!response.ok) {
    let detail = "";
    try {
      const payload = JSON.parse(responseText);
      detail = String(payload.description || payload.message || "").slice(0, 240);
    } catch {
      // Provider returned a non-JSON error. Do not reflect arbitrary content.
    }
    throw new Error(`${bot.provider} returned HTTP ${response.status}${detail ? `: ${detail}` : ""}`);
  }
}

function uniqueTelegramTopics(values) {
  const byID = new Map();
  for (const topic of Array.isArray(values) ? values : []) {
    const id = Number(topic?.id) || 0;
    if (id <= 1) continue;
    const name = String(topic.name || "").normalize("NFKC")
      .replace(/[\u200B-\u200D\u2060\uFEFF]/gu, "")
      .trim().replace(/\s+/gu, " ") || `Channel ${id}`;
    byID.set(id, { id, name });
  }
  const byName = new Map();
  for (const topic of byID.values()) {
    const key = topic.name.normalize("NFKC")
      .replace(/[\u200B-\u200D\u2060\uFEFF]/gu, "")
      .trim().replace(/\s+/gu, " ").toLocaleLowerCase();
    const current = byName.get(key);
    if (!current || topic.id > current.id) byName.set(key, topic);
  }
  return Array.from(byName.values()).sort((left, right) => left.name.localeCompare(right.name));
}

async function discoverTelegramGroups(bot, fetchImpl = fetch) {
  const request = async (method) => {
    let response;
    try {
      response = await fetchImpl(`https://api.telegram.org/bot${bot.token}/${method}`);
    } catch {
      throw new Error("telegram discovery request failed");
    }
    let payload = {};
    try { payload = JSON.parse(await response.text()); } catch { /* handled below */ }
    if (!response.ok || !payload.ok) {
      const detail = String(payload.description || "").slice(0, 200);
      throw new Error(`telegram returned HTTP ${response.status}${detail ? `: ${detail}` : ""}`);
    }
    return payload.result;
  };
  const me = await request("getMe");
  const updates = await request(`getUpdates?allowed_updates=${encodeURIComponent(JSON.stringify([
    "message", "edited_message", "channel_post", "edited_channel_post", "my_chat_member",
  ]))}&limit=100&timeout=0`);
  const byID = new Map();
  const topicsByChat = new Map();
  const add = (chat) => {
    if (!chat || !["group", "supergroup"].includes(chat.type)) return;
    const id = String(chat.id);
    const current = byID.get(id) || {};
    byID.set(id, {
      ...current, id, name: chat.title || (chat.username ? `@${chat.username}` : id),
      type: chat.type, forum: !!(current.forum || chat.is_forum),
    });
  };
  const addMessage = (message) => {
    if (!message) return;
    add(message.chat);
    if (!message.chat || !["group", "supergroup"].includes(message.chat.type)) return;
    let threadID = Number(message.message_thread_id) || 0;
    if (threadID <= 1 && message.forum_topic_created) threadID = Number(message.message_id) || 0;
    if (threadID <= 1 || (!message.is_topic_message && !message.forum_topic_created && !message.forum_topic_edited)) return;
    const chatID = String(message.chat.id);
    if (!topicsByChat.has(chatID)) topicsByChat.set(chatID, new Map());
    const topicMap = topicsByChat.get(chatID);
    const events = [
      message.forum_topic_created, message.forum_topic_edited,
      message.reply_to_message?.forum_topic_created, message.reply_to_message?.forum_topic_edited,
    ].filter(Boolean);
    const name = events.map((event) => String(event.name || "").trim()).filter(Boolean).at(-1)
      || topicMap.get(threadID)?.name || `Channel ${threadID}`;
    topicMap.set(threadID, { id: threadID, name });
    byID.get(chatID).forum = true;
  };
  for (const update of Array.isArray(updates) ? updates : []) {
    addMessage(update.message);
    addMessage(update.edited_message);
    addMessage(update.channel_post);
    addMessage(update.edited_channel_post);
    const membership = update.my_chat_member;
    if (["left", "kicked"].includes(membership?.new_chat_member?.status)) {
      byID.delete(String(membership.chat?.id));
    } else {
      add(membership?.chat);
    }
  }
  const groups = Array.from(byID.values()).map((group) => ({
    ...group,
    topics: uniqueTelegramTopics(Array.from(topicsByChat.get(group.id)?.values() || [])),
  })).sort((left, right) => left.name.localeCompare(right.name));
  for (const group of groups.slice(0, 20)) {
    try {
      const fullChat = await request(`getChat?chat_id=${encodeURIComponent(group.id)}`);
      group.forum = !!(group.forum || fullChat?.is_forum);
      const fileID = String(fullChat?.photo?.small_file_id || "").trim();
      if (!fileID) continue;
      group.photo_file_id = fileID;
      const file = await request(`getFile?file_id=${encodeURIComponent(fileID)}`);
      const filePath = String(file?.file_path || "").replace(/^\/+/, "");
      if (!filePath || filePath.split("/").some((part) => !part || part === "." || part === "..")) continue;
      const photoResponse = await fetchImpl(`https://api.telegram.org/file/bot${bot.token}/${filePath}`);
      if (!photoResponse.ok || typeof photoResponse.arrayBuffer !== "function") continue;
      const data = Buffer.from(await photoResponse.arrayBuffer());
      if (!data.length || data.length > 512 * 1024) continue;
      const declaredType = String(photoResponse.headers?.get?.("content-type") || "").split(";", 1)[0].toLowerCase();
      const contentType = ["image/jpeg", "image/png", "image/webp"].includes(declaredType)
        ? declaredType
        : (/\.png$/i.test(filePath) ? "image/png" : /\.webp$/i.test(filePath) ? "image/webp" : "image/jpeg");
      group.photo_data_url = `data:${contentType};base64,${data.toString("base64")}`;
    } catch {
      // A missing group photo must not prevent selecting that group.
    }
  }
  return { bot_username: me?.username || "", groups };
}

function jsonResponse(response, status, payload) {
  const body = JSON.stringify(payload);
  response.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Content-Length": Buffer.byteLength(body),
    "Cache-Control": "no-store",
    "X-Content-Type-Options": "nosniff",
  });
  response.end(body);
}

function readJSON(request, limit = 64 * 1024) {
  return new Promise((resolve, reject) => {
    let size = 0;
    const chunks = [];
    request.on("data", (chunk) => {
      size += chunk.length;
      if (size > limit) {
        reject(new Error("request body is too large"));
        request.destroy();
        return;
      }
      chunks.push(chunk);
    });
    request.on("end", () => {
      try {
        resolve(chunks.length ? JSON.parse(Buffer.concat(chunks).toString("utf8")) : {});
      } catch {
        reject(new Error("invalid JSON request"));
      }
    });
    request.on("error", reject);
  });
}

function contentType(filePath) {
  switch (path.extname(filePath).toLowerCase()) {
  case ".html": return "text/html; charset=utf-8";
  case ".js": return "text/javascript; charset=utf-8";
  case ".css": return "text/css; charset=utf-8";
  case ".png": return "image/png";
  case ".svg": return "image/svg+xml";
  case ".txt": return "text/plain; charset=utf-8";
  default: return "application/octet-stream";
  }
}

function createDevServer(options = {}) {
  const env = options.env || loadDevelopmentEnv(options.envPath);
  const bots = options.bots || configuredBots(env);
  const botByID = new Map(bots.map((bot) => [bot.id, bot]));
  const fetchImpl = options.fetchImpl || fetch;
  const staticRoot = options.staticRoot || STATIC_ROOT;

  return http.createServer(async (request, response) => {
    const requestURL = new URL(request.url, "http://127.0.0.1");
    if (requestURL.pathname.startsWith("/__dev/bots")) {
      if (request.headers[RELAY_HEADER] !== "1") {
        jsonResponse(response, 403, { error: "development relay header required" });
        return;
      }
      try {
        if (request.method === "GET" && requestURL.pathname === "/__dev/bots") {
          jsonResponse(response, 200, bots.map((bot) => ({ ...bot.public })));
          return;
        }
        const newDiscovery = requestURL.pathname === "/__dev/bots/telegram/discover";
        const existingDiscovery = requestURL.pathname.match(/^\/__dev\/bots\/(\d+)\/telegram\/discover$/);
        if (request.method === "POST" && (newDiscovery || existingDiscovery)) {
          await readJSON(request);
          const bot = existingDiscovery
            ? botByID.get(Number(existingDiscovery[1]))
            : bots.find((candidate) => candidate.provider === "telegram");
          if (!bot || bot.provider !== "telegram") throw new Error("Telegram development bot is not configured");
          const discovery = await discoverTelegramGroups(bot, fetchImpl);
          if (!bot.discoveredGroups) bot.discoveredGroups = new Map();
          for (const group of discovery.groups) {
            const previous = bot.discoveredGroups.get(group.id) || {};
            const topics = uniqueTelegramTopics([...(previous.topics || []), ...(group.topics || [])]);
            bot.discoveredGroups.set(group.id, {
              ...previous, ...group,
              photo_file_id: group.photo_file_id || previous.photo_file_id || "",
              photo_data_url: group.photo_data_url || previous.photo_data_url || "",
              forum: !!(group.forum || previous.forum || topics.length),
              topics,
            });
          }
          discovery.groups = Array.from(bot.discoveredGroups.values())
            .sort((left, right) => left.name.localeCompare(right.name));
          bot.public.destinations = bot.public.destinations.map((destination) => ({
            ...destination,
            ...(bot.discoveredGroups.get(destination.id) || {}),
          }));
          jsonResponse(response, 200, discovery);
          return;
        }
        const testMatch = requestURL.pathname.match(/^\/__dev\/bots\/(\d+)\/test$/);
        if (request.method === "POST" && testMatch) {
          const bot = botByID.get(Number(testMatch[1]));
          if (!bot) throw new Error("development bot is not configured");
          await readJSON(request);
          for (const destination of bot.destinations) {
            await providerRequest(bot, `Bonghos local development test\n${bot.name} is connected.`, fetchImpl, destination.id, destination.thread_id);
          }
          jsonResponse(response, 200, { ok: true });
          return;
        }
        const patchMatch = requestURL.pathname.match(/^\/__dev\/bots\/(\d+)$/);
        if (request.method === "PATCH" && patchMatch) {
          const bot = botByID.get(Number(patchMatch[1]));
          if (!bot) throw new Error("development bot is not configured");
          const patch = await readJSON(request);
          for (const field of ["enabled", "notify_server_started", "notify_server_stopped", "notify_player_joined", "notify_player_left"]) {
            if (typeof patch[field] === "boolean") bot.public[field] = patch[field];
          }
          if (typeof patch.name === "string" && patch.name.trim()) {
            bot.name = patch.name.trim().slice(0, 80);
            bot.public.name = bot.name;
          }
          if (Array.isArray(patch.destinations)) {
            if (bot.provider !== "telegram" || patch.destinations.length < 1 || patch.destinations.length > 3) {
              throw new Error("Telegram requires between 1 and 3 destination groups");
            }
            const destinations = patch.destinations.map((destination) => {
              const id = String(destination.id || "").trim();
              const discovered = bot.discoveredGroups?.get(id) || {};
              return {
                id,
                name: String(destination.name || discovered.name || destination.id || "").trim(),
                type: String(destination.type || discovered.type || "group").trim(),
                photo_file_id: String(destination.photo_file_id || discovered.photo_file_id || "").trim(),
                photo_data_url: String(discovered.photo_data_url || ""),
                forum: !!(destination.forum || discovered.forum),
                thread_id: Number(destination.thread_id) || 0,
                thread_name: String(destination.thread_name || "").trim(),
                topics: Array.isArray(discovered.topics) ? discovered.topics.map((topic) => ({ ...topic })) : [],
              };
            });
            if (destinations.some((destination) => !/^(?:-?\d+|@[A-Za-z0-9_]{5,})$/.test(destination.id))) {
              throw new Error("Telegram chat IDs must be numeric or public @channel usernames");
            }
            if (new Set(destinations.map((destination) => destination.id)).size !== destinations.length) {
              throw new Error("Telegram destination groups must be unique");
            }
            bot.destinations = destinations;
            bot.destination = destinations[0].id;
            bot.public.destinations = destinations.map((destination) => ({ ...destination }));
            bot.public.destination_id = destinations[0].id;
          }
          jsonResponse(response, 200, { ...bot.public });
          return;
        }
        jsonResponse(response, 405, { error: "edit .env.development and restart the development relay" });
      } catch (error) {
        jsonResponse(response, 400, { error: error.message || "development relay request failed" });
      }
      return;
    }

    if (request.method !== "GET" && request.method !== "HEAD") {
      response.writeHead(405, { "Content-Type": "text/plain; charset=utf-8" });
      response.end("Method not allowed");
      return;
    }
    let relativePath;
    try {
      relativePath = decodeURIComponent(requestURL.pathname);
    } catch {
      response.writeHead(400, { "Content-Type": "text/plain; charset=utf-8" });
      response.end("Invalid path");
      return;
    }
    if (relativePath === "/") relativePath = "/index.html";
    const filePath = path.resolve(staticRoot, "." + relativePath);
    const rootPrefix = path.resolve(staticRoot) + path.sep;
    if (!filePath.startsWith(rootPrefix)) {
      response.writeHead(404);
      response.end();
      return;
    }
    fs.readFile(filePath, (error, data) => {
      if (error) {
        response.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
        response.end("Not found");
        return;
      }
      response.writeHead(200, {
        "Content-Type": contentType(filePath),
        "Content-Length": data.length,
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      });
      if (request.method === "HEAD") response.end();
      else response.end(data);
    });
  });
}

if (require.main === module) {
  try {
    const env = loadDevelopmentEnv();
    const port = Number.parseInt(env.BONGHOS_DEV_PORT || "8000", 10);
    if (!Number.isInteger(port) || port < 1024 || port > 65535) {
      throw new Error("BONGHOS_DEV_PORT must be between 1024 and 65535");
    }
    const bots = configuredBots(env);
    const server = createDevServer({ env, bots });
    let stopTelegramCommands = () => {};
    server.on("close", () => stopTelegramCommands());
    server.on("error", (error) => {
	  stopTelegramCommands();
      if (error.code === "EADDRINUSE") {
        console.error(`Development server failed: port ${port} is already in use. Stop the existing server or set BONGHOS_DEV_PORT to another port.`);
      } else {
        console.error(`Development server failed: ${error.message}`);
      }
      process.exitCode = 1;
    });
    server.listen(port, "127.0.0.1", () => {
	  stopTelegramCommands = startDevelopmentTelegramCommands(bots);
      const providers = bots.map((bot) => bot.provider).join(" and ") || "no providers";
      console.log(`Bonghos development Web UI: http://127.0.0.1:${port}/?demo&debug-bots`);
      console.log(`Configured from .env.development: ${providers}`);
      if (!bots.length) console.log("Add one Telegram and/or one Discord credential pair, then restart this command.");
    });
  } catch (error) {
    console.error(`Development server failed: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = { parseEnv, configuredBots, providerRequest, discoverTelegramGroups, pollDevelopmentTelegramCommands, createDevServer };
